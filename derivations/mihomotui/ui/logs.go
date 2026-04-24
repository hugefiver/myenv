package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

type logEntryMsg struct {
	entry api.LogEntry
	err   error
}

type logsModel struct {
	cli     *api.Client
	entries []api.LogEntry
	cancel  context.CancelFunc
	level   string
	auto    bool
	scroll  int
	width   int
	height  int
	maxKeep int
}

func newLogsModel(cli *api.Client) *logsModel {
	return &logsModel{cli: cli, level: "info", auto: true, maxKeep: 1000}
}

func (m *logsModel) start(send func(tea.Msg)) tea.Cmd {
	m.stop()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	cli := m.cli
	level := m.level
	go func() {
		ch := make(chan api.LogEntry, 32)
		errCh := make(chan error, 1)
		go func() { errCh <- cli.StreamLogs(ctx, level, ch) }()
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-ch:
				send(logEntryMsg{entry: e})
			case err := <-errCh:
				if err != nil && ctx.Err() == nil {
					send(logEntryMsg{err: err})
				}
				return
			}
		}
	}()
	return nil
}

func (m *logsModel) stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *logsModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case logEntryMsg:
		if v.err != nil {
			return nil, "logs: " + v.err.Error(), true
		}
		m.entries = append(m.entries, v.entry)
		if len(m.entries) > m.maxKeep {
			m.entries = m.entries[len(m.entries)-m.maxKeep:]
		}
		if m.auto {
			m.scroll = m.maxScroll()
		}
	case tea.KeyMsg:
		switch v.String() {
		case "up", "k":
			m.auto = false
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			if m.scroll < m.maxScroll() {
				m.scroll++
			}
		case "a":
			m.auto = !m.auto
			if m.auto {
				m.scroll = m.maxScroll()
			}
		case "g":
			m.scroll = m.maxScroll()
			m.auto = true
		case "c":
			m.entries = nil
			m.scroll = 0
		}
	}
	return nil, "", false
}

func (m *logsModel) maxScroll() int {
	view := m.viewLines()
	n := len(m.entries) - view
	if n < 0 {
		return 0
	}
	return n
}

func (m *logsModel) viewLines() int {
	v := m.height - 6
	if v < 5 {
		v = 5
	}
	return v
}

func (m *logsModel) View() string {
	var b strings.Builder
	auto := "off"
	if m.auto {
		auto = "on"
	}
	b.WriteString(stTitle.Render("Logs"))
	b.WriteString(stMuted.Render("  level=" + m.level + "  auto=" + auto + "  (a=toggle, c=clear, g=bottom)"))
	b.WriteString("\n\n")
	view := m.viewLines()
	start := m.scroll
	end := start + view
	if end > len(m.entries) {
		end = len(m.entries)
	}
	for _, e := range m.entries[start:end] {
		col := stBase
		switch strings.ToLower(e.Type) {
		case "warning":
			col = stWarn
		case "error":
			col = stErr
		case "info":
			col = stOK
		}
		b.WriteString(col.Render("["+e.Type+"] ") + e.Payload + "\n")
	}
	return b.String()
}
