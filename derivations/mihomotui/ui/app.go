package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
)

const reqTimeout = 3 * time.Second

type forwardMsg struct{ inner tea.Msg }

type App struct {
	cli       *api.Client
	active    tab
	width     int
	height    int
	status    string
	statusErr bool

	groups  *groupsModel
	proxies *proxiesModel
	conns   *connsModel
	logs    *logsModel
	traffic *trafficModel
	config  *configModel

	bus chan tea.Msg
}

func NewApp(cli *api.Client) *App {
	return &App{
		cli:     cli,
		groups:  newGroupsModel(cli),
		proxies: newProxiesModel(cli),
		conns:   newConnsModel(cli),
		logs:    newLogsModel(cli),
		traffic: newTrafficModel(cli),
		config:  newConfigModel(cli),
		bus:     make(chan tea.Msg, 64),
	}
}

func (a *App) send(m tea.Msg) {
	select {
	case a.bus <- m:
	default:
	}
}

func (a *App) waitBus() tea.Cmd {
	return func() tea.Msg {
		m, ok := <-a.bus
		if !ok {
			return nil
		}
		return forwardMsg{inner: m}
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
	case tabConfig:
		return a.config.load()
	}
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if fm, ok := msg.(forwardMsg); ok {
		cmd := a.dispatchInner(fm.inner)
		return a, tea.Batch(cmd, a.waitBus())
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = v.Width, v.Height
		a.groups.width, a.groups.height = v.Width, v.Height
		a.proxies.width, a.proxies.height = v.Width, v.Height
		a.conns.width, a.conns.height = v.Width, v.Height
		a.logs.width, a.logs.height = v.Width, v.Height
		return a, nil
	case tea.KeyMsg:
		if cmd, handled := a.handleGlobalKey(v); handled {
			return a, cmd
		}
		return a, a.dispatchInner(v)
	}
	return a, a.dispatchInner(msg)
}

func (a *App) handleGlobalKey(k tea.KeyMsg) (tea.Cmd, bool) {
	switch k.String() {
	case "ctrl+c", "q":
		a.conns.stop()
		a.logs.stop()
		a.traffic.stop()
		return tea.Quit, true
	case "1":
		return a.switchTab(tabGroups), true
	case "2":
		return a.switchTab(tabProxies), true
	case "3":
		return a.switchTab(tabConns), true
	case "4":
		return a.switchTab(tabLogs), true
	case "5":
		return a.switchTab(tabTraffic), true
	case "6":
		return a.switchTab(tabConfig), true
	case "tab":
		return a.switchTab((a.active + 1) % tabCount), true
	case "shift+tab":
		return a.switchTab((a.active + tabCount - 1) % tabCount), true
	}
	return nil, false
}

func (a *App) dispatchInner(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var status string
	var isErr bool
	switch a.active {
	case tabGroups:
		cmd, status, isErr = a.groups.Update(msg)
	case tabProxies:
		cmd, status, isErr = a.proxies.Update(msg)
	case tabConns:
		cmd, status, isErr = a.conns.Update(msg)
	case tabLogs:
		cmd, status, isErr = a.logs.Update(msg)
	case tabTraffic:
		cmd, status, isErr = a.traffic.Update(msg)
	case tabConfig:
		cmd, status, isErr = a.config.Update(msg)
	}
	if status != "" {
		a.status = status
		a.statusErr = isErr
	}
	return cmd
}

func (a *App) View() string {
	if a.width == 0 {
		return "loading..."
	}
	top := renderTabs(a.active, a.width)
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
	case tabConfig:
		body = a.config.View()
	}
	detail := a.active == tabGroups && a.groups.inDetail
	help := renderHelp(a.width, helpFor(a.active, detail))
	status := renderStatus(a.width, a.status, a.statusErr)
	bodyHeight := a.height - lipgloss.Height(top) - lipgloss.Height(help) - lipgloss.Height(status)
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	bodyBox := lipgloss.NewStyle().Width(a.width).Height(bodyHeight).Render(body)
	return strings.Join([]string{top, bodyBox, status, help}, "\n")
}
