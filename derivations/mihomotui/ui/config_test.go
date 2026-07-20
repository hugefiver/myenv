package ui

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/internal/termtext"
)

func TestConfigRefreshUsesOnlyGETConfigs(t *testing.T) {
	var configGETs atomic.Int32
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/configs" || r.Method != http.MethodGet {
			t.Fatalf("config refresh request = %s %s, want GET /configs", r.Method, r.URL.Path)
		}
		configGETs.Add(1)
		_, _ = w.Write([]byte(`{"mode":"rule"}`))
	}))
	app.active = tabConfig

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("r returned no refresh command")
	}
	msg := cmd()
	if _, ok := msg.(configLoadedMsg); !ok {
		t.Fatalf("r command returned %T, want configLoadedMsg", msg)
	}
	app.Update(msg)

	if got := configGETs.Load(); got != 1 {
		t.Fatalf("GET /configs calls = %d, want 1", got)
	}
	if strings.Contains(app.View(), "restart") || strings.Contains(helpFor(tabConfig, ""), "restart") {
		t.Fatalf("config UI still advertises restart: view=%q help=%q", app.View(), helpFor(tabConfig, ""))
	}
}

func TestConfigLoadErrorsRemainSanitizedAndVisible(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/configs" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "東京\x1b]52;c;CONFIG-OSC\x07 unavailable\x1bPCONFIG-DCS\x1b\\", http.StatusServiceUnavailable)
	}))
	app.active = tabConfig
	app.width, app.height = 80, 20

	app.Update(app.config.load()())
	if !app.statusErr || !strings.Contains(app.status, "東京") || strings.Contains(app.status, "CONFIG-") || strings.Contains(app.status, "\x1b") {
		t.Fatalf("initial config status = %q, error=%v", app.status, app.statusErr)
	}
	_, refresh := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	app.Update(refresh())
	if !app.statusErr || !strings.Contains(app.View(), "東京") || strings.Contains(termtext.Multiline(app.View()), "CONFIG-") {
		t.Fatalf("refreshed config error not sanitized and visible: status=%q view=%q", app.status, app.View())
	}
}
