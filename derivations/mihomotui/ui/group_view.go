package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"mihomotui/internal/termtext"
)

func (m *groupsModel) View() string {
	header := m.renderHeader()
	return header + "\n" + m.renderSplit(header)
}

func (m *groupsModel) renderHeader() string {
	mode := "?"
	tunOn := false
	if m.cfg != nil {
		mode = termtext.Truncate(strings.ToLower(termtext.SingleLine(m.cfg.Mode)), m.width)
		tunOn = m.cfg.Tun.Enable
	}
	modeStyle := stMuted
	switch mode {
	case "global":
		modeStyle = stWarn
	case "direct":
		modeStyle = stCyan
	case "rule":
		modeStyle = stOK
	}
	tunText, tunStyle := "off", stErr
	if tunOn {
		tunText, tunStyle = "on", stOK
	}
	parts := []textSegment{
		segment(stMuted, "mode: "), segment(modeStyle, mode),
		segment(stMuted, "   tun: "), segment(tunStyle, tunText),
		segment(stMuted, "   groups: "), segment(stBase, formatInt(len(m.visibleRows()))),
	}
	if m.searching || m.search != "" {
		tag := "/" + termtext.Truncate(m.search, m.width)
		if m.searching {
			tag += "_"
		}
		parts = append(parts, segment(stMark, "   "+tag))
	}
	if m.hideGlobal() {
		parts = append(parts, segment(stMuted, "   (GLOBAL hidden, g to show)"))
	}
	return renderTextLine(m.width, parts...)
}

func (m *groupsModel) renderSplit(header string) string {
	availableHeight := m.height - lipgloss.Height(header) - 1
	if availableHeight < 5 {
		availableHeight = 5
	}
	width := m.width
	if width < 46 {
		if m.focus == paneRight {
			return m.renderRight(width, availableHeight)
		}
		return m.renderLeft(width, availableHeight)
	}
	leftWidth := width * 35 / 100
	if leftWidth < 24 {
		leftWidth = 24
	}
	if leftWidth > width-30 {
		leftWidth = width - 30
	}
	rightWidth := width - leftWidth - 2
	if rightWidth < 20 {
		rightWidth = 20
	}
	innerHeight := availableHeight - 2
	if innerHeight < 3 {
		innerHeight = 3
	}
	left := m.renderLeft(leftWidth-4, innerHeight)
	right := m.renderRight(rightWidth-4, innerHeight)
	leftStyle, rightStyle := stPane, stPane
	if m.focus == paneLeft {
		leftStyle = stPaneA
	} else {
		rightStyle = stPaneA
	}
	leftBox := leftStyle.Width(leftWidth - 2).Height(innerHeight).Render(left)
	rightBox := rightStyle.Width(rightWidth - 2).Height(innerHeight).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

func (m *groupsModel) renderLeft(width, height int) string {
	rows := m.visibleRows()
	if len(rows) == 0 {
		m.leftCursor, m.leftScroll, m.rightCursor, m.rightScroll = 0, 0, 0, 0
		return stMuted.Render("no groups")
	}
	if !validIndex(m.leftCursor, len(rows)) {
		m.leftCursor = 0
	}
	m.leftScroll = clampScroll(m.leftCursor, m.leftScroll, height, len(rows))
	end := m.leftScroll + height
	if end > len(rows) {
		end = len(rows)
	}
	var output strings.Builder
	for index := m.leftScroll; index < end; index++ {
		row := rows[index]
		cursor := "  "
		cursorStyle := stPlain
		if index == m.leftCursor {
			cursor, cursorStyle = "▸ ", stMark
		}
		name := termtext.Truncate(row.name, width)
		nameStyle := stPlain
		if index == m.leftCursor {
			nameStyle = lipgloss.NewStyle().Bold(true)
		}
		latency := ""
		if delay, ok := m.delay[row.name]; ok && delay > 0 {
			latency = "  " + itoaMs(delay)
		}
		typeText := "[" + termtext.Truncate(shortType(row.typ), width) + "] "
		nameWidth := width - termtext.Width(cursor) - termtext.Width(typeText) -
			termtext.Width(" → ") - termtext.Width(row.now) - termtext.Width(latency)
		name = termtext.Truncate(row.name, nameWidth)
		output.WriteString(renderTextLine(width,
			segment(cursorStyle, cursor),
			segment(stMuted, typeText),
			segment(nameStyle, name),
			segment(stMuted, " → "),
			segment(stOK, termtext.Truncate(row.now, width)),
			segment(stMuted, latency),
		) + "\n")
	}
	return output.String()
}

func (m *groupsModel) renderRight(width, height int) string {
	group := m.currentGroup()
	if group == nil {
		m.rightCursor, m.rightScroll = 0, 0
		return stMuted.Render("no group selected")
	}
	nodes := m.visibleNodes()
	header := renderTextLine(width,
		segment(stTitle, group.Name),
		segment(stMuted, "  ["+termtext.Truncate(group.Type, width)+"]  current → "),
		segment(stOK, group.Now),
	)
	bodyHeight := height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if len(nodes) == 0 {
		m.rightCursor, m.rightScroll = 0, 0
		return header + "\n\n" + stMuted.Render("no nodes")
	}
	if !validIndex(m.rightCursor, len(nodes)) {
		m.rightCursor = 0
	}
	m.rightScroll = clampScroll(m.rightCursor, m.rightScroll, bodyHeight, len(nodes))
	end := m.rightScroll + bodyHeight
	if end > len(nodes) {
		end = len(nodes)
	}
	nameWidth := width - 28
	if nameWidth < 10 {
		nameWidth = 10
	}
	var output strings.Builder
	output.WriteString(header + "\n\n")
	for index := m.rightScroll; index < end; index++ {
		node := nodes[index]
		cursor := "  "
		cursorStyle := stPlain
		if index == m.rightCursor {
			cursor, cursorStyle = "▸ ", stMark
		}
		marker, markerStyle := "  ", stPlain
		if node == group.Now {
			marker, markerStyle = "● ", stOK
		}
		display := termtext.PadRight(node, nameWidth)
		displayStyle := stPlain
		if index == m.rightCursor {
			displayStyle = lipgloss.NewStyle().Bold(true)
		}
		proxyType := ""
		if proxy, ok := m.all[node]; ok {
			proxyType = proxy.Type
		}
		latency, latencyStyle := "", stPlain
		if delay, ok := m.delay[node]; ok {
			if delay <= 0 {
				latency, latencyStyle = "timeout", stErr
			} else {
				latency, latencyStyle = itoaMs(delay), stMuted
			}
		}
		output.WriteString(renderTextLine(width,
			segment(cursorStyle, cursor),
			segment(markerStyle, marker),
			segment(displayStyle, display),
			segment(stPlain, " "),
			segment(stMuted, termtext.PadRight(proxyType, 12)),
			segment(stPlain, " "),
			segment(latencyStyle, latency),
		) + "\n")
	}
	return output.String()
}

func itoaMs(delay int) string {
	if delay <= 0 {
		return "-"
	}
	return formatInt(delay) + "ms"
}

func formatInt(number int) string {
	if number == 0 {
		return "0"
	}
	negative := number < 0
	if negative {
		number = -number
	}
	var buffer [20]byte
	index := len(buffer)
	for number > 0 {
		index--
		buffer[index] = byte('0' + number%10)
		number /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
