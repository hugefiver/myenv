package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

type configLoadedMsg struct {
	RequestID requestID
	Target    string
	Config    *api.Config
	Err       error
}

type configChangedMsg struct {
	RequestID requestID
	Target    string
	What      string
	Err       error
}

type configModel struct {
	ctx              context.Context
	cli              *api.Client
	loadRequests     requestTracker
	mutationRequests requestTracker
	cfg              *api.Config
	width            int
	height           int
}

const configLoadTarget = "config"

func newConfigModel(ctx context.Context, cli *api.Client) *configModel {
	return &configModel{ctx: ctx, cli: cli}
}

func (m *configModel) load() tea.Cmd {
	id := m.loadRequests.Begin(configLoadTarget)
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		c, err := cli.Configs(ctx)
		return configLoadedMsg{RequestID: id, Target: configLoadTarget, Config: c, Err: err}
	}
}

func (m *configModel) cycleMode() tea.Cmd {
	if m.cfg == nil {
		return m.load()
	}
	next := nextMode(m.cfg.Mode)
	id := m.mutationRequests.Begin("mode")
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		err := cli.SetMode(ctx, next)
		return configChangedMsg{RequestID: id, Target: "mode", What: "mode=" + next, Err: err}
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
	id := m.mutationRequests.Begin("tun")
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		err := cli.SetTun(ctx, next)
		return configChangedMsg{RequestID: id, Target: "tun", What: "tun=" + boolStr(next), Err: err}
	}
}

func boolStr(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (m *configModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case configLoadedMsg:
		if !m.loadRequests.IsCurrent(v.RequestID, v.Target) {
			return nil, "", false
		}
		if v.Err != nil {
			return nil, "config: " + v.Err.Error(), true
		}
		m.cfg = v.Config
		return nil, "config loaded", false
	case configChangedMsg:
		if !m.mutationRequests.IsCurrent(v.RequestID, v.Target) {
			return nil, "", false
		}
		if v.Err != nil {
			return nil, v.What + " failed: " + v.Err.Error(), true
		}
		return m.load(), v.What + " ok", false
	case tea.KeyMsg:
		switch v.String() {
		case "m":
			return m.cycleMode(), "switching mode", false
		case "t":
			return m.toggleTun(), "toggling tun", false
		case "r":
			return m.load(), "refreshing config", false
		}
	}
	return nil, "", false
}

func (m *configModel) View() string {
	var b strings.Builder
	b.WriteString(renderTextLine(m.width, segment(stTitle, "Config")) + "\n\n")
	if m.cfg == nil {
		b.WriteString(renderTextLine(m.width, segment(stMuted, "loading...")))
		return b.String()
	}
	b.WriteString(renderTextLine(m.width,
		segment(stBase, "mode      "),
		segment(stOK, m.cfg.Mode),
		segment(stMuted, "   (m to cycle)"),
	) + "\n")
	tun := []textSegment{segment(stBase, "tun       "), segment(stOK, boolStr(m.cfg.Tun.Enable))}
	if m.cfg.Tun.Stack != "" {
		tun = append(tun, segment(stMuted, " stack="+m.cfg.Tun.Stack))
	}
	tun = append(tun, segment(stMuted, "   (t to toggle)"))
	b.WriteString(renderTextLine(m.width, tun...) + "\n")
	b.WriteString(renderTextLine(m.width, segment(stBase, "log-level "), segment(stMuted, m.cfg.LogLevel)) + "\n")
	b.WriteString(renderTextLine(m.width,
		segment(stBase, "ports     "),
		segment(stMuted, "http="), segment(stPlain, intStr(m.cfg.Port)),
		segment(stMuted, "  socks="), segment(stPlain, intStr(m.cfg.SocksPort)),
		segment(stMuted, "  mixed="), segment(stPlain, intStr(m.cfg.MixedPort)),
	) + "\n")
	b.WriteString(renderTextLine(m.width, segment(stBase, "allow-lan "), segment(stMuted, boolStr(m.cfg.AllowLan))) + "\n\n")
	b.WriteString(renderTextLine(m.width, segment(stMuted, "[r] refresh config")))
	return b.String()
}

func intStr(n int) string { return formatInt(n) }
