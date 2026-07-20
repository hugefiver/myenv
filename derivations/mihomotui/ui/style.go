package ui

import "github.com/charmbracelet/lipgloss"

var (
	colBase     = lipgloss.Color("248")
	colMuted    = lipgloss.Color("242")
	colAccent   = lipgloss.Color("75")
	colOK       = lipgloss.Color("78")
	colWarn     = lipgloss.Color("214")
	colErr      = lipgloss.Color("203")
	colBgAlt    = lipgloss.Color("236")
	colBorder   = lipgloss.Color("240")
	colBorderHi = lipgloss.Color("75")
	colCyan     = lipgloss.Color("44")

	stTabActive = lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(colAccent).Bold(true)
	stTabIdle   = lipgloss.NewStyle().Foreground(colMuted)
	stTabBar    = lipgloss.NewStyle().Background(colBgAlt)

	stTitle  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stPlain  = lipgloss.NewStyle()
	stMuted  = lipgloss.NewStyle().Foreground(colMuted)
	stOK     = lipgloss.NewStyle().Foreground(colOK)
	stWarn   = lipgloss.NewStyle().Foreground(colWarn)
	stErr    = lipgloss.NewStyle().Foreground(colErr)
	stBase   = lipgloss.NewStyle().Foreground(colBase)
	stHelp   = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	stStatus = lipgloss.NewStyle().Foreground(colBase).Background(colBgAlt).Padding(0, 1)
	stMark   = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stCyan   = lipgloss.NewStyle().Foreground(colCyan)

	stFrame = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colBorder).Padding(0, 1)
	stPane  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colBorder).Padding(0, 1)
	stPaneA = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colBorderHi).Padding(0, 1)
)
