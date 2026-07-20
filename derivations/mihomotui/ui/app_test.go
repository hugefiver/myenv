package ui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	profileStore "mihomotui/profiles"
)

func TestAsyncProfileResultRoutesWhileInactive(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	app.active = tabGroups
	request, ok := app.profiles.beginProfileRequest(profileRequestImportFile, "profile.yaml")
	if !ok {
		t.Fatal("beginProfileRequest() rejected an idle model")
	}

	app.Update(profileResultMsg{
		request:   request,
		entry:     profileStore.Entry{Name: "profile.yaml"},
		persisted: true,
	})

	if app.profiles.busy {
		t.Fatal("inactive profile result left the profile model busy")
	}
	if app.status != "imported: profile.yaml" || app.statusErr {
		t.Fatalf("global status = %q, error = %v", app.status, app.statusErr)
	}
}

func TestAsyncProfileStatusIsSanitized(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	app.active = tabGroups
	request, ok := app.profiles.beginProfileRequest(profileRequestActivation, "profile-id")
	if !ok {
		t.Fatal("beginProfileRequest() rejected an idle model")
	}

	app.Update(profileResultMsg{
		request: request,
		entry:   profileStore.Entry{Name: "profile\x1b]0;hidden\x07-name\x9b31m"},
		err:     errors.New("request\x1b[31m-failed\x1b[0m\r"),
	})

	if !app.statusErr {
		t.Fatal("sanitized async failure lost error status")
	}
	if !strings.Contains(app.status, "profile-name") || !strings.Contains(app.status, "request-failed") {
		t.Fatalf("sanitized status lost visible text: %q", app.status)
	}
	for _, r := range app.status {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			t.Fatalf("status retains control rune %U: %q", r, app.status)
		}
	}
}

func TestInitialConfigErrorIsGlobalWhileInactive(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/configs" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "controller unavailable", http.StatusServiceUnavailable)
	}))
	app.active = tabGroups

	msg := app.config.load()()
	app.Update(msg)

	if !app.statusErr || !strings.Contains(app.status, "config:") || !strings.Contains(app.status, "controller unavailable") {
		t.Fatalf("inactive config error status = %q, error = %v", app.status, app.statusErr)
	}
}

func TestInitialConfigErrorSurvivesGroupsSuccessUntilConfigRecovers(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	app.active = tabGroups

	configLoadID := app.config.loadRequests.Begin(configLoadTarget)
	app.Update(configLoadedMsg{
		RequestID: configLoadID,
		Target:    configLoadTarget,
		Err:       errors.New("config\x1b]0;hidden\x07 unavailable"),
	})
	initialStatus, initialErr := app.statusLine()
	if !initialErr || !strings.Contains(initialStatus, "config unavailable") || strings.Contains(initialStatus, "hidden") {
		t.Fatalf("initial config status = %q, error = %v", initialStatus, initialErr)
	}

	groupsLoadID := app.groups.loadRequests.Begin(groupsLoadTarget)
	app.Update(groupsLoadedMsg{RequestID: groupsLoadID, Target: groupsLoadTarget})
	statusAfterGroups, statusAfterGroupsErr := app.statusLine()
	if !statusAfterGroupsErr || statusAfterGroups != initialStatus {
		t.Fatalf("groups success replaced config failure: status = %q, error = %v", statusAfterGroups, statusAfterGroupsErr)
	}

	configLoadID = app.config.loadRequests.Begin(configLoadTarget)
	app.Update(configLoadedMsg{RequestID: configLoadID, Target: configLoadTarget, Config: &api.Config{Mode: "rule"}})
	statusAfterRecovery, statusAfterRecoveryErr := app.statusLine()
	if statusAfterRecoveryErr || statusAfterRecovery != "config loaded" {
		t.Fatalf("config recovery status = %q, error = %v", statusAfterRecovery, statusAfterRecoveryErr)
	}
}

func TestCtrlCCancelsAppBeforeModalDispatch(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*App)
	}{
		{"profiles import", func(app *App) { app.active, app.profiles.importing = tabProfiles, true }},
		{"groups search", func(app *App) { app.active, app.groups.searching = tabGroups, true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			test.setup(app)

			_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			if cmd == nil {
				t.Fatal("Ctrl-C returned nil command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("Ctrl-C command returned %T, want tea.QuitMsg", cmd())
			}
			select {
			case <-app.ctx.Done():
			case <-time.After(time.Second):
				t.Fatal("Ctrl-C did not cancel the app context")
			}
		})
	}
}

func TestAppCloseCancelsBlockedConfigRequest(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/configs" {
			http.NotFound(w, r)
			return
		}
		close(started)
		<-r.Context().Done()
		close(canceled)
	}))

	result := make(chan tea.Msg, 1)
	go func() { result <- app.config.load()() }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("config request did not reach the handler")
	}
	app.Close()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("app.Close() did not cancel the config request within one second")
	}
	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("config command did not return after cancellation")
	}
}

