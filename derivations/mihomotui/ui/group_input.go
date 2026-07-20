package ui

import tea "github.com/charmbracelet/bubbletea"

func (m *groupsModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch result := msg.(type) {
	case groupsLoadedMsg:
		return m.handleGroupsLoaded(result)
	case selectedMsg:
		return m.handleSelected(result)
	case groupDelayMsg:
		return m.handleGroupDelay(result)
	case groupNodeDelayMsg:
		target := result.Target
		if target == "" {
			target = groupNodeDelayTarget(result.Group, result.Node)
		}
		if !m.delayRequests.IsCurrent(result.RequestID, target) {
			return nil, "", false
		}
		if result.Err != nil {
			m.delay[result.Node] = -1
			return nil, "node test: " + result.Err.Error(), true
		}
		m.delay[result.Node] = result.Delay
		return nil, "node tested", false
	case tea.KeyMsg:
		return m.handleKey(result)
	}
	return nil, "", false
}

func (m *groupsModel) handleKey(key tea.KeyMsg) (tea.Cmd, string, bool) {
	m.normalizeCursors()
	if m.searching {
		return m.handleSearchKey(key)
	}
	switch key.String() {
	case "tab", "h", "l", "left", "right":
		if len(m.visibleRows()) == 0 || len(m.visibleNodes()) == 0 {
			return nil, "", false
		}
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
		m.normalizeCursors()
		return nil, "", false
	case "r":
		return m.load(), "refreshing", false
	}
	if m.focus == paneLeft {
		return m.handleLeftKey(key)
	}
	return m.handleRightKey(key)
}

func (m *groupsModel) handleSearchKey(key tea.KeyMsg) (tea.Cmd, string, bool) {
	switch key.Type {
	case tea.KeyEsc:
		m.searching = false
		m.search = ""
	case tea.KeyEnter:
		m.searching = false
	case tea.KeyBackspace:
		m.search = removeLastRune(m.search)
	default:
		m.search = appendKeyRunes(m.search, key)
	}
	m.normalizeCursors()
	return nil, "", false
}

func (m *groupsModel) handleLeftKey(key tea.KeyMsg) (tea.Cmd, string, bool) {
	rows := m.visibleRows()
	if len(rows) == 0 {
		m.normalizeCursors()
		return nil, "", false
	}
	if !validIndex(m.leftCursor, len(rows)) {
		m.leftCursor = 0
	}
	switch key.String() {
	case "up", "k":
		if m.leftCursor > 0 {
			m.leftCursor--
		}
	case "down", "j":
		if m.leftCursor+1 < len(rows) {
			m.leftCursor++
		}
	case "home":
		m.leftCursor = 0
	case "end":
		m.leftCursor = len(rows) - 1
	case "enter":
		if len(m.visibleNodes()) == 0 {
			return nil, "", false
		}
		m.focus, m.rightCursor, m.rightScroll = paneRight, 0, 0
	case "t", "T":
		if len(m.visibleNodes()) == 0 {
			return nil, "", false
		}
		selectFastest := key.String() == "T"
		return m.testGroup(rows[m.leftCursor].name, selectFastest), groupTestStatus(selectFastest), false
	}
	return nil, "", false
}

func (m *groupsModel) handleRightKey(key tea.KeyMsg) (tea.Cmd, string, bool) {
	group := m.currentGroup()
	if group == nil {
		m.normalizeCursors()
		return nil, "", false
	}
	nodes := m.visibleNodes()
	if len(nodes) == 0 {
		m.normalizeCursors()
		return nil, "", false
	}
	if !validIndex(m.rightCursor, len(nodes)) {
		m.rightCursor = 0
	}
	switch key.String() {
	case "up", "k":
		if m.rightCursor > 0 {
			m.rightCursor--
		}
	case "down", "j":
		if m.rightCursor+1 < len(nodes) {
			m.rightCursor++
		}
	case "home":
		m.rightCursor = 0
	case "end":
		m.rightCursor = len(nodes) - 1
	case "enter":
		return m.selectNode(group.Name, nodes[m.rightCursor]), "selecting", false
	case "t", "d":
		return m.testNode(group.Name, nodes[m.rightCursor]), "testing node", false
	case "T":
		return m.testGroup(group.Name, true), groupTestStatus(true), false
	}
	return nil, "", false
}

func groupTestStatus(selectFastest bool) string {
	if selectFastest {
		return "Auto: testing & selecting"
	}
	return "testing group"
}
