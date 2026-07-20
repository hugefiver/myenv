package ui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/rivo/uniseg"

	"mihomotui/api"
	"mihomotui/internal/termtext"
	"mihomotui/profiles"
)

func TestViewsSanitizeDynamicTextAndFitTerminalWidth(t *testing.T) {
	app := newAppForTest(t, nil)
	app.width, app.height = 40, 14

	host := "東京\x1b]52;c;HOST-OSC\x07"
	process := "进程\x1bPPROCESS-DCS\x1b\\"
	groupName := "组\x1b[31m"
	node := "节点\x1bPNODE-DCS\x1b\\"
	proxyName := "代理\x1b]0;PROXY-OSC\x07"
	profileName := "配置\x1bPPROFILE-DCS\x1b\\"
	profileFile := "文件\x1b]0;FILE-OSC\x07"
	profileURL := "https://例子.invalid/\x1bPURL-DCS\x1b\\"
	logPayload := "日志\x1b]52;c;LOG-OSC\x07"
	configMode := "rule\x1bPMODE-DCS\x1b\\"

	app.groups.rows = []groupRow{{name: groupName, now: node, typ: "Selector\x1b]0;TYPE-OSC\x07"}}
	app.groups.groups = map[string]api.Proxy{groupName: {Name: groupName, Type: "Selector", Now: node, All: []string{node}}}
	app.groups.all = map[string]api.Proxy{node: {Name: node, Type: "Shadowsocks\x1b[31m"}}
	app.proxies.ingest(map[string]api.Proxy{proxyName: {Name: proxyName, Type: "Shadowsocks\x1b[31m"}}, nil)
	app.conns.ingest(api.ConnectionsSnapshot{Connections: []api.Connection{{
		ID: "connection",
		Metadata: api.ConnMeta{
			Host:            host,
			ProcessPath:     process,
			DestinationPort: "443\x1b]0;PORT-OSC\x07",
			Network:         "tcp\x1bPNET-DCS\x1b\\",
		},
		Chains: []string{node},
		Rule:   "规则\x1b]0;RULE-OSC\x07",
	}}})
	app.logs.retain(api.LogEntry{Type: "warning\x1b]0;LOGTYPE-OSC\x07", Payload: logPayload})
	app.profiles.entries = []profiles.Entry{{
		ID:        "profile",
		Name:      profileName,
		Kind:      profiles.KindURL,
		File:      profileFile,
		URL:       profileURL,
		CreatedAt: time.Unix(0, 0),
		UpdatedAt: time.Unix(0, 0),
	}}
	app.config.cfg = &api.Config{Mode: configMode, LogLevel: "info\x1b]0;LEVEL-OSC\x07", Tun: api.Tun{Stack: "system\x1bPSTACK-DCS\x1b\\"}}
	app.groups.setConfig(app.config.cfg)
	app.setStatus("状态\x1b]52;c;STATUS-OSC\x07 error\x1bPERROR-DCS\x1b\\", true)

	for _, active := range []tab{tabGroups, tabProxies, tabConns, tabLogs, tabTraffic, tabProfiles, tabConfig} {
		app.active = active
		assertSafeRenderedView(t, app.View(), app.width)
	}
}

func TestViewsDoNotPanicAtNarrowOrNegativeWidths(t *testing.T) {
	app := newAppForTest(t, nil)
	for _, width := range []int{1, -1} {
		for _, active := range []tab{tabGroups, tabProxies, tabConns, tabLogs, tabTraffic, tabProfiles, tabConfig} {
			app.width, app.height, app.active = width, 1, active
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						t.Fatalf("View() panicked at width=%d tab=%d: %v", width, active, recovered)
					}
				}()
				_ = app.View()
			}()
		}
	}
}

func assertSafeRenderedView(t *testing.T, view string, width int) {
	t.Helper()
	if !utf8.ValidString(view) {
		t.Fatalf("View() is not valid UTF-8: %q", view)
	}
	plain := termtext.Multiline(view)
	if plain != termtext.Multiline(plain) || strings.ContainsRune(plain, '\x1b') {
		t.Fatalf("View() retains an untrusted terminal control sequence: %q", view)
	}
	for _, payload := range []string{
		"HOST-OSC", "PROCESS-DCS", "NODE-DCS", "PROXY-OSC", "PROFILE-DCS",
		"FILE-OSC", "URL-DCS", "LOG-OSC", "MODE-DCS", "LEVEL-OSC", "STACK-DCS", "STATUS-OSC", "ERROR-DCS",
	} {
		if strings.Contains(plain, payload) {
			t.Fatalf("View() exposes control payload %q: %q", payload, view)
		}
	}
	for _, line := range strings.Split(plain, "\n") {
		if cells := viewCellWidth(line); cells > width {
			t.Fatalf("View() line has %d cells, exceeds %d: %q", cells, width, line)
		}
	}
}

func viewCellWidth(text string) int {
	width := 0
	for graphemes := uniseg.NewGraphemes(text); graphemes.Next(); {
		width += graphemes.Width()
	}
	return width
}