func TestAsyncOwnerRoutingWhileAnotherTabIsActive(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	app.active = tabTraffic

	groupsLoadID := app.groups.loadRequests.Begin(groupsLoadTarget)
	app.Update(groupsLoadedMsg{
		RequestID: groupsLoadID,
		Target:    groupsLoadTarget,
		Proxies: map[string]api.Proxy{
			"node": {Name: "node", Type: "Shadowsocks"},
		},
		Groups: []api.Proxy{{Name: "Relay", Type: "Relay", Now: "node", All: []string{"node"}}},
	})
	if len(app.groups.rows) != 1 || app.groups.rows[0].name != "Relay" {
		t.Fatalf("inactive groups did not receive /group identity: %+v", app.groups.rows)
	}

	selectionID := app.groups.selectionRequests.Begin("G")
	app.Update(selectedMsg{RequestID: selectionID, Target: "G", Group: "G", Node: "node"})
	if app.status != "selected node" {
		t.Fatalf("selectedMsg status = %q", app.status)
	}

	groupDelayID := app.groups.delayRequests.Begin("G")
	app.Update(groupDelayMsg{RequestID: groupDelayID, Target: "G", Group: "G", Delay: api.GroupDelayResp{"node": 11}})
	if app.groups.delay["node"] != 11 {
		t.Fatalf("groupDelayMsg delay = %d", app.groups.delay["node"])
	}

	groupNodeID := app.groups.delayRequests.Begin(groupNodeDelayTarget("G", "node"))
	app.Update(groupNodeDelayMsg{RequestID: groupNodeID, Group: "G", Node: "node", Delay: 12})
	if app.groups.delay["node"] != 12 {
		t.Fatalf("groupNodeDelayMsg delay = %d", app.groups.delay["node"])
	}

	proxiesLoadID := app.proxies.loadRequests.Begin(proxiesLoadTarget)
	app.Update(proxiesAllMsg{
		RequestID: proxiesLoadID,
		Target:    proxiesLoadTarget,
		Proxies:   map[string]api.Proxy{"P": {Name: "P", Type: "Shadowsocks"}},
		Groups:    []api.Proxy{},
	})
	if len(app.proxies.rows) != 1 {
		t.Fatalf("proxies did not receive proxiesAllMsg: %+v", app.proxies.rows)
	}

	proxyDelayID := app.proxies.delayRequests.Begin("P")
	app.Update(proxyDelayMsg{RequestID: proxyDelayID, Node: "P", Delay: 23})
	if app.proxies.rows[0].delay != 23 {
		t.Fatalf("proxyDelayMsg delay = %d", app.proxies.rows[0].delay)
	}

	profileRequest, ok := app.profiles.beginProfileRequest(profileRequestImportFile, "inactive.yaml")
	if !ok {
		t.Fatal("beginProfileRequest() rejected an idle model")
	}
	app.Update(profileResultMsg{request: profileRequest, entry: profileStore.Entry{Name: "inactive.yaml"}, persisted: true})
	if app.profiles.busy {
		t.Fatal("profiles did not receive profileResultMsg")
	}

	configLoadID := app.config.loadRequests.Begin(configLoadTarget)
	config := &api.Config{Mode: "rule"}
	app.Update(configLoadedMsg{RequestID: configLoadID, Target: configLoadTarget, Config: config})
	if app.config.cfg != config || app.groups.cfg != config {
		t.Fatal("configLoadedMsg did not update config and groups")
	}
	configChangeID := app.config.mutationRequests.Begin("mode")
	app.Update(configChangedMsg{RequestID: configChangeID, Target: "mode", What: "mode=global"})
	if app.status != "mode=global ok" {
		t.Fatalf("configChangedMsg status = %q", app.status)
	}

	app.active = tabGroups
	app.groups.rows = append(app.groups.rows, groupRow{name: "G2", typ: "Selector"})
	app.groups.all["G2"] = api.Proxy{Name: "G2", Type: "Selector"}
	app.proxies.cursor = 0
	app.Update(tea.KeyMsg{Type: tea.KeyDown})
	if app.groups.leftCursor != 1 || app.proxies.cursor != 0 {
		t.Fatalf("key dispatch changed wrong owner: groups=%d proxies=%d", app.groups.leftCursor, app.proxies.cursor)
	}
	app.Update(tea.MouseMsg{})
}

