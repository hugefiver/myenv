package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

func TestGroupsLoadUsesGroupIdentityWhileInactive(t *testing.T) {
	var groupCalls atomic.Int32
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxies":
			_, _ = w.Write([]byte(`{"proxies":{"node":{"name":"node","type":"Shadowsocks"}}}`))
		case "/group":
			groupCalls.Add(1)
			_, _ = w.Write([]byte(`{"proxies":[{"name":"Relay","type":"Relay","now":"node","all":["node"]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	app.active = tabTraffic

	app.Update(app.groups.load()())

	if got := groupCalls.Load(); got != 1 {
		t.Fatalf("/group calls = %d, want 1", got)
	}
	if len(app.groups.rows) != 1 || app.groups.rows[0].name != "Relay" {
		t.Fatalf("rows = %#v, want Relay from /group", app.groups.rows)
	}
}

func TestGroupsEmptyNavigationKeepsNeutralCursors(t *testing.T) {
	keys := []tea.KeyMsg{
		{Type: tea.KeyHome},
		{Type: tea.KeyEnd},
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyLeft},
		{Type: tea.KeyRight},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("t")},
		{Type: tea.KeyRunes, Runes: []rune("T")},
	}
	for _, tc := range []struct {
		name  string
		setup func(*groupsModel)
	}{
		{
			name: "no groups",
			setup: func(m *groupsModel) {
				m.leftCursor, m.rightCursor = -1, -1
			},
		},
		{
			name: "no nodes",
			setup: func(m *groupsModel) {
				m.rows = []groupRow{{name: "Alpha", typ: "Selector"}}
				m.all = map[string]api.Proxy{"Alpha": {Name: "Alpha", Type: "Selector"}}
				m.leftCursor, m.rightCursor, m.focus = -1, -1, paneRight
			},
		},
		{
			name: "zero search results",
			setup: func(m *groupsModel) {
				m.rows = []groupRow{{name: "Alpha", typ: "Selector"}}
				m.all = map[string]api.Proxy{"Alpha": {Name: "Alpha", Type: "Selector", All: []string{"node"}}}
				m.search = "missing"
				m.leftCursor, m.rightCursor = -1, -1
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newGroupsModel(context.Background(), nil)
			m.width, m.height = 80, 24
			tc.setup(m)
			for _, key := range keys {
				m.Update(key)
				_ = m.View()
				if m.leftCursor < 0 || m.rightCursor < 0 {
					t.Fatalf("key %q left=%d right=%d, want non-negative cursors", key.String(), m.leftCursor, m.rightCursor)
				}
			}
		})
	}
}

func TestGroupSearchAcceptsPasteIMEAndRuneBackspace(t *testing.T) {
	m := newGroupsModel(context.Background(), nil)
	m.searching = true
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("東京"), Paste: true})
	if got, want := m.search, "東京"; got != want {
		t.Fatalf("paste search = %q, want %q", got, want)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got, want := m.search, "東"; got != want {
		t.Fatalf("backspace search = %q, want %q", got, want)
	}

	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	app.active = tabGroups
	app.groups.searching = true
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q"), Paste: true})
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("pasted q quit the app instead of becoming search text")
		}
	}
	if got := app.groups.search; got != "q" {
		t.Fatalf("pasted shortcut search = %q, want q", got)
	}
}

func TestRenderedGroupDelayUsesGroupName(t *testing.T) {
	m := newGroupsModel(context.Background(), nil)
	m.width, m.height = 80, 24
	m.rows = []groupRow{{name: "Alpha", now: "node", typ: "Selector"}}
	m.all = map[string]api.Proxy{"Alpha": {Name: "Alpha", Type: "Selector", Now: "node"}}
	m.delay["Alpha"] = 42
	m.delay["node"] = 7

	if got := m.View(); !strings.Contains(got, "42ms") {
		t.Fatalf("View() = %q, want group delay keyed by Alpha", got)
	}
}

func newGroupsTestModel(t *testing.T, handler http.Handler) *groupsModel {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := api.New(server.URL, "")
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	t.Cleanup(server.Close)
	return newGroupsModel(context.Background(), client)
}
