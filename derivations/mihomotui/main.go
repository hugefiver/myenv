package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	"mihomotui/ui"
)

func main() {
	base := os.Getenv("MIHOMO_API")
	if base == "" {
		base = "http://127.0.0.1:9090"
	}
	secret := os.Getenv("MIHOMO_SECRET")
	cli := api.New(base, secret)
	app := ui.NewApp(cli)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
