package ui

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

func TestGroupAutoSelectStaysWithItsRequestAndGroup(t *testing.T) {
	type selection struct {
		group string
		node  string
	}
	var selections []selection
	m := newGroupsTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/group/Alpha/delay":
			_, _ = w.Write([]byte(`{"a":20,"b":10}`))
		case r.URL.Path == "/group/Beta/delay":
			_, _ = w.Write([]byte(`{"x":30,"y":15}`))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/proxies/"):
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode SelectProxy body: %v", err)
				return
			}
			selections = append(selections, selection{
				group: strings.TrimPrefix(r.URL.Path, "/proxies/"),
				node:  body.Name,
			})
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	m.rows = []groupRow{{name: "Alpha", typ: "Selector"}, {name: "Beta", typ: "Selector"}}
	m.all = map[string]api.Proxy{
		"Alpha": {Name: "Alpha", Type: "Selector", All: []string{"a", "b"}},
		"Beta":  {Name: "Beta", Type: "Selector", All: []string{"x", "y"}},
	}

	alphaCmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	betaCmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})

	if selectionCmd, _, _ := m.Update(betaCmd()); selectionCmd != nil {
		t.Fatal("test-only Beta result generated a selection command")
	}
	selectionCmd, _, _ := m.Update(alphaCmd())
	if selectionCmd == nil {
		t.Fatal("Alpha auto-select result did not generate a selection command")
	}
	m.Update(selectionCmd())
	if got, want := selections, []selection{{group: "Alpha", node: "b"}}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("selections = %#v, want %#v", got, want)
	}
}

func TestStaleGroupAutoSelectCannotOverrideNewerRequest(t *testing.T) {
	m := newGroupsTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/group/Alpha/delay" {
			_, _ = w.Write([]byte(`{"old":50,"new":10}`))
			return
		}
		http.NotFound(w, r)
	}))
	m.rows = []groupRow{{name: "Alpha", typ: "Selector"}}
	m.all = map[string]api.Proxy{"Alpha": {Name: "Alpha", Type: "Selector", All: []string{"old", "new"}}}

	staleCmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	currentCmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	if selectionCmd, _, _ := m.Update(currentCmd()); selectionCmd == nil {
		t.Fatal("newest auto-select request did not select")
	}
	if selectionCmd, _, _ := m.Update(staleCmd()); selectionCmd != nil {
		t.Fatal("stale auto-select request generated a selection command")
	}
}

func TestManualSelectionInvalidatesOlderAutoSelect(t *testing.T) {
	type selection struct {
		group string
		node  string
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var selections []selection
	m := newGroupsTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/group/G/delay":
			close(started)
			<-release
			_, _ = w.Write([]byte(`{"Alpha":50,"Beta":10}`))
		case r.Method == http.MethodPut && r.URL.Path == "/proxies/G":
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode selection: %v", err)
				return
			}
			mu.Lock()
			selections = append(selections, selection{group: "G", node: body.Name})
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	m.rows = []groupRow{{name: "G", typ: "Selector"}}
	m.all = map[string]api.Proxy{"G": {Name: "G", Type: "Selector", All: []string{"Alpha", "Beta"}}}
	m.focus = paneRight

	autoCmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
	autoResult := make(chan tea.Msg, 1)
	go func() { autoResult <- autoCmd() }()
	<-started
	manualCmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if manualCmd == nil {
		t.Fatal("manual selection returned nil command")
	}
	close(release)
	if staleSelect, _, _ := m.Update(<-autoResult); staleSelect != nil {
		t.Fatal("stale auto test generated a selection command after manual selection")
	}
	m.Update(manualCmd())
	mu.Lock()
	got := append([]selection(nil), selections...)
	mu.Unlock()
	want := []selection{{group: "G", node: "Alpha"}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("selections = %#v, want %#v", got, want)
	}
}

func TestAutomaticGroupsUsePerNodeDelaysWithoutSelection(t *testing.T) {
	for _, typ := range []string{"URLTest", "Fallback"} {
		t.Run(typ, func(t *testing.T) {
			var groupDelayCalls, selectCalls int
			var nodeDelayCalls []string
			var callsMu sync.Mutex
			m := newGroupsTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callsMu.Lock()
				defer callsMu.Unlock()
				switch {
				case strings.HasPrefix(r.URL.Path, "/group/"):
					groupDelayCalls++
					http.Error(w, "group delay must not be called", http.StatusInternalServerError)
				case strings.HasSuffix(r.URL.Path, "/delay"):
					nodeDelayCalls = append(nodeDelayCalls, r.URL.Path)
					_, _ = w.Write([]byte(`{"delay":12}`))
				case r.Method == http.MethodPut:
					selectCalls++
					w.WriteHeader(http.StatusNoContent)
				default:
					http.NotFound(w, r)
				}
			}))
			m.rows = []groupRow{{name: typ, typ: typ}}
			m.all = map[string]api.Proxy{typ: {Name: typ, Type: typ, All: []string{"first", "second"}}}

			cmd, _, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})
			if cmd == nil {
				t.Fatal("T returned nil command")
			}
			if next, _, _ := m.Update(cmd()); next != nil {
				t.Fatal("automatic group testing generated SelectProxy command")
			}
			callsMu.Lock()
			defer callsMu.Unlock()
			if groupDelayCalls != 0 {
				t.Fatalf("group delay calls = %d, want 0", groupDelayCalls)
			}
			if selectCalls != 0 {
				t.Fatalf("SelectProxy calls = %d, want 0", selectCalls)
			}
			if got, want := len(nodeDelayCalls), 2; got != want {
				t.Fatalf("per-node delay calls = %#v, want %d", nodeDelayCalls, want)
			}
		})
	}
}
