package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

type trafficMsg struct {
	t   api.Traffic
	err error
}

type trafficModel struct {
	cli     *api.Client
	cur     api.Traffic
	totalUp int64
	totalDn int64
	maxUp   int64
	maxDn   int64
	cancel  context.CancelFunc
	width   int
	height  int
}

func newTrafficModel(cli *api.Client) *trafficModel {
	return &trafficModel{cli: cli}
}

func (m *trafficModel) start(send func(tea.Msg)) tea.Cmd {
	m.stop()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
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
				send(trafficMsg{t: t})
			case err := <-errCh:
				if err != nil && ctx.Err() == nil {
					send(trafficMsg{err: err})
				}
				return
			}
		}
	}()
	return nil
}

func (m *trafficModel) stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *trafficModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case trafficMsg:
		if v.err != nil {
			return nil, "traffic: " + v.err.Error(), true
		}
		m.cur = v.t
		m.totalUp = v.t.Up
		m.totalDn = v.t.Down
		if v.t.Up > m.maxUp {
			m.maxUp = v.t.Up
		}
		if v.t.Down > m.maxDn {
			m.maxDn = v.t.Down
		}
	}
	return nil, "", false
}

func (m *trafficModel) View() string {
	var b strings.Builder
	b.WriteString(stTitle.Render("Traffic") + "\n\n")
	b.WriteString(stBase.Render("Upload   ") + stOK.Render(humanBytes(m.cur.Up)+"/s") + "\n")
	b.WriteString(stBase.Render("         ") + bar(m.cur.Up, m.maxUp, 40) + "\n\n")
	b.WriteString(stBase.Render("Download ") + stOK.Render(humanBytes(m.cur.Down)+"/s") + "\n")
	b.WriteString(stBase.Render("         ") + bar(m.cur.Down, m.maxDn, 40) + "\n\n")
	b.WriteString(stMuted.Render("peak ↑" + humanBytes(m.maxUp) + "/s  ↓" + humanBytes(m.maxDn) + "/s\n"))
	b.WriteString(stMuted.Render("session ↑" + humanBytes(m.totalUp) + "  ↓" + humanBytes(m.totalDn)))
	return b.String()
}

func bar(v, max int64, w int) string {
	if max <= 0 {
		return strings.Repeat("·", w)
	}
	n := int(float64(v) / float64(max) * float64(w))
	if n > w {
		n = w
	}
	return stOK.Render(strings.Repeat("█", n)) + stMuted.Render(strings.Repeat("·", w-n))
}
