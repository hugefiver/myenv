package ui

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	"mihomotui/internal/termtext"
)

type connsSnapMsg struct {
	generation requestID
	snap       api.ConnectionsSnapshot
	err        error
	end        bool
}

type connRow struct {
	id       string
	target   string
	network  string
	rule     string
	chain    string
	up       int64
	down     int64
	upRate   int64
	downRate int64
}

type connsModel struct {
	ctx        context.Context
	cli        *api.Client
	bus        *messageBus
	rows       []connRow
	prev       map[string]connRow
	prevTime   time.Time
	cancel     context.CancelFunc
	generation requestID
	send       func(tea.Msg)
	errText    string
	cursor     int
	scroll     int
	follow     bool
	upTotal    int64
	dnTotal    int64
	width      int
	height     int
}

func newConnsModel(ctx context.Context, cli *api.Client, bus *messageBus) *connsModel {
	return &connsModel{ctx: ctx, cli: cli, bus: bus, prev: map[string]connRow{}, follow: true}
}

func (m *connsModel) start(send func(tea.Msg)) tea.Cmd {
	ctx, generation := beginStream(m.ctx, m.bus, streamConnections, &m.generation, &m.cancel)
	m.send = send
	m.errText = ""
	cli := m.cli
	go func() {
		ch := make(chan api.ConnectionsSnapshot, 8)
		errCh := make(chan error, 1)
		go func() { errCh <- cli.StreamConnections(ctx, ch) }()
		for {
			select {
			case <-ctx.Done():
				return
			case s := <-ch:
				send(connsSnapMsg{generation: generation, snap: s})
			case err := <-errCh:
				if ctx.Err() == nil && !streamCanceled(err) {
					send(connsSnapMsg{generation: generation, err: err, end: true})
				}
				return
			}
		}
	}()
	return nil
}

func (m *connsModel) stop() {
	stopStream(&m.generation, &m.cancel)
}

func (m *connsModel) ingest(s api.ConnectionsSnapshot) {
	now := time.Now()
	dt := now.Sub(m.prevTime).Seconds()
	if m.prevTime.IsZero() || dt <= 0 {
		dt = 1
	}
	m.upTotal = s.UploadTotal
	m.dnTotal = s.DownloadTotal
	rows := make([]connRow, 0, len(s.Connections))
	next := make(map[string]connRow, len(s.Connections))
	for _, c := range s.Connections {
		host := termtext.SingleLine(c.Metadata.Host)
		if host == "" {
			host = termtext.SingleLine(c.Metadata.DestinationIP)
		}
		target := host + ":" + termtext.SingleLine(c.Metadata.DestinationPort)
		chain := termtext.SingleLine(strings.Join(c.Chains, "→"))
		r := connRow{
			id: c.ID, target: target, network: termtext.SingleLine(c.Metadata.Network),
			rule: termtext.SingleLine(c.Rule), chain: chain, up: c.Upload, down: c.Download,
		}
		if p, ok := m.prev[c.ID]; ok {
			if c.Upload >= p.up {
				r.upRate = int64(float64(c.Upload-p.up) / dt)
			}
			if c.Download >= p.down {
				r.downRate = int64(float64(c.Download-p.down) / dt)
			}
		}
		rows = append(rows, r)
		next[c.ID] = r
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].downRate+rows[i].upRate > rows[j].downRate+rows[j].upRate
	})
	m.rows = rows
	m.prev = next
	m.prevTime = now
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

func (m *connsModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case connsSnapMsg:
		if v.generation != m.generation {
			return nil, "", false
		}
		if v.err != nil {
			if streamCanceled(v.err) {
				return nil, "", false
			}
			m.errText = termtext.SingleLine(v.err.Error())
			return nil, "connections: " + m.errText, true
		}
		if v.end {
			m.errText = "stream ended unexpectedly"
			return nil, "connections: " + m.errText, true
		}
		m.errText = ""
		m.ingest(v.snap)
		if m.follow && len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}
	case tea.KeyMsg:
		switch v.String() {
		case "r":
			return m.start(m.send), "connections retrying", false
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.follow = false
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
			if m.cursor >= len(m.rows)-1 {
				m.follow = true
			}
		case "g", "home":
			m.cursor = 0
			m.follow = false
		case "G", "end":
			if len(m.rows) > 0 {
				m.cursor = len(m.rows) - 1
			}
			m.follow = true
		}
	}
	return nil, "", false
}

func (m *connsModel) View() string {
	var b strings.Builder
	follow := "off"
	if m.follow {
		follow = "on"
	}
	title := []textSegment{
		segment(stTitle, "Connections"),
		segment(stMuted, "  total ↑"+humanBytes(m.upTotal)+" ↓"+humanBytes(m.dnTotal)+"  follow="+follow),
		segment(stMuted, "  r=retry"),
	}
	if m.errText != "" {
		title = append(title, segment(stErr, "  error="+m.errText))
	}
	b.WriteString(renderTextLine(m.width, title...))
	b.WriteString("\n\n")
	header := termtext.PadRight("TARGET", 38) + termtext.PadRight("NET", 5) + termtext.PadRight("RULE", 12) + termtext.PadRight("CHAIN", 24) + termtext.PadRight("UP/DN", 16) + "RATE"
	b.WriteString(renderTextLine(m.width, segment(stMuted, header)) + "\n")
	view := m.height - 4
	if view <= 0 {
		view = 1
	}
	m.scroll = clampScroll(m.cursor, m.scroll, view, len(m.rows))
	end := m.scroll + view
	if end > len(m.rows) {
		end = len(m.rows)
	}
	for i := m.scroll; i < end; i++ {
		r := m.rows[i]
		marker := "  "
		markerStyle := stPlain
		if i == m.cursor {
			marker, markerStyle = "▸ ", stMark
		}
		b.WriteString(renderTextLine(m.width,
			segment(markerStyle, marker),
			segment(stPlain, termtext.PadRight(r.target, 37)),
			segment(stPlain, termtext.PadRight(r.network, 5)),
			segment(stPlain, termtext.PadRight(r.rule, 12)),
			segment(stPlain, termtext.PadRight(r.chain, 24)),
			segment(stPlain, termtext.PadRight("↑"+humanBytes(r.up)+" ↓"+humanBytes(r.down), 16)),
			segment(stPlain, "↑"+humanBytes(r.upRate)+"/s ↓"+humanBytes(r.downRate)+"/s"),
		) + "\n")
	}
	if len(m.rows) == 0 {
		b.WriteString(stMuted.Render("no connections"))
	}
	return b.String()
}
