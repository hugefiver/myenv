package ui

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

type groupsLoadedMsg struct {
	RequestID requestID
	Target    string
	Proxies   map[string]api.Proxy
	Groups    []api.Proxy
	Err       error
}

type selectedMsg struct {
	RequestID       requestID
	Target          string
	Group           string
	Node            string
	OperationID     requestID
	SelectionIntent bool
	Err             error
}

type groupDelayMsg struct {
	RequestID       requestID
	Target          string
	Group           string
	SelectionIntent bool
	Delay           api.GroupDelayResp
	Err             error
}

type groupNodeDelayMsg struct {
	RequestID       requestID
	Target          string
	Group           string
	Node            string
	SelectionIntent bool
	Delay           int
	Err             error
}

const groupsLoadTarget = "groups"

func groupNodeDelayTarget(group, node string) string { return group + "\x00" + node }

func (m *groupsModel) load() tea.Cmd {
	id := m.loadRequests.Begin(groupsLoadTarget)
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		type proxyResult struct {
			proxies map[string]api.Proxy
			err     error
		}
		type groupResult struct {
			groups []api.Proxy
			err    error
		}
		proxies := make(chan proxyResult, 1)
		groups := make(chan groupResult, 1)
		go func() {
			result, err := cli.Proxies(ctx)
			proxies <- proxyResult{proxies: result, err: err}
		}()
		go func() {
			result, err := cli.Groups(ctx)
			groups <- groupResult{groups: result, err: err}
		}()
		loadedProxies := <-proxies
		loadedGroups := <-groups
		return groupsLoadedMsg{
			RequestID: id,
			Target:    groupsLoadTarget,
			Proxies:   loadedProxies.proxies,
			Groups:    loadedGroups.groups,
			Err:       errors.Join(loadedProxies.err, loadedGroups.err),
		}
	}
}

func (m *groupsModel) selectNode(group, node string) tea.Cmd {
	m.delayRequests.Begin(group)
	return m.selectNodeForOperation(group, node, 0)
}

func (m *groupsModel) selectNodeForOperation(group, node string, operationID requestID) tea.Cmd {
	id := m.selectionRequests.Begin(group)
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, reqTimeout)
		defer cancel()
		err := cli.SelectProxy(ctx, group, node)
		return selectedMsg{RequestID: id, Target: group, Group: group, Node: node, OperationID: operationID, SelectionIntent: operationID != 0, Err: err}
	}
}

func (m *groupsModel) testGroup(name string, selectionIntent bool) tea.Cmd {
	id := m.delayRequests.Begin(name)
	group, ok := m.groupByName(name)
	cli := m.cli
	return func() tea.Msg {
		if !ok {
			return groupDelayMsg{RequestID: id, Target: name, Group: name, SelectionIntent: selectionIntent, Err: errors.New("group no longer exists")}
		}
		ctx, cancel := context.WithTimeout(m.ctx, 10*reqTimeout)
		defer cancel()
		if group.Type == "Selector" {
			delay, err := cli.GroupDelay(ctx, name, 5000)
			return groupDelayMsg{RequestID: id, Target: name, Group: name, SelectionIntent: selectionIntent, Delay: delay, Err: err}
		}
		delay, err := proxyDelays(ctx, cli, group.All)
		return groupDelayMsg{RequestID: id, Target: name, Group: name, SelectionIntent: selectionIntent, Delay: delay, Err: err}
	}
}

func proxyDelays(ctx context.Context, cli *api.Client, nodes []string) (api.GroupDelayResp, error) {
	type result struct {
		node  string
		delay int
		err   error
	}
	results := make(chan result, len(nodes))
	for _, node := range nodes {
		node := node
		go func() {
			delay, err := cli.ProxyDelay(ctx, node, 5000)
			results <- result{node: node, delay: delay, err: err}
		}()
	}
	delays := make(api.GroupDelayResp, len(nodes))
	var errs []error
	for range nodes {
		result := <-results
		if result.err != nil {
			delays[result.node] = -1
			errs = append(errs, fmt.Errorf("%s: %w", result.node, result.err))
			continue
		}
		delays[result.node] = result.delay
	}
	return delays, errors.Join(errs...)
}

func (m *groupsModel) testNode(group, node string) tea.Cmd {
	target := groupNodeDelayTarget(group, node)
	id := m.delayRequests.Begin(target)
	cli := m.cli
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*reqTimeout)
		defer cancel()
		delay, err := cli.ProxyDelay(ctx, node, 5000)
		return groupNodeDelayMsg{RequestID: id, Target: target, Group: group, Node: node, Delay: delay, Err: err}
	}
}

func (m *groupsModel) handleGroupsLoaded(result groupsLoadedMsg) (tea.Cmd, string, bool) {
	if !m.loadRequests.IsCurrent(result.RequestID, result.Target) {
		return nil, "", false
	}
	if result.Err != nil {
		return nil, "load failed: " + result.Err.Error(), true
	}
	m.ingest(result.Proxies, result.Groups)
	for name, proxy := range result.Proxies {
		if n := len(proxy.History); n > 0 {
			m.delay[name] = proxy.History[n-1].Delay
		}
	}
	return nil, "groups loaded", false
}

func (m *groupsModel) handleGroupDelay(result groupDelayMsg) (tea.Cmd, string, bool) {
	target := result.Target
	if target == "" {
		target = result.Group
	}
	if !m.delayRequests.IsCurrent(result.RequestID, target) {
		return nil, "", false
	}
	for node, delay := range result.Delay {
		m.delay[node] = delay
	}
	if result.Err != nil {
		return nil, "group test failed: " + result.Err.Error(), true
	}
	if !result.SelectionIntent || !m.selectableGroup(result.Group) {
		return nil, "group tested", false
	}
	group, ok := m.groupByName(result.Group)
	if !ok {
		return nil, "group no longer exists", true
	}
	best, bestDelay := fastestNode(group.All, result.Delay)
	if best == "" {
		return nil, "Auto: all nodes timed out", true
	}
	return m.selectNodeForOperation(result.Group, best, result.RequestID), "Auto: " + best + " (" + itoaMs(bestDelay) + ")", false
}

func fastestNode(nodes []string, delays api.GroupDelayResp) (string, int) {
	best := ""
	bestDelay := 0
	for _, node := range nodes {
		delay := delays[node]
		if delay <= 0 || (best != "" && delay >= bestDelay) {
			continue
		}
		best, bestDelay = node, delay
	}
	return best, bestDelay
}

func (m *groupsModel) handleSelected(result selectedMsg) (tea.Cmd, string, bool) {
	if !m.selectionRequests.IsCurrent(result.RequestID, result.Target) {
		return nil, "", false
	}
	if result.OperationID != 0 && !m.delayRequests.IsCurrent(result.OperationID, result.Group) {
		return nil, "", false
	}
	if result.Err != nil {
		return nil, "select failed: " + result.Err.Error(), true
	}
	return m.load(), "selected " + result.Node, false
}
