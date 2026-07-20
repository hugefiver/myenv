package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
	"mihomotui/internal/termtext"
	"mihomotui/profiles"
)

const reqTimeout = 3 * time.Second

type App struct {
	ctx       context.Context
	cancel    context.CancelFunc
	cli       *api.Client
	bus       *messageBus
	warning   string
	active    tab
	width     int
	height    int
	status    string
	statusErr bool
	configErr string

	groups   *groupsModel
	proxies  *proxiesModel
	conns    *connsModel
	logs     *logsModel
	traffic  *trafficModel
	profiles *profilesModel
	config   *configModel
}

func NewApp(parent context.Context, cli *api.Client, store *profiles.Store) *App {
	ctx, cancel := context.WithCancel(parent)
	bus := newMessageBus()
	return &App{
		ctx:      ctx,
		cancel:   cancel,
		cli:      cli,
		bus:      bus,
		warning:  termtext.SingleLine(cli.Warning()),
		groups:   newGroupsModel(ctx, cli),
		proxies:  newProxiesModel(ctx, cli),
		conns:    newConnsModel(ctx, cli, bus),
		logs:     newLogsModel(ctx, cli, bus),
		traffic:  newTrafficModel(ctx, cli, bus),
		profiles: newProfilesModel(ctx, cli, store),
		config:   newConfigModel(ctx, cli),
	}
}

func (a *App) Close() {
	a.stopStreams()
	a.cancel()
}

func (a *App) send(m tea.Msg) {
	switch v := m.(type) {
	case connsSnapMsg:
		if v.err != nil || v.end {
			a.bus.SendControl(a.ctx, m)
			return
		}
		a.bus.SendLatest(streamConnections, v.generation, m)
	case logEntryMsg:
		if v.err != nil || v.end {
			a.bus.SendControl(a.ctx, m)
			return
		}
		a.bus.SendLog(v.generation, m)
	case trafficMsg:
		if v.err != nil || v.end {
			a.bus.SendControl(a.ctx, m)
			return
		}
		a.bus.SendLatest(streamTraffic, v.generation, m)
	default:
		a.bus.SendControl(a.ctx, m)
	}
}

func (a *App) waitBus() tea.Cmd {
	return func() tea.Msg {
		return a.bus.Wait(a.ctx)
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.groups.load(),
		a.config.load(),
		a.waitBus(),
	)
}

func (a *App) switchTab(t tab) tea.Cmd {
	if a.active == t {
		return nil
	}
	switch a.active {
	case tabConns:
		a.conns.stop()
	case tabLogs:
		a.logs.stop()
	case tabTraffic:
		a.traffic.stop()
	}
	a.active = t
	switch t {
	case tabGroups:
		return a.groups.load()
	case tabProxies:
		return a.proxies.load()
	case tabConns:
		return a.conns.start(a.send)
	case tabLogs:
		return a.logs.start(a.send)
	case tabTraffic:
		return a.traffic.start(a.send)
	case tabProfiles:
		return a.profiles.load()
	case tabConfig:
		return a.config.load()
	}
	return nil
}

func (a *App) bodyDims() (int, int) {
	width := a.width
	if width < 1 {
		width = 1
	}
	top := renderTabs(a.active, width)
	help := renderHelp(width, "x")
	statusText, statusErr := a.statusLine()
	status := renderStatus(width, statusText, statusErr)
	bh := a.height - lipgloss.Height(top) - lipgloss.Height(help) - lipgloss.Height(status)
	if bh < 1 {
		bh = 1
	}
	bw := width - 4
	if bw < 1 {
		bw = 1
	}
	innerH := bh - 2
	if innerH < 1 {
		innerH = 1
	}
	return bw, innerH
}

func (a *App) View() string {
	if a.width <= 0 {
		return "loading..."
	}
	width := a.width
	top := renderTabs(a.active, width)
	help := renderHelp(width, helpFor(a.active, a.groups.helpMode()))
	statusText, statusErr := a.statusLine()
	status := renderStatus(width, statusText, statusErr)
	bodyHeight := a.height - lipgloss.Height(top) - lipgloss.Height(help) - lipgloss.Height(status)
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}
	innerH := bodyHeight - 2
	if innerH < 1 {
		innerH = 1
	}
	a.groups.width, a.groups.height = innerW, innerH
	a.proxies.width, a.proxies.height = innerW, innerH
	a.conns.width, a.conns.height = innerW, innerH
	a.logs.width, a.logs.height = innerW, innerH
	a.traffic.width, a.traffic.height = innerW, innerH
	a.profiles.width, a.profiles.height = innerW, innerH
	a.config.width, a.config.height = innerW, innerH

	var body string
	switch a.active {
	case tabGroups:
		body = a.groups.View()
	case tabProxies:
		body = a.proxies.View()
	case tabConns:
		body = a.conns.View()
	case tabLogs:
		body = a.logs.View()
	case tabTraffic:
		body = a.traffic.View()
	case tabProfiles:
		body = a.profiles.View()
	case tabConfig:
		body = a.config.View()
	}
	frameWidth := width - 2
	if frameWidth < 1 {
		frameWidth = 1
	}
	frameHeight := bodyHeight - 2
	if frameHeight < 1 {
		frameHeight = 1
	}
	bodyBox := stFrame.Width(frameWidth).Height(frameHeight).Render(body)
	return strings.Join([]string{top, bodyBox, status, help}, "\n")
}