func TestAsyncPlaintextWarningPersistsAfterConfigSuccess(t *testing.T) {
	client, err := api.New("http://controller.example", "secret")
	if err != nil {
		t.Fatal(err)
	}
	store, err := profileStore.Open(t.TempDir(), client)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(context.Background(), client, store)
	t.Cleanup(app.Close)
	app.width, app.height = 100, 30
	configLoadID := app.config.loadRequests.Begin(configLoadTarget)
	app.Update(configLoadedMsg{RequestID: configLoadID, Target: configLoadTarget, Config: &api.Config{Mode: "rule"}})

	if !strings.Contains(app.View(), "plaintext HTTP sends the Mihomo secret") {
		t.Fatalf("persistent client warning missing from view: %q", app.View())
	}
}

func TestMessageBusLatestGenerationAndDrops(t *testing.T) {
	t.Run("latest wins", func(t *testing.T) {
		bus := newMessageBus()
		bus.AdvanceStream(streamConnections, 1)
		bus.SendLatest(streamConnections, 1, "first")
		bus.SendLatest(streamConnections, 1, "last")
		if got := waitBusMsg(t, bus).Inner; got != "last" {
			t.Fatalf("latest message = %#v", got)
		}
	})

	t.Run("stale generation rejected", func(t *testing.T) {
		bus := newMessageBus()
		bus.AdvanceStream(streamConnections, 2)
		bus.SendLatest(streamConnections, 1, "stale")
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()
		if got := bus.Wait(ctx); got != nil {
			t.Fatalf("stale generation produced %#v", got)
		}
	})

	t.Run("advance drains old data without overload", func(t *testing.T) {
		bus := newMessageBus()
		bus.AdvanceStream(streamTraffic, 1)
		bus.SendLatest(streamTraffic, 1, "old")
		bus.AdvanceStream(streamTraffic, 2)
		bus.SendLatest(streamTraffic, 2, "new")
		got := waitBusMsg(t, bus)
		if got.Inner != "new" || got.LogDrops != 0 {
			t.Fatalf("advanced stream message = %#v", got)
		}
	})

	t.Run("current full log increments drop count", func(t *testing.T) {
		bus := newMessageBus()
		bus.AdvanceStream(streamLogs, 1)
		for i := 0; i < 257; i++ {
			bus.SendLog(1, i)
		}
		if got := waitBusMsg(t, bus); got.LogDrops != 1 {
			t.Fatalf("log drops = %d, want 1", got.LogDrops)
		}
	})

	t.Run("stale cannot displace current", func(t *testing.T) {
		bus := newMessageBus()
		bus.AdvanceStream(streamConnections, 1)
		bus.SendLatest(streamConnections, 1, "old")
		bus.AdvanceStream(streamConnections, 2)
		bus.SendLatest(streamConnections, 2, "current")
		bus.SendLatest(streamConnections, 1, "late stale")
		if got := waitBusMsg(t, bus).Inner; got != "current" {
			t.Fatalf("current message displaced by stale message: %#v", got)
		}
	})
}

func TestMessageBusControlCancellation(t *testing.T) {
	t.Run("blocked control resumes", func(t *testing.T) {
		bus := newMessageBus()
		ctx := context.Background()
		for i := 0; i < 32; i++ {
			if !bus.SendControl(ctx, i) {
				t.Fatalf("control send %d failed", i)
			}
		}
		result := make(chan bool, 1)
		go func() { result <- bus.SendControl(ctx, "unblocked") }()
		select {
		case <-result:
			t.Fatal("control send did not block on a full queue")
		case <-time.After(25 * time.Millisecond):
		}
		_ = waitBusMsg(t, bus)
		select {
		case ok := <-result:
			if !ok {
				t.Fatal("unblocked control send returned false")
			}
		case <-time.After(time.Second):
			t.Fatal("blocked control send did not resume")
		}
	})

	t.Run("canceled blocked control returns without leak", func(t *testing.T) {
		bus := newMessageBus()
		for i := 0; i < 32; i++ {
			if !bus.SendControl(context.Background(), i) {
				t.Fatalf("control send %d failed", i)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan bool, 1)
		go func() { result <- bus.SendControl(ctx, errors.New("cancel me")) }()
		cancel()
		select {
		case ok := <-result:
			if ok {
				t.Fatal("canceled blocked control send returned true")
			}
		case <-time.After(time.Second):
			t.Fatal("canceled blocked control send leaked")
		}
	})
}

func newAppForTest(t *testing.T, handler http.Handler) *App {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := api.New(server.URL, "controller-secret")
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	store, err := profileStore.Open(t.TempDir(), client)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	app := NewApp(context.Background(), client, store)
	t.Cleanup(func() {
		app.Close()
		server.Close()
	})
	return app
}

func waitBusMsg(t *testing.T, bus *messageBus) busMsg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg := bus.Wait(ctx)
	got, ok := msg.(busMsg)
	if !ok {
		t.Fatalf("Wait() = %T (%#v), want busMsg", msg, msg)
	}
	return got
}
