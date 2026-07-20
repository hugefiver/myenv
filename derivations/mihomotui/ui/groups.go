package ui

import (
	"context"
	"sort"
	"strings"

	"mihomotui/api"
)

type groupRow struct {
	name string
	now  string
	typ  string
}

type pane int

const (
	paneLeft pane = iota
	paneRight
)

type groupsModel struct {
	ctx               context.Context
	cli               *api.Client
	loadRequests      requestTracker
	selectionRequests requestTracker
	delayRequests     requestTracker
	rows              []groupRow
	all               map[string]api.Proxy
	groups            map[string]api.Proxy
	leftCursor        int
	leftScroll        int
	rightCursor       int
	rightScroll       int
	focus             pane
	delay             map[string]int
	width             int
	height            int

	cfg        *api.Config
	showGlobal bool
	searching  bool
	search     string
}

func newGroupsModel(ctx context.Context, cli *api.Client) *groupsModel {
	return &groupsModel{
		ctx:   ctx,
		cli:   cli,
		delay: map[string]int{},
		focus: paneLeft,
	}
}

func (m *groupsModel) setConfig(c *api.Config) {
	m.cfg = c
}

func (m *groupsModel) helpMode() string {
	if m.searching {
		return "search"
	}
	return ""
}

func shortType(t string) string {
	switch t {
	case "Selector":
		return "SEL"
	case "URLTest":
		return "URL"
	case "Fallback":
		return "FBK"
	case "LoadBalance":
		return "LB "
	}
	return t
}

func (m *groupsModel) ingest(proxies map[string]api.Proxy, groups []api.Proxy) {
	m.all = proxies
	if m.all == nil {
		m.all = map[string]api.Proxy{}
	}
	m.groups = make(map[string]api.Proxy, len(groups))
	rows := make([]groupRow, 0, len(groups))
	for _, group := range groups {
		m.groups[group.Name] = group
		rows = append(rows, groupRow{name: group.Name, now: group.Now, typ: group.Type})
	}
	sort.Slice(rows, func(i, j int) bool {
		ai, aj := rows[i].name == "GLOBAL", rows[j].name == "GLOBAL"
		if ai != aj {
			return aj
		}
		return rows[i].name < rows[j].name
	})
	m.rows = rows
	m.normalizeCursors()
}

func (m *groupsModel) hideGlobal() bool {
	return !m.showGlobal && m.cfg != nil && strings.EqualFold(m.cfg.Mode, "rule")
}

func (m *groupsModel) visibleRows() []groupRow {
	out := make([]groupRow, 0, len(m.rows))
	hideGlobal := m.hideGlobal()
	query := strings.ToLower(m.search)
	for _, row := range m.rows {
		if hideGlobal && row.name == "GLOBAL" {
			continue
		}
		if query != "" && m.focus == paneLeft && !strings.Contains(strings.ToLower(row.name), query) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func (m *groupsModel) groupByName(name string) (api.Proxy, bool) {
	if m.groups != nil {
		group, ok := m.groups[name]
		return group, ok
	}
	group, ok := m.all[name]
	return group, ok
}

func (m *groupsModel) currentGroup() *api.Proxy {
	rows := m.visibleRows()
	if !validIndex(m.leftCursor, len(rows)) {
		return nil
	}
	group, ok := m.groupByName(rows[m.leftCursor].name)
	if !ok {
		return nil
	}
	return &group
}

func (m *groupsModel) visibleNodes() []string {
	group := m.currentGroup()
	if group == nil {
		return nil
	}
	if m.focus != paneRight || m.search == "" {
		return group.All
	}
	query := strings.ToLower(m.search)
	nodes := make([]string, 0, len(group.All))
	for _, node := range group.All {
		if strings.Contains(strings.ToLower(node), query) {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (m *groupsModel) selectableGroup(name string) bool {
	group, ok := m.groupByName(name)
	return ok && group.Type == "Selector"
}

func (m *groupsModel) normalizeCursors() {
	rows := m.visibleRows()
	if !validIndex(m.leftCursor, len(rows)) {
		m.leftCursor = 0
	}
	if len(rows) == 0 {
		m.leftScroll, m.rightCursor, m.rightScroll = 0, 0, 0
		return
	}
	nodes := m.visibleNodes()
	if !validIndex(m.rightCursor, len(nodes)) {
		m.rightCursor = 0
	}
	if len(nodes) == 0 {
		m.rightScroll = 0
	}
}

func validIndex(index, length int) bool {
	return index >= 0 && index < length
}
