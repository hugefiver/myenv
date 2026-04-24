package ui

import (
	"context"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
)

type proxiesLoadedMsg struct {
	proxies map[string]api.Proxy
	err     error
}

type selectedMsg struct {
	group string
	node  string
	err   error
}

type groupDelayMsg struct {
	group string
	res   api.GroupDelayResp
	err   error
}

type nodeDelayMsg struct {
	name  string
	delay int
	err   error
}

type groupRow struct {
	name string
	now  string
	typ  string
}

type groupsModel struct {
	cli       *api.Client
	rows      []groupRow
	all       map[string]api.Proxy
	cursor    int
	inDetail  bool
	detailGrp string
	detailIdx int
	delay     map[string]int
	loading   bool
	width     int
	height    int
}

func newGroupsModel(cli *api.Client) *groupsModel {
	return &groupsModel{cli: cli, delay: map[string]int{}}
}

func (m *groupsModel) load() tea.Cmd {
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
		defer cancel()
		p, err := cli.Proxies(ctx)
		return proxiesLoadedMsg{proxies: p, err: err}
	}
}

func isGroupType(t string) bool {
	switch t {
	case "Selector", "URLTest", "Fallback", "LoadBalance":
		return true
	}
	return false
}

func (m *groupsModel) ingest(p map[string]api.Proxy) {
	m.all = p
	rows := make([]groupRow, 0, len(p))
	for name, pr := range p {
		if !isGroupType(pr.Type) {
			continue
		}
		rows = append(rows, groupRow{name: name, now: pr.Now, typ: pr.Type})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	m.rows = rows
	if m.cursor >= len(m.rows) {
		m.cursor = 0
	}
}

func (m *groupsModel) currentGroup() *api.Proxy {
	if m.detailGrp == "" {
		return nil
	}
	if g, ok := m.all[m.detailGrp]; ok {
		return &g
	}
	return nil
}

func (m *groupsModel) selectNode(group, node string) tea.Cmd {
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
		defer cancel()
		err := cli.SelectProxy(ctx, group, node)
		return selectedMsg{group: group, node: node, err: err}
	}
}

func (m *groupsModel) testGroup(name string) tea.Cmd {
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*reqTimeout)
		defer cancel()
		r, err := cli.GroupDelay(ctx, name, 5000)
		return groupDelayMsg{group: name, res: r, err: err}
	}
}

func (m *groupsModel) testNode(name string) tea.Cmd {
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*reqTimeout)
		defer cancel()
		d, err := cli.ProxyDelay(ctx, name, 5000)
		return nodeDelayMsg{name: name, delay: d, err: err}
	}
}

func (m *groupsModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case proxiesLoadedMsg:
		m.loading = false
		if v.err != nil {
			return nil, "load failed: " + v.err.Error(), true
		}
		m.ingest(v.proxies)
		return nil, "groups loaded", false
	case selectedMsg:
		if v.err != nil {
			return nil, "select failed: " + v.err.Error(), true
		}
		return m.load(), "selected " + v.node, false
	case groupDelayMsg:
		if v.err != nil {
			return nil, "group test failed: " + v.err.Error(), true
		}
		for k, d := range v.res {
			m.delay[k] = d
		}
		return nil, "group tested", false
	case nodeDelayMsg:
		if v.err != nil {
			m.delay[v.name] = -1
			return nil, "node test: " + v.err.Error(), true
		}
		m.delay[v.name] = v.delay
		return nil, "node tested", false
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return nil, "", false
}

func (m *groupsModel) handleKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	if !m.inDetail {
		switch k.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.rows) {
				m.detailGrp = m.rows[m.cursor].name
				m.detailIdx = 0
				m.inDetail = true
			}
		case "r":
			return m.load(), "refreshing", false
		}
		return nil, "", false
	}
	g := m.currentGroup()
	if g == nil {
		m.inDetail = false
		return nil, "", false
	}
	switch k.String() {
	case "esc":
		m.inDetail = false
	case "up", "k":
		if m.detailIdx > 0 {
			m.detailIdx--
		}
	case "down", "j":
		if m.detailIdx < len(g.All)-1 {
			m.detailIdx++
		}
	case "enter":
		if m.detailIdx < len(g.All) {
			return m.selectNode(g.Name, g.All[m.detailIdx]), "selecting", false
		}
	case "t":
		return m.testGroup(g.Name), "testing group", false
	case "d":
		if m.detailIdx < len(g.All) {
			return m.testNode(g.All[m.detailIdx]), "testing node", false
		}
	}
	return nil, "", false
}

func (m *groupsModel) View() string {
	if !m.inDetail {
		return m.viewList()
	}
	return m.viewDetail()
}

func (m *groupsModel) viewList() string {
	if len(m.rows) == 0 {
		return stMuted.Render("no groups (press r to refresh)")
	}
	var b strings.Builder
	b.WriteString(stTitle.Render("Groups") + "\n\n")
	for i, r := range m.rows {
		cur := "  "
		name := r.name
		if i == m.cursor {
			cur = stMark.Render("> ")
			name = lipgloss.NewStyle().Bold(true).Render(name)
		}
		b.WriteString(cur)
		b.WriteString(name)
		b.WriteString(stMuted.Render("  [" + r.typ + "]  → "))
		b.WriteString(stOK.Render(r.now))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *groupsModel) viewDetail() string {
	g := m.currentGroup()
	if g == nil {
		return stMuted.Render("no group")
	}
	var b strings.Builder
	b.WriteString(stTitle.Render("Group: "+g.Name) + stMuted.Render("  ["+g.Type+"]") + "\n")
	b.WriteString(stMuted.Render("current → ") + stOK.Render(g.Now) + "\n\n")
	for i, n := range g.All {
		cur := "  "
		display := n
		if i == m.detailIdx {
			cur = stMark.Render("> ")
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		marker := "  "
		if n == g.Now {
			marker = stOK.Render("● ")
		}
		b.WriteString(cur + marker + display)
		if d, ok := m.delay[n]; ok {
			if d <= 0 {
				b.WriteString(stErr.Render("  timeout"))
			} else {
				b.WriteString(stMuted.Render("  " + itoaMs(d)))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func itoaMs(d int) string {
	if d <= 0 {
		return "-"
	}
	return formatInt(d) + "ms"
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
