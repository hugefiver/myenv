package ui

import (
	"fmt"
)

type tab int

const (
	tabGroups tab = iota
	tabProxies
	tabConns
	tabLogs
	tabTraffic
	tabProfiles
	tabConfig
	tabCount
)

var tabNames = [...]string{"1 Groups", "2 Proxies", "3 Conns", "4 Logs", "5 Traffic", "6 Profiles", "7 Config"}

func renderTabs(active tab, width int) string {
	if width <= 0 {
		return ""
	}
	cells := make([]textSegment, 0, tabCount)
	for i := tab(0); i < tabCount; i++ {
		s := stTabIdle
		if i == active {
			s = stTabActive
		}
		cells = append(cells, segment(s, " "+tabNames[i]+" "))
	}
	return stTabBar.Width(width).Render(renderTextLine(width, cells...))
}

func renderStatus(width int, status string, isErr bool) string {
	if width <= 2 {
		return ""
	}
	st := stStatus
	if isErr {
		st = st.Foreground(colErr)
	}
	if status == "" {
		status = " "
	}
	contentWidth := width - 2
	return st.Width(contentWidth).Render(renderTextLine(contentWidth, segment(stPlain, status)))
}

func renderHelp(width int, help string) string {
	if width <= 0 {
		return ""
	}
	return stHelp.Width(width).Render(renderTextLine(width, segment(stHelp, help)))
}

func humanBytes(n int64) string {
	const k = 1024.0
	f := float64(n)
	switch {
	case f < k:
		return fmt.Sprintf("%dB", n)
	case f < k*k:
		return fmt.Sprintf("%.1fK", f/k)
	case f < k*k*k:
		return fmt.Sprintf("%.1fM", f/(k*k))
	case f < k*k*k*k:
		return fmt.Sprintf("%.2fG", f/(k*k*k))
	default:
		return fmt.Sprintf("%.2fT", f/(k*k*k*k))
	}
}
