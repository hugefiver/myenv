package ui

import (
	"context"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
)

type proxyRow struct {
	name  string
	typ   string
	delay int
}

type proxiesModel struct {
	cli    *api.Client
	rows   []proxyRow
	cursor int
	scroll int
	width  int
	height int
}

func newProxiesModel(cli *api.Client) *proxiesModel {
	return &proxiesModel{cli: cli}
}

func (m *proxiesModel) load() tea.Cmd {
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
		defer cancel()
		p, err := cli.Proxies(ctx)
		return proxiesAllMsg{proxies: p, err: err}
	}
}

type proxiesAllMsg struct {
	proxies map[string]api.Proxy
	err     error
}

func isLeafProxy(t string) bool {
	switch t {
	case "Direct", "Reject", "RejectDrop", "Compatible", "Pass":
		return false
	}
	return !isGroupType(t)
}

func (m *proxiesModel) ingest(p map[string]api.Proxy) {
	rows := make([]proxyRow, 0, len(p))
	for _, pr := range p {
		if !isLeafProxy(pr.Type) {
			continue
		}
		d := 0
		if n := len(pr.History); n > 0 {
			d = pr.History[n-1].Delay
		}
		rows = append(rows, proxyRow{name: pr.Name, typ: pr.Type, delay: d})
	}
	sort.Slice(rows, func(i, j int) bool {
		di, dj := rows[i].delay, rows[j].delay
		if di == 0 {
			di = 1 << 30
		}
		if dj == 0 {
			dj = 1 << 30
		}
		if di != dj {
			return di < dj
		}
		return rows[i].name < rows[j].name
	})
	m.rows = rows
	if m.cursor >= len(m.rows) {
		m.cursor = 0
	}
}

func (m *proxiesModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case proxiesAllMsg:
		if v.err != nil {
			return nil, "load failed: " + v.err.Error(), true
		}
		m.ingest(v.proxies)
		return nil, "proxies loaded", false
	case nodeDelayMsg:
		if v.err != nil {
			return nil, "test: " + v.err.Error(), true
		}
		for i := range m.rows {
			if m.rows[i].name == v.name {
				m.rows[i].delay = v.delay
			}
		}
		return nil, "tested " + v.name, false
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return nil, "", false
}

func (m *proxiesModel) handleKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	switch k.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
		}
	case "r":
		return m.load(), "refreshing", false
	case "d", "t":
		if m.cursor < len(m.rows) {
			name := m.rows[m.cursor].name
			cli := m.cli
			return func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 10*reqTimeout)
				defer cancel()
				d, err := cli.ProxyDelay(ctx, name, 5000)
				return nodeDelayMsg{name: name, delay: d, err: err}
			}, "testing " + name, false
		}
	}
	return nil, "", false
}

func (m *proxiesModel) View() string {
	if len(m.rows) == 0 {
		return stMuted.Render("no proxies (press r)")
	}
	var b strings.Builder
	b.WriteString(stTitle.Render("Proxies") + "  " + stMuted.Render("(d/t test, r refresh)") + "\n\n")
	view := m.height - 3
	if view < 1 {
		view = 1
	}
	m.scroll = clampScroll(m.cursor, m.scroll, view, len(m.rows))
	end := m.scroll + view
	if end > len(m.rows) {
		end = len(m.rows)
	}
	for i := m.scroll; i < end; i++ {
		r := m.rows[i]
		cur := "  "
		name := r.name
		if i == m.cursor {
			cur = stMark.Render("▸ ")
			name = lipgloss.NewStyle().Bold(true).Render(name)
		}
		line := cur + name + stMuted.Render("  ["+r.typ+"]")
		if r.delay > 0 {
			line += stMuted.Render("  " + itoaMs(r.delay))
		} else if r.delay < 0 {
			line += stErr.Render("  timeout")
		}
		b.WriteString(truncWide(line, m.width) + "\n")
	}
	return b.String()
}
