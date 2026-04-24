package ui

import "github.com/charmbracelet/lipgloss"

var (
	colBase   = lipgloss.Color("248")
	colMuted  = lipgloss.Color("242")
	colAccent = lipgloss.Color("75")
	colOK     = lipgloss.Color("78")
	colWarn   = lipgloss.Color("214")
	colErr    = lipgloss.Color("203")
	colBgAlt  = lipgloss.Color("236")

	stTabActive = lipgloss.NewStyle().Foreground(colAccent).Bold(true).Underline(true).Padding(0, 1)
	stTabIdle   = lipgloss.NewStyle().Foreground(colMuted).Padding(0, 1)
	stTabBar    = lipgloss.NewStyle().Background(colBgAlt)

	stTitle  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stMuted  = lipgloss.NewStyle().Foreground(colMuted)
	stOK     = lipgloss.NewStyle().Foreground(colOK)
	stWarn   = lipgloss.NewStyle().Foreground(colWarn)
	stErr    = lipgloss.NewStyle().Foreground(colErr)
	stBase   = lipgloss.NewStyle().Foreground(colBase)
	stHelp   = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	stStatus = lipgloss.NewStyle().Foreground(colBase).Background(colBgAlt).Padding(0, 1)
	stMark   = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
)
