package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabGroups tab = iota
	tabProxies
	tabConns
	tabLogs
	tabTraffic
	tabConfig
	tabCount
)

var tabNames = [...]string{"1 Groups", "2 Proxies", "3 Conns", "4 Logs", "5 Traffic", "6 Config"}

func renderTabs(active tab, width int) string {
	cells := make([]string, 0, tabCount)
	for i := tab(0); i < tabCount; i++ {
		s := stTabIdle
		if i == active {
			s = stTabActive
		}
		cells = append(cells, s.Render(tabNames[i]))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	return stTabBar.Width(width).Render(row)
}

func renderStatus(width int, status string, isErr bool) string {
	st := stStatus
	if isErr {
		st = st.Foreground(colErr)
	}
	if status == "" {
		status = " "
	}
	return st.Width(width).Render(status)
}

func renderHelp(width int, help string) string {
	return stHelp.Width(width).Render(help)
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
