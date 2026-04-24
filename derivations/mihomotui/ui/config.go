package ui

import (
	"context"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

type configLoadedMsg struct {
	cfg *api.Config
	err error
}

type configChangedMsg struct {
	what string
	err  error
}

type restartMsg struct {
	out string
	err error
}

type configModel struct {
	cli *api.Client
	cfg *api.Config
}

func newConfigModel(cli *api.Client) *configModel {
	return &configModel{cli: cli}
}

func (m *configModel) load() tea.Cmd {
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
		defer cancel()
		c, err := cli.Configs(ctx)
		return configLoadedMsg{cfg: c, err: err}
	}
}

func (m *configModel) cycleMode() tea.Cmd {
	if m.cfg == nil {
		return m.load()
	}
	next := nextMode(m.cfg.Mode)
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
		defer cancel()
		err := cli.SetMode(ctx, next)
		return configChangedMsg{what: "mode=" + next, err: err}
	}
}

func nextMode(m string) string {
	switch strings.ToLower(m) {
	case "rule":
		return "global"
	case "global":
		return "direct"
	default:
		return "rule"
	}
}

func (m *configModel) toggleTun() tea.Cmd {
	if m.cfg == nil {
		return m.load()
	}
	next := !m.cfg.Tun.Enable
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
		defer cancel()
		err := cli.SetTun(ctx, next)
		return configChangedMsg{what: "tun=" + boolStr(next), err: err}
	}
}

func boolStr(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (m *configModel) restart() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*reqTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "systemctl", "restart", "mihomo-boot.service")
		out, err := cmd.CombinedOutput()
		return restartMsg{out: strings.TrimSpace(string(out)), err: err}
	}
}

func (m *configModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case configLoadedMsg:
		if v.err != nil {
			return nil, "config: " + v.err.Error(), true
		}
		m.cfg = v.cfg
		return nil, "config loaded", false
	case configChangedMsg:
		if v.err != nil {
			return nil, v.what + " failed: " + v.err.Error(), true
		}
		return m.load(), v.what + " ok", false
	case restartMsg:
		if v.err != nil {
			msg := "restart failed: " + v.err.Error()
			if v.out != "" {
				msg += " | " + v.out
			}
			return nil, msg, true
		}
		return m.load(), "restarted: " + v.out, false
	case tea.KeyMsg:
		switch v.String() {
		case "m":
			return m.cycleMode(), "switching mode", false
		case "t":
			return m.toggleTun(), "toggling tun", false
		case "r":
			return m.restart(), "restarting service", false
		case "R":
			return m.load(), "reloading", false
		}
	}
	return nil, "", false
}

func (m *configModel) View() string {
	var b strings.Builder
	b.WriteString(stTitle.Render("Config") + "\n\n")
	if m.cfg == nil {
		b.WriteString(stMuted.Render("loading..."))
		return b.String()
	}
	b.WriteString(stBase.Render("mode      ") + stOK.Render(m.cfg.Mode) + stMuted.Render("   (m to cycle)") + "\n")
	b.WriteString(stBase.Render("tun       ") + stOK.Render(boolStr(m.cfg.Tun.Enable)))
	if m.cfg.Tun.Stack != "" {
		b.WriteString(stMuted.Render(" stack=" + m.cfg.Tun.Stack))
	}
	b.WriteString(stMuted.Render("   (t to toggle)") + "\n")
	b.WriteString(stBase.Render("log-level ") + stMuted.Render(m.cfg.LogLevel) + "\n")
	b.WriteString(stBase.Render("ports     ") + stMuted.Render("http=") + intStr(m.cfg.Port))
	b.WriteString(stMuted.Render("  socks=") + intStr(m.cfg.SocksPort))
	b.WriteString(stMuted.Render("  mixed=") + intStr(m.cfg.MixedPort) + "\n")
	b.WriteString(stBase.Render("allow-lan ") + stMuted.Render(boolStr(m.cfg.AllowLan)) + "\n\n")
	b.WriteString(stMuted.Render("[r] restart mihomo-boot.service   [R] reload config"))
	return b.String()
}

func intStr(n int) string { return formatInt(n) }
