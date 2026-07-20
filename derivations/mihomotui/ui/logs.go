package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	"mihomotui/internal/termtext"
)

type logEntryMsg struct {
	generation requestID
	entry      api.LogEntry
	err        error
	end        bool
}

const (
	maxLogEntries   = 1000
	maxLogTextBytes = 4 << 20
)

type logsModel struct {
	ctx        context.Context
	cli        *api.Client
	bus        *messageBus
	entries    []api.LogEntry
	cancel     context.CancelFunc
	generation requestID
	send       func(tea.Msg)
	level      string
	auto       bool
	scroll     int
	width      int
	height     int
	textBytes  int
	dropped    uint64
	errText    string
}

func newLogsModel(ctx context.Context, cli *api.Client, bus *messageBus) *logsModel {
	return &logsModel{ctx: ctx, cli: cli, bus: bus, level: "info", auto: true}
}

func (m *logsModel) start(send func(tea.Msg)) tea.Cmd {
	ctx, generation := beginStream(m.ctx, m.bus, streamLogs, &m.generation, &m.cancel)
	m.send = send
	m.errText = ""
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
				send(logEntryMsg{generation: generation, entry: e})
			case err := <-errCh:
				if ctx.Err() == nil && !streamCanceled(err) {
					send(logEntryMsg{generation: generation, err: err, end: true})
				}
				return
			}
		}
	}()
	return nil
}

func (m *logsModel) addDrops(count uint64) {
	m.dropped += count
}

func (m *logsModel) stop() {
	stopStream(&m.generation, &m.cancel)
}

func (m *logsModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case logEntryMsg:
		if v.generation != m.generation {
			return nil, "", false
		}
		if v.err != nil {
			if streamCanceled(v.err) {
				return nil, "", false
			}
			m.errText = termtext.SingleLine(v.err.Error())
			return nil, "logs: " + m.errText, true
		}
		if v.end {
			m.errText = "stream ended unexpectedly"
			return nil, "logs: " + m.errText, true
		}
		m.errText = ""
		m.retain(v.entry)
		if m.auto {
			m.scroll = m.maxScroll()
		}
	case tea.KeyMsg:
		switch v.String() {
		case "r":
			return m.start(m.send), "logs retrying", false
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
			m.textBytes = 0
			m.scroll = 0
		}
	}
	return nil, "", false
}

func (m *logsModel) retain(entry api.LogEntry) {
	entry.Type = termtext.SingleLine(entry.Type)
	entry.Payload = termtext.Multiline(entry.Payload)
	bytes := len(entry.Type) + len(entry.Payload)
	if bytes > maxLogTextBytes {
		m.dropped++
		return
	}
	m.entries = append(m.entries, entry)
	m.textBytes += bytes
	for len(m.entries) > maxLogEntries || m.textBytes > maxLogTextBytes {
		m.textBytes -= logTextBytes(m.entries[0])
		m.entries = m.entries[1:]
		m.dropped++
	}
}

func logTextBytes(entry api.LogEntry) int {
	// Retention counts UTF-8 bytes in sanitized type and payload text.
	return len(entry.Type) + len(entry.Payload)
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
	header := []textSegment{
		segment(stTitle, "Logs"),
		segment(stMuted, "  level="+m.level+"  auto="+auto+"  (a=toggle, c=clear, g=bottom, r=retry)"),
	}
	if m.errText != "" {
		header = append(header, segment(stErr, "  error="+m.errText))
	}
	if m.dropped > 0 {
		header = append(header, segment(stWarn, "  dropped="+formatInt(int(m.dropped))))
	}
	b.WriteString(renderTextLine(m.width, header...))
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
		for _, line := range strings.Split(e.Payload, "\n") {
			b.WriteString(renderTextLine(m.width,
				segment(col, "["+e.Type+"] "),
				segment(col, line),
			) + "\n")
		}
	}
	return b.String()
}
