package ui

import (
	"context"
	"errors"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
	"mihomotui/internal/termtext"
)

type proxyRow struct {
	rawName string
	name    string
	typ     string
	delay   int
}

type proxiesModel struct {
	ctx           context.Context
	cli           *api.Client
	loadRequests  requestTracker
	delayRequests requestTracker
	rows          []proxyRow
	cursor        int
	scroll        int
	width         int
	height        int
}

const proxiesLoadTarget = "proxies"

func newProxiesModel(ctx context.Context, cli *api.Client) *proxiesModel {
	return &proxiesModel{ctx: ctx, cli: cli}
}

func (m *proxiesModel) load() tea.Cmd {
	id := m.loadRequests.Begin(proxiesLoadTarget)
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		type proxyResult struct {
			proxies map[string]api.Proxy
			err     error
		}
		type groupResult struct {
			groups []api.Proxy
			err    error
		}
		proxies := make(chan proxyResult, 1)
		groups := make(chan groupResult, 1)
		go func() {
			result, err := cli.Proxies(ctx)
			proxies <- proxyResult{proxies: result, err: err}
		}()
		go func() {
			result, err := cli.Groups(ctx)
			groups <- groupResult{groups: result, err: err}
		}()
		loadedProxies := <-proxies
		loadedGroups := <-groups
		return proxiesAllMsg{
			RequestID: id,
			Target:    proxiesLoadTarget,
			Proxies:   loadedProxies.proxies,
			Groups:    loadedGroups.groups,
			Err:       errors.Join(loadedProxies.err, loadedGroups.err),
		}
	}
}

type proxiesAllMsg struct {
	RequestID requestID
	Target    string
	Proxies   map[string]api.Proxy
	Groups    []api.Proxy
	Err       error
}

type proxyDelayMsg struct {
	RequestID requestID
	Node      string
	Delay     int
	Err       error
}

func isLeafProxy(t string) bool {
	switch t {
	case "Direct", "Reject", "RejectDrop", "Compatible", "Pass":
		return false
	}
	return true
}

func (m *proxiesModel) ingest(p map[string]api.Proxy, groups []api.Proxy) {
	groupNames := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupNames[group.Name] = struct{}{}
	}
	rows := make([]proxyRow, 0, len(p))
	for _, pr := range p {
		if _, isGroup := groupNames[pr.Name]; isGroup || !isLeafProxy(pr.Type) {
			continue
		}
		name := termtext.SingleLine(pr.Name)
		typ := termtext.SingleLine(pr.Type)
		d := 0
		if n := len(pr.History); n > 0 {
			d = pr.History[n-1].Delay
		}
		rows = append(rows, proxyRow{rawName: pr.Name, name: name, typ: typ, delay: d})
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
		if !m.loadRequests.IsCurrent(v.RequestID, v.Target) {
			return nil, "", false
		}
		if v.Err != nil {
			return nil, "load failed: " + v.Err.Error(), true
		}
		m.ingest(v.Proxies, v.Groups)
		return nil, "proxies loaded", false
	case proxyDelayMsg:
		if !m.delayRequests.IsCurrent(v.RequestID, v.Node) {
			return nil, "", false
		}
		if v.Err != nil {
			return nil, "test: " + v.Err.Error(), true
		}
		for i := range m.rows {
			if m.rows[i].rawName == v.Node {
				m.rows[i].delay = v.Delay
			}
		}
		return nil, "tested " + termtext.SingleLine(v.Node), false
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
			row := m.rows[m.cursor]
			id := m.delayRequests.Begin(row.rawName)
			cli := m.cli
			return func() tea.Msg {
				ctx, cancel := context.WithTimeout(m.ctx, 10*reqTimeout)
				defer cancel()
				d, err := cli.ProxyDelay(ctx, row.rawName, 5000)
				return proxyDelayMsg{RequestID: id, Node: row.rawName, Delay: d, Err: err}
			}, "testing " + row.name, false
		}
	}
	return nil, "", false
}

func (m *proxiesModel) View() string {
	if len(m.rows) == 0 {
		return renderTextLine(m.width, segment(stMuted, "no proxies (press r)"))
	}
	var b strings.Builder
	b.WriteString(renderTextLine(m.width, segment(stTitle, "Proxies"), segment(stMuted, "  (d/t test, r refresh)")) + "\n\n")
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
		curStyle := stPlain
		nameStyle := stPlain
		if i == m.cursor {
			cur, curStyle = "▸ ", stMark
			nameStyle = lipgloss.NewStyle().Bold(true)
		}
		parts := []textSegment{
			segment(curStyle, cur),
			segment(nameStyle, termtext.Truncate(r.name, m.width)),
			segment(stMuted, "  ["+termtext.Truncate(r.typ, m.width)+"]"),
		}
		if r.delay > 0 {
			parts = append(parts, segment(stMuted, "  "+itoaMs(r.delay)))
		} else if r.delay < 0 {
			parts = append(parts, segment(stErr, "  timeout"))
		}
		b.WriteString(renderTextLine(m.width, parts...) + "\n")
	}
	return b.String()
}
