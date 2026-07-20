package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	"mihomotui/profiles"
	"mihomotui/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	base := os.Getenv("MIHOMO_API")
	if base == "" {
		base = "http://127.0.0.1:9090"
	}
	secret := os.Getenv("MIHOMO_SECRET")
	client, err := api.New(base, secret)
	if err != nil {
		return fmt.Errorf("configure Mihomo API: %w", err)
	}
	dataDir, err := profiles.ResolveDataDir()
	if err != nil {
		return fmt.Errorf("resolve profile data directory: %w", err)
	}
	store, err := profiles.Open(dataDir, client)
	if err != nil {
		return fmt.Errorf("open profile store: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := ui.NewApp(ctx, client, store)
	defer app.Close()
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run UI: %w", err)
	}
	return nil
}
