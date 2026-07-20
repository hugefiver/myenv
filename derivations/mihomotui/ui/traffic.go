package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	"mihomotui/internal/termtext"
)

type trafficMsg struct {
	generation requestID
	t          api.Traffic
	err        error
	end        bool
}

type trafficModel struct {
	ctx        context.Context
	cli        *api.Client
	bus        *messageBus
	cur        api.Traffic
	totalUp    int64
	totalDn    int64
	maxUp      int64
	maxDn      int64
	cancel     context.CancelFunc
	generation requestID
	send       func(tea.Msg)
	errText    string
	width      int
	height     int
}

func newTrafficModel(ctx context.Context, cli *api.Client, bus *messageBus) *trafficModel {
	return &trafficModel{ctx: ctx, cli: cli, bus: bus}
}

func (m *trafficModel) start(send func(tea.Msg)) tea.Cmd {
	ctx, generation := beginStream(m.ctx, m.bus, streamTraffic, &m.generation, &m.cancel)
	m.send = send
	m.errText = ""
	cli := m.cli
	go func() {
		ch := make(chan api.Traffic, 16)
		errCh := make(chan error, 1)
		go func() { errCh <- cli.StreamTraffic(ctx, ch) }()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ch:
				send(trafficMsg{generation: generation, t: t})
			case err := <-errCh:
				if ctx.Err() == nil && !streamCanceled(err) {
					send(trafficMsg{generation: generation, err: err, end: true})
				}
				return
			}
		}
	}()
	return nil
}

func (m *trafficModel) stop() {
	stopStream(&m.generation, &m.cancel)
}

func (m *trafficModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case trafficMsg:
		if v.generation != m.generation {
			return nil, "", false
		}
		if v.err != nil {
			if streamCanceled(v.err) {
				return nil, "", false
			}
			m.errText = termtext.SingleLine(v.err.Error())
			return nil, "traffic: " + m.errText, true
		}
		if v.end {
			m.errText = "stream ended unexpectedly"
			return nil, "traffic: " + m.errText, true
		}
		m.errText = ""
		m.cur = v.t
		m.totalUp = v.t.UpTotal
		m.totalDn = v.t.DownTotal
		if v.t.Up > m.maxUp {
			m.maxUp = v.t.Up
		}
		if v.t.Down > m.maxDn {
			m.maxDn = v.t.Down
		}
	case tea.KeyMsg:
		if v.String() == "r" {
			return m.start(m.send), "traffic retrying", false
		}
	}
	return nil, "", false
}

func (m *trafficModel) View() string {
	var b strings.Builder
	writeLine := func(parts ...textSegment) {
		b.WriteString(renderTextLine(m.width, parts...) + "\n")
	}
	header := []textSegment{segment(stTitle, "Traffic"), segment(stMuted, "  r=retry")}
	if m.errText != "" {
		header = append(header, segment(stErr, "  error="+m.errText))
	}
	writeLine(header...)
	b.WriteString("\n")
	up, upRest := bar(m.cur.Up, m.maxUp, 40)
	writeLine(segment(stBase, "Upload   "), segment(stOK, humanBytes(m.cur.Up)+"/s"))
	writeLine(segment(stBase, "         "), segment(stOK, up), segment(stMuted, upRest))
	b.WriteString("\n")
	down, downRest := bar(m.cur.Down, m.maxDn, 40)
	writeLine(segment(stBase, "Download "), segment(stOK, humanBytes(m.cur.Down)+"/s"))
	writeLine(segment(stBase, "         "), segment(stOK, down), segment(stMuted, downRest))
	b.WriteString("\n")
	writeLine(segment(stMuted, "peak ↑"+humanBytes(m.maxUp)+"/s  ↓"+humanBytes(m.maxDn)+"/s"))
	b.WriteString(renderTextLine(m.width, segment(stMuted, "session ↑"+humanBytes(m.totalUp)+"  ↓"+humanBytes(m.totalDn))))
	return b.String()
}

func bar(v, max int64, w int) (string, string) {
	if max <= 0 {
		return "", strings.Repeat("·", w)
	}
	n := int(float64(v) / float64(max) * float64(w))
	if n > w {
		n = w
	}
	return strings.Repeat("█", n), strings.Repeat("·", w-n)
}
