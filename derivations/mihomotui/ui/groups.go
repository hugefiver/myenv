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

type pane int

const (
	paneLeft pane = iota
	paneRight
)

type groupsModel struct {
	cli         *api.Client
	rows        []groupRow
	all         map[string]api.Proxy
	leftCursor  int
	leftScroll  int
	rightCursor int
	rightScroll int
	focus       pane
	delay       map[string]int
	width       int
	height      int

	cfg        *api.Config
	showGlobal bool

	searching bool
	search    string
	autoSelect bool
}

func newGroupsModel(cli *api.Client) *groupsModel {
	return &groupsModel{cli: cli, delay: map[string]int{}, focus: paneLeft}
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

func (m *groupsModel) ingest(p map[string]api.Proxy) {
	m.all = p
	rows := make([]groupRow, 0, len(p))
	for name, pr := range p {
		if !isGroupType(pr.Type) {
			continue
		}
		rows = append(rows, groupRow{name: name, now: pr.Now, typ: pr.Type})
	}
	sort.Slice(rows, func(i, j int) bool {
		ai, aj := rows[i].name == "GLOBAL", rows[j].name == "GLOBAL"
		if ai != aj {
			return aj
		}
		return rows[i].name < rows[j].name
	})
	m.rows = rows
	if m.leftCursor >= len(m.visibleRows()) {
		m.leftCursor = 0
	}
}

func (m *groupsModel) hideGlobal() bool {
	if m.showGlobal {
		return false
	}
	if m.cfg == nil {
		return false
	}
	return strings.EqualFold(m.cfg.Mode, "rule")
}

func (m *groupsModel) visibleRows() []groupRow {
	out := make([]groupRow, 0, len(m.rows))
	hideG := m.hideGlobal()
	q := strings.ToLower(m.search)
	for _, r := range m.rows {
		if hideG && r.name == "GLOBAL" {
			continue
		}
		if q != "" && m.focus == paneLeft {
			if !strings.Contains(strings.ToLower(r.name), q) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func (m *groupsModel) currentGroup() *api.Proxy {
	vr := m.visibleRows()
	if m.leftCursor >= len(vr) {
		return nil
	}
	g, ok := m.all[vr[m.leftCursor].name]
	if !ok {
		return nil
	}
	return &g
}

func (m *groupsModel) visibleNodes() []string {
	g := m.currentGroup()
	if g == nil {
		return nil
	}
	if m.focus == paneRight && m.search != "" {
		q := strings.ToLower(m.search)
		out := make([]string, 0, len(g.All))
		for _, n := range g.All {
			if strings.Contains(strings.ToLower(n), q) {
				out = append(out, n)
			}
		}
		return out
	}
	return g.All
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
		if v.err != nil {
			return nil, "load failed: " + v.err.Error(), true
		}
		m.ingest(v.proxies)
		for _, p := range v.proxies {
			if n := len(p.History); n > 0 {
				m.delay[p.Name] = p.History[n-1].Delay
			}
		}
		return nil, "groups loaded", false
	case selectedMsg:
		if v.err != nil {
			return nil, "select failed: " + v.err.Error(), true
		}
		return m.load(), "selected " + v.node, false
	case groupDelayMsg:
		if v.err != nil {
			m.autoSelect = false
			return nil, "group test failed: " + v.err.Error(), true
		}
		for k, d := range v.res {
			m.delay[k] = d
		}
		if m.autoSelect {
			m.autoSelect = false
			best := ""
			bestDelay := 0
			for name, d := range v.res {
				if d <= 0 {
					continue
				}
				if best == "" || d < bestDelay {
					best = name
					bestDelay = d
				}
			}
			if best == "" {
				return nil, "Auto: all nodes timed out", true
			}
			return m.selectNode(v.group, best), "Auto: " + best + " (" + itoaMs(bestDelay) + ")", false
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
	if m.searching {
		return m.handleSearchKey(k)
	}
	switch k.String() {
	case "tab", "h", "l", "left", "right":
		if m.focus == paneLeft {
			m.focus = paneRight
		} else {
			m.focus = paneLeft
		}
		return nil, "", false
	case "esc":
		return nil, "", false
	case "/":
		m.searching = true
		m.search = ""
		return nil, "search:", false
	case "g":
		m.showGlobal = !m.showGlobal
		if m.leftCursor >= len(m.visibleRows()) {
			m.leftCursor = 0
		}
		return nil, "", false
	case "r":
		return m.load(), "refreshing", false
	}
	if m.focus == paneLeft {
		return m.handleLeftKey(k)
	}
	return m.handleRightKey(k)
}

func (m *groupsModel) handleSearchKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	switch k.String() {
	case "esc":
		m.searching = false
		m.search = ""
	case "enter":
		m.searching = false
	case "backspace":
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
		}
	default:
		s := k.String()
		if len(s) == 1 {
			m.search += s
		}
	}
	return nil, "", false
}

func (m *groupsModel) handleLeftKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	vr := m.visibleRows()
	switch k.String() {
	case "up", "k":
		if m.leftCursor > 0 {
			m.leftCursor--
		}
	case "down", "j":
		if m.leftCursor < len(vr)-1 {
			m.leftCursor++
		}
	case "home":
		m.leftCursor = 0
	case "end":
		m.leftCursor = len(vr) - 1
	case "enter":
		m.focus = paneRight
		m.rightCursor = 0
		m.rightScroll = 0
	case "t":
		if m.leftCursor < len(vr) {
			return m.testGroup(vr[m.leftCursor].name), "testing group", false
		}
	case "T":
		if m.leftCursor < len(vr) {
			m.autoSelect = true
			return m.testGroup(vr[m.leftCursor].name), "Auto: testing & selecting", false
		}
	}
	return nil, "", false
}

func (m *groupsModel) handleRightKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	g := m.currentGroup()
	if g == nil {
		m.focus = paneLeft
		return nil, "", false
	}
	nodes := m.visibleNodes()
	switch k.String() {
	case "up", "k":
		if m.rightCursor > 0 {
			m.rightCursor--
		}
	case "down", "j":
		if m.rightCursor < len(nodes)-1 {
			m.rightCursor++
		}
	case "home":
		m.rightCursor = 0
	case "end":
		m.rightCursor = len(nodes) - 1
	case "enter":
		if m.rightCursor < len(nodes) {
			return m.selectNode(g.Name, nodes[m.rightCursor]), "selecting", false
		}
	case "t", "d":
		if m.rightCursor < len(nodes) {
			return m.testNode(nodes[m.rightCursor]), "testing node", false
		}
	case "T":
		m.autoSelect = true
		return m.testGroup(g.Name), "Auto: testing & selecting", false
	}
	return nil, "", false
}

func (m *groupsModel) View() string {
	header := m.renderHeader()
	body := m.renderSplit(header)
	return header + "\n" + body
}

func (m *groupsModel) renderHeader() string {
	mode := "?"
	tunOn := false
	if m.cfg != nil {
		mode = strings.ToLower(m.cfg.Mode)
		tunOn = m.cfg.Tun.Enable
	}
	var modeSt lipgloss.Style
	switch mode {
	case "global":
		modeSt = stWarn
	case "direct":
		modeSt = stCyan
	case "rule":
		modeSt = stOK
	default:
		modeSt = stMuted
	}
	tunStr := "off"
	tunSt := stErr
	if tunOn {
		tunStr = "on"
		tunSt = stOK
	}
	parts := []string{
		stMuted.Render("mode: ") + modeSt.Render(mode),
		stMuted.Render("tun: ") + tunSt.Render(tunStr),
		stMuted.Render("groups: ") + stBase.Render(formatInt(len(m.visibleRows()))),
	}
	if m.searching || m.search != "" {
		tag := "/" + m.search
		if m.searching {
			tag += "_"
		}
		parts = append(parts, stMark.Render(tag))
	}
	if !m.showGlobal && m.cfg != nil && strings.EqualFold(m.cfg.Mode, "rule") {
		parts = append(parts, stMuted.Render("(GLOBAL hidden, g to show)"))
	}
	return strings.Join(parts, stMuted.Render("   "))
}

func (m *groupsModel) renderSplit(header string) string {
	headerH := lipgloss.Height(header) + 1
	availH := m.height - headerH
	if availH < 5 {
		availH = 5
	}
	leftW := m.width * 35 / 100
	if leftW < 24 {
		leftW = 24
	}
	if leftW > m.width-30 {
		leftW = m.width - 30
	}
	rightW := m.width - leftW - 2
	if rightW < 20 {
		rightW = 20
	}
	innerH := availH - 2
	if innerH < 3 {
		innerH = 3
	}
	left := m.renderLeft(leftW-4, innerH)
	right := m.renderRight(rightW-4, innerH)
	leftSt := stPane
	rightSt := stPane
	if m.focus == paneLeft {
		leftSt = stPaneA
	} else {
		rightSt = stPaneA
	}
	leftBox := leftSt.Width(leftW - 2).Height(innerH).Render(left)
	rightBox := rightSt.Width(rightW - 2).Height(innerH).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func (m *groupsModel) renderLeft(width, height int) string {
	vr := m.visibleRows()
	if len(vr) == 0 {
		return stMuted.Render("no groups")
	}
	m.leftScroll = clampScroll(m.leftCursor, m.leftScroll, height, len(vr))
	end := m.leftScroll + height
	if end > len(vr) {
		end = len(vr)
	}
	var b strings.Builder
	for i := m.leftScroll; i < end; i++ {
		r := vr[i]
		cur := "  "
		if i == m.leftCursor {
			cur = stMark.Render("▸ ")
		}
		typeTag := stMuted.Render("[" + shortType(r.typ) + "] ")
		name := r.name
		if i == m.leftCursor {
			name = lipgloss.NewStyle().Bold(true).Render(name)
		}
		now := stOK.Render(r.now)
		latency := ""
		if d, ok := m.delay[r.now]; ok && d > 0 {
			latency = stMuted.Render("  " + itoaMs(d))
		}
		line := cur + typeTag + name + stMuted.Render(" → ") + now + latency
		b.WriteString(truncWide(line, width) + "\n")
	}
	return b.String()
}

func (m *groupsModel) renderRight(width, height int) string {
	g := m.currentGroup()
	if g == nil {
		return stMuted.Render("no group selected")
	}
	nodes := m.visibleNodes()
	title := stTitle.Render(g.Name) + stMuted.Render("  ["+g.Type+"]  current → ") + stOK.Render(g.Now)
	header := truncWide(title, width)
	bodyH := height - 2
	if bodyH < 1 {
		bodyH = 1
	}
	if len(nodes) == 0 {
		return header + "\n\n" + stMuted.Render("no nodes")
	}
	if m.rightCursor >= len(nodes) {
		m.rightCursor = len(nodes) - 1
	}
	m.rightScroll = clampScroll(m.rightCursor, m.rightScroll, bodyH, len(nodes))
	end := m.rightScroll + bodyH
	if end > len(nodes) {
		end = len(nodes)
	}
	nameW := width - 28
	if nameW < 10 {
		nameW = 10
	}
	var b strings.Builder
	b.WriteString(header + "\n\n")
	for i := m.rightScroll; i < end; i++ {
		n := nodes[i]
		cur := "  "
		if i == m.rightCursor {
			cur = stMark.Render("▸ ")
		}
		marker := "  "
		if n == g.Now {
			marker = stOK.Render("● ")
		}
		display := n
		if i == m.rightCursor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		typ := ""
		if pr, ok := m.all[n]; ok {
			typ = pr.Type
		}
		latency := ""
		if d, ok := m.delay[n]; ok {
			if d <= 0 {
				latency = stErr.Render("timeout")
			} else {
				latency = stMuted.Render(itoaMs(d))
			}
		}
		line := cur + marker + padR(trunc(display, nameW), nameW) + " " + stMuted.Render(padR(trunc(typ, 12), 12)) + " " + latency
		b.WriteString(truncWide(line, width) + "\n")
	}
	return b.String()
}

func truncWide(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(w).MaxHeight(1).Inline(true).Render(s)
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
