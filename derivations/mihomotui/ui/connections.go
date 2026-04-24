package ui

import (
	"context"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

type connsTickMsg struct{}

type connsSnapMsg struct {
	snap api.ConnectionsSnapshot
	err  error
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
	cli      *api.Client
	rows     []connRow
	prev     map[string]connRow
	prevTime time.Time
	cancel   context.CancelFunc
	scroll   int
	upTotal  int64
	dnTotal  int64
	width    int
	height   int
}

func newConnsModel(cli *api.Client) *connsModel {
	return &connsModel{cli: cli, prev: map[string]connRow{}}
}

func (m *connsModel) start(send func(tea.Msg)) tea.Cmd {
	m.stop()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
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
				send(connsSnapMsg{snap: s})
			case err := <-errCh:
				if err != nil && ctx.Err() == nil {
					send(connsSnapMsg{err: err})
				}
				return
			}
		}
	}()
	return nil
}

func (m *connsModel) stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
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
		host := c.Metadata.Host
		if host == "" {
			host = c.Metadata.DestinationIP
		}
		target := host + ":" + c.Metadata.DestinationPort
		chain := strings.Join(c.Chains, "→")
		r := connRow{
			id: c.ID, target: target, network: c.Metadata.Network,
			rule: c.Rule, chain: chain, up: c.Upload, down: c.Download,
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
}

func (m *connsModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case connsSnapMsg:
		if v.err != nil {
			return nil, "conns: " + v.err.Error(), true
		}
		m.ingest(v.snap)
	case tea.KeyMsg:
		switch v.String() {
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			if m.scroll < m.maxScroll() {
				m.scroll++
			}
		case "g":
			m.scroll = 0
		}
	}
	return nil, "", false
}

func (m *connsModel) View() string {
	var b strings.Builder
	b.WriteString(stTitle.Render("Connections"))
	b.WriteString(stMuted.Render("  total ↑" + humanBytes(m.upTotal) + " ↓" + humanBytes(m.dnTotal)))
	b.WriteString("\n\n")
	header := padR("TARGET", 38) + padR("NET", 5) + padR("RULE", 12) + padR("CHAIN", 24) + padR("UP/DN", 16) + "RATE"
	b.WriteString(stMuted.Render(header) + "\n")
	max := m.height - 6
	if max <= 0 {
		max = 20
	}
	start := m.scroll
	if start > len(m.rows) {
		start = len(m.rows)
		m.scroll = start
	}
	end := start + max
	if end > len(m.rows) {
		end = len(m.rows)
	}
	for _, r := range m.rows[start:end] {
		line := padR(trunc(r.target, 37), 38) +
			padR(trunc(r.network, 4), 5) +
			padR(trunc(r.rule, 11), 12) +
			padR(trunc(r.chain, 23), 24) +
			padR("↑"+humanBytes(r.up)+" ↓"+humanBytes(r.down), 16) +
			"↑" + humanBytes(r.upRate) + "/s ↓" + humanBytes(r.downRate) + "/s"
		b.WriteString(line + "\n")
	}
	if len(m.rows) == 0 {
		b.WriteString(stMuted.Render("no connections"))
	}
	return b.String()
}

func (m *connsModel) maxScroll() int {
	v := m.height - 6
	if v < 1 {
		v = 1
	}
	n := len(m.rows) - v
	if n < 0 {
		return 0
	}
	return n
}

func padR(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

var _ = connsTickMsg{}
