package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

func TestProxyDelayUsesRawNameWithSanitizedDisplay(t *testing.T) {
	rawName := "Node\x1b]0;hidden\x07-ID"
	var target string
	m := newProxiesTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"delay":17}`))
	}))
	m.width, m.height = 80, 20
	m.ingest(map[string]api.Proxy{
		rawName: {Name: rawName, Type: "Shadowsocks"},
	}, nil)
	if view := m.View(); strings.Contains(view, "hidden") || strings.Contains(view, "\x1b]") {
		t.Fatalf("proxy view is not sanitized: %q", view)
	}

	cmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Fatal("delay test returned nil command")
	}
	m.Update(cmd())
	want := "/proxies/" + url.PathEscape(rawName) + "/delay"
	if target != want {
		t.Fatalf("delay target = %q, want raw proxy path %q", target, want)
	}
}

func TestProxiesLoadExcludesNamesReportedByGroup(t *testing.T) {
	var groupCalls atomic.Int32
	m := newProxiesTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxies":
			_, _ = w.Write([]byte(`{"proxies":{"Relay":{"name":"Relay","type":"Shadowsocks"},"leaf":{"name":"leaf","type":"Shadowsocks"}}}`))
		case "/group":
			groupCalls.Add(1)
			_, _ = w.Write([]byte(`{"proxies":[{"name":"Relay","type":"FutureGroup"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))

	msg := m.load()()
	m.Update(msg)
	if got := groupCalls.Load(); got != 1 {
		t.Fatalf("/group calls = %d, want 1", got)
	}
	if len(m.rows) != 1 || m.rows[0].name != "leaf" {
		t.Fatalf("leaf rows = %#v, want only leaf", m.rows)
	}
}

func newProxiesTestModel(t *testing.T, handler http.Handler) *proxiesModel {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := api.New(server.URL, "")
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	t.Cleanup(server.Close)
	return newProxiesModel(context.Background(), client)
}
