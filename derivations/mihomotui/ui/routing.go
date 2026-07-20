package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/internal/termtext"
)

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		a.stopStreams()
		a.Close()
		return a, tea.Quit
	}
	switch v := msg.(type) {
	case busMsg:
		if v.LogDrops != 0 {
			a.logs.addDrops(v.LogDrops)
		}
		if v.Inner == nil {
			return a, nil
		}
		ownerCmd := a.dispatchOwned(v.Inner)
		return a, tea.Batch(ownerCmd, a.waitBus())
	case tea.WindowSizeMsg:
		a.width, a.height = v.Width, v.Height
		bw, bh := a.bodyDims()
		a.groups.width, a.groups.height = bw, bh
		a.proxies.width, a.proxies.height = bw, bh
		a.conns.width, a.conns.height = bw, bh
		a.logs.width, a.logs.height = bw, bh
		a.traffic.width, a.traffic.height = bw, bh
		a.profiles.width, a.profiles.height = bw, bh
		a.config.width, a.config.height = bw, bh
		return a, nil
	case tea.KeyMsg:
		if cmd, handled := a.handleGlobalKey(v); handled {
			return a, cmd
		}
		return a, a.dispatchActive(v)
	case tea.MouseMsg:
		return a, a.dispatchActive(v)
	}
	return a, a.dispatchOwned(msg)
}

func (a *App) handleGlobalKey(k tea.KeyMsg) (tea.Cmd, bool) {
	if a.active == tabGroups && a.groups.searching {
		return nil, false
	}
	if a.active == tabProfiles && a.profiles.importing {
		return nil, false
	}
	switch k.String() {
	case "q":
		a.stopStreams()
		a.Close()
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
		return a.switchTab(tabProfiles), true
	case "7":
		return a.switchTab(tabConfig), true
	case "shift+tab":
		return a.switchTab((a.active + tabCount - 1) % tabCount), true
	}
	return nil, false
}

func (a *App) stopStreams() {
	a.conns.stop()
	a.logs.stop()
	a.traffic.stop()
}

func (a *App) dispatchActive(msg tea.Msg) tea.Cmd {
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
	case tabProfiles:
		cmd, status, isErr = a.profiles.Update(msg)
	case tabConfig:
		cmd, status, isErr = a.config.Update(msg)
	}
	a.setStatus(status, isErr)
	return cmd
}

func (a *App) dispatchOwned(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var status string
	var isErr bool
	switch v := msg.(type) {
	case groupsLoadedMsg, selectedMsg, groupDelayMsg, groupNodeDelayMsg:
		cmd, status, isErr = a.groups.Update(v)
	case proxiesAllMsg, proxyDelayMsg:
		cmd, status, isErr = a.proxies.Update(v)
	case profileResultMsg:
		cmd, status, isErr = a.profiles.Update(v)
	case configLoadedMsg:
		current := a.config.loadRequests.IsCurrent(v.RequestID, v.Target)
		cmd, status, isErr = a.config.Update(v)
		if current {
			if v.Err != nil {
				a.configErr = termtext.SingleLine("config: " + v.Err.Error())
			} else {
				a.configErr = ""
			}
		}
		if current && v.Err == nil && v.Config != nil && a.config.cfg == v.Config {
			a.groups.setConfig(v.Config)
		}
	case configChangedMsg:
		cmd, status, isErr = a.config.Update(v)
	case connsSnapMsg:
		cmd, status, isErr = a.conns.Update(v)
	case logEntryMsg:
		cmd, status, isErr = a.logs.Update(v)
	case trafficMsg:
		cmd, status, isErr = a.traffic.Update(v)
	default:
		return nil
	}
	a.setStatus(status, isErr)
	return cmd
}

func (a *App) setStatus(status string, isErr bool) {
	status = termtext.SingleLine(status)
	if status != "" {
		a.status = status
		a.statusErr = isErr
	}
}

func (a *App) statusLine() (string, bool) {
	status, isErr := a.status, a.statusErr
	if a.configErr != "" {
		status, isErr = a.configErr, true
	}
	if a.warning == "" {
		return status, isErr
	}
	if status == "" {
		return a.warning, false
	}
	return status + " | " + a.warning, isErr
}
