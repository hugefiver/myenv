package ui

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
)

func TestStreamRetryRejectsBlockedGeneration(t *testing.T) {
	firstStarted := make(chan struct{})
	firstCanceled := make(chan struct{})
	secondStarted := make(chan struct{})
	var requests atomic.Int32
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" {
			http.NotFound(w, r)
			return
		}
		switch requests.Add(1) {
		case 1:
			close(firstStarted)
			<-r.Context().Done()
			close(firstCanceled)
		case 2:
			close(secondStarted)
			<-r.Context().Done()
		}
	}))

	app.switchTab(tabConns)
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first connections stream did not start")
	}
	first := app.conns.generation
	app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("retry did not start a second connections stream")
	}
	select {
	case <-firstCanceled:
	case <-time.After(time.Second):
		t.Fatal("retry did not cancel the first connections stream")
	}
	second := app.conns.generation
	if first == second {
		t.Fatalf("retry generation = %d, want a new generation after %d", second, first)
	}

	stale := connsSnapMsg{generation: first, snap: api.ConnectionsSnapshot{UploadTotal: 1}}
	app.send(stale)
	current := connsSnapMsg{generation: second, snap: api.ConnectionsSnapshot{UploadTotal: 2}}
	app.send(current)
	msg := waitBusMsg(t, app.bus)
	got, ok := msg.Inner.(connsSnapMsg)
	if !ok || got.snap.UploadTotal != 2 {
		t.Fatalf("queued snapshot = %#v, want only generation %d data", msg.Inner, second)
	}
	app.Update(msg)
	if app.conns.upTotal != 2 {
		t.Fatalf("connections upload total = %d, want current generation data", app.conns.upTotal)
	}

	app.status = "current generation status"
	staleErr := connsSnapMsg{generation: first, err: errors.New("stale\x1b[31m error")}
	app.send(staleErr)
	app.Update(waitBusMsg(t, app.bus))
	staleEnd := connsSnapMsg{generation: first, end: true}
	app.send(staleEnd)
	app.Update(waitBusMsg(t, app.bus))
	if app.status != "current generation status" {
		t.Fatalf("stale stream result overwrote status: %q", app.status)
	}
}

func TestStreamModelsUseLatestBusSnapshot(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))

	app.switchTab(tabConns)
	connectionsGeneration := app.conns.generation
	firstConnection := connsSnapMsg{generation: connectionsGeneration, snap: api.ConnectionsSnapshot{UploadTotal: 1}}
	lastConnection := connsSnapMsg{generation: connectionsGeneration, snap: api.ConnectionsSnapshot{UploadTotal: 2}}
	app.send(firstConnection)
	app.send(lastConnection)
	connections := waitBusMsg(t, app.bus).Inner.(connsSnapMsg)
	if connections.snap.UploadTotal != 2 {
		t.Fatalf("connections pending snapshot = %d, want latest 2", connections.snap.UploadTotal)
	}

	app.switchTab(tabTraffic)
	trafficGeneration := app.traffic.generation
	firstTraffic := trafficMsg{generation: trafficGeneration, t: api.Traffic{Up: 1}}
	lastTraffic := trafficMsg{generation: trafficGeneration, t: api.Traffic{Up: 2}}
	app.send(firstTraffic)
	app.send(lastTraffic)
	traffic := waitBusMsg(t, app.bus).Inner.(trafficMsg)
	if traffic.t.Up != 2 {
		t.Fatalf("traffic pending snapshot = %d, want latest 2", traffic.t.Up)
	}
}

func TestStreamControlsDoNotReplaceLatestSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name    string
		class   streamClass
		data    tea.Msg
		control tea.Msg
		isData  func(tea.Msg) bool
		isCtrl  func(tea.Msg) bool
	}{
		{
			name:    "connections",
			class:   streamConnections,
			data:    connsSnapMsg{generation: 1, snap: api.ConnectionsSnapshot{UploadTotal: 1}},
			control: connsSnapMsg{generation: 1, err: errors.New("connections ended"), end: true},
			isData: func(message tea.Msg) bool {
				value, ok := message.(connsSnapMsg)
				return ok && value.err == nil && !value.end && value.snap.UploadTotal == 1
			},
			isCtrl: func(message tea.Msg) bool {
				value, ok := message.(connsSnapMsg)
				return ok && value.err != nil && value.end
			},
		},
		{
			name:    "traffic",
			class:   streamTraffic,
			data:    trafficMsg{generation: 1, t: api.Traffic{Up: 1}},
			control: trafficMsg{generation: 1, err: errors.New("traffic ended"), end: true},
			isData: func(message tea.Msg) bool {
				value, ok := message.(trafficMsg)
				return ok && value.err == nil && !value.end && value.t.Up == 1
			},
			isCtrl: func(message tea.Msg) bool {
				value, ok := message.(trafficMsg)
				return ok && value.err != nil && value.end
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			app.bus.AdvanceStream(tc.class, 1)
			app.send(tc.data)
			app.send(tc.control)

			first := waitBusMsg(t, app.bus).Inner
			second := waitBusMsg(t, app.bus).Inner
			if !(tc.isData(first) || tc.isData(second)) || !(tc.isCtrl(first) || tc.isCtrl(second)) {
				t.Fatalf("stream queue lost data or control: first=%#v second=%#v", first, second)
			}
		})
	}
}

func TestStreamLogControlSurvivesFullDataQueue(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	app.bus.AdvanceStream(streamLogs, 1)
	for i := 0; i < 257; i++ {
		app.send(logEntryMsg{generation: 1, entry: api.LogEntry{Type: "info", Payload: "data"}})
	}
	app.send(logEntryMsg{generation: 1, err: errors.New("logs ended"), end: true})

	var drops uint64
	messages := make([]tea.Msg, 0, 257)
	for i := 0; i < 257; i++ {
		message := waitBusMsg(t, app.bus)
		drops += message.LogDrops
		messages = append(messages, message.Inner)
	}
	if drops != 1 {
		t.Fatalf("log data overload drops = %d, want 1", drops)
	}
	if !hasLogData(messages...) || !hasLogControl(messages...) {
		t.Fatalf("full log queue lost data or control")
	}
}

func hasLogData(messages ...tea.Msg) bool {
	for _, message := range messages {
		value, ok := message.(logEntryMsg)
		if ok && value.err == nil && !value.end && value.entry.Payload == "data" {
			return true
		}
	}
	return false
}

func hasLogControl(messages ...tea.Msg) bool {
	for _, message := range messages {
		value, ok := message.(logEntryMsg)
		if ok && value.err != nil && value.end {
			return true
		}
	}
	return false
}

func TestStreamLogRetentionLimits(t *testing.T) {
	t.Run("entry count", func(t *testing.T) {
		model := newLogsModel(context.Background(), nil, newMessageBus())
		for i := 0; i < 1001; i++ {
			model.Update(logEntryMsg{entry: api.LogEntry{Type: "info", Payload: "entry"}})
		}
		if len(model.entries) != 1000 || model.dropped != 1 {
			t.Fatalf("retained=%d dropped=%d, want retained=1000 dropped=1", len(model.entries), model.dropped)
		}
	})

	t.Run("aggregate text", func(t *testing.T) {
		model := newLogsModel(context.Background(), nil, newMessageBus())
		entry := api.LogEntry{Type: "info", Payload: strings.Repeat("x", (4<<20)/2+1)}
		model.Update(logEntryMsg{entry: entry})
		model.Update(logEntryMsg{entry: api.LogEntry{Type: "info", Payload: strings.Repeat("y", (4<<20)/2+1)}})
		if len(model.entries) != 1 || !strings.HasPrefix(model.entries[0].Payload, "y") || model.dropped != 1 {
			t.Fatalf("aggregate retention retained=%d first=%q dropped=%d, want latest only and dropped=1", len(model.entries), model.entries[0].Payload, model.dropped)
		}
	})

	t.Run("oversized sanitized entry", func(t *testing.T) {
		model := newLogsModel(context.Background(), nil, newMessageBus())
		model.Update(logEntryMsg{entry: api.LogEntry{Type: "info", Payload: "\x1b[31m" + strings.Repeat("x", 4<<20) + "\x1b[0m"}})
		if len(model.entries) != 0 || model.dropped != 1 {
			t.Fatalf("oversized entry retained=%d dropped=%d, want retained=0 dropped=1", len(model.entries), model.dropped)
		}
	})
}

func TestStreamLogDropsCombineBusAndRetention(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for i := 0; i < 257; i++ {
		app.bus.SendLog(0, logEntryMsg{entry: api.LogEntry{Type: "info", Payload: "bus"}})
	}
	app.Update(waitBusMsg(t, app.bus))
	for i := 0; i < 1000; i++ {
		app.Update(logEntryMsg{entry: api.LogEntry{Type: "info", Payload: "retained"}})
	}
	if app.logs.dropped != 2 {
		t.Fatalf("combined dropped count = %d, want bus overload + retention eviction = 2", app.logs.dropped)
	}
	if !strings.Contains(app.logs.View(), "dropped=2") {
		t.Fatalf("log view does not show combined drop count: %q", app.logs.View())
	}
}

func TestStreamErrorsAreSanitizedAndCancellationIsSilent(t *testing.T) {
	app := newAppForTest(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for _, tc := range []struct {
		name string
		msg  tea.Msg
		want string
	}{
		{"connections 401", connsSnapMsg{err: errors.New("GET /connections: 401 \x1b[31mdenied\x1b[0m")}, "401 denied"},
		{"logs malformed", logEntryMsg{err: errors.New("GET /logs: decode malformed")}, "decode malformed"},
		{"traffic overflow", trafficMsg{err: errors.New("GET /traffic: stream line exceeds 16777216 bytes")}, "exceeds 16777216"},
		{"traffic EOF", trafficMsg{err: errors.New("GET /traffic: EOF")}, "EOF"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app.status = ""
			app.Update(tc.msg)
			if !app.statusErr || !strings.Contains(app.status, tc.want) || strings.Contains(app.status, "\x1b") {
				t.Fatalf("stream status = %q, error=%v, want sanitized %q", app.status, app.statusErr, tc.want)
			}
		})
	}
	app.status = ""
	app.statusErr = false
	app.Update(connsSnapMsg{err: context.Canceled})
	if app.status != "" || app.statusErr {
		t.Fatalf("cancellation surfaced as stream error: %q, error=%v", app.status, app.statusErr)
	}
}

func TestStreamTabsAdvertiseAndHandleRetry(t *testing.T) {
	for _, tc := range []struct {
		name string
		tab  tab
	}{
		{"connections", tabConns},
		{"logs", tabLogs},
		{"traffic", tabTraffic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(helpFor(tc.tab, ""), "r retry") {
				t.Fatalf("%s help lacks retry action: %q", tc.name, helpFor(tc.tab, ""))
			}
		})
	}
}

func TestStreamRetryKeyStartsNewGenerationForAllTabs(t *testing.T) {
	for _, tc := range []struct {
		name string
		tab  tab
		path string
	}{
		{"connections", tabConns, "/connections"},
		{"logs", tabLogs, "/logs"},
		{"traffic", tabTraffic, "/traffic"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			firstStarted := make(chan struct{})
			secondStarted := make(chan struct{})
			var requests atomic.Int32
			app := newAppForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					http.NotFound(w, r)
					return
				}
				switch requests.Add(1) {
				case 1:
					close(firstStarted)
				case 2:
					close(secondStarted)
				}
				<-r.Context().Done()
			}))

			app.switchTab(tc.tab)
			select {
			case <-firstStarted:
			case <-time.After(time.Second):
				t.Fatal("first stream did not start")
			}
			first := streamGenerationForTab(app, tc.tab)
			app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
			select {
			case <-secondStarted:
			case <-time.After(time.Second):
				t.Fatal("retry key did not start a replacement stream")
			}
			if current := streamGenerationForTab(app, tc.tab); current == first {
				t.Fatalf("retry generation = %d, want a new generation after %d", current, first)
			}
		})
	}
}

func streamGenerationForTab(app *App, active tab) requestID {
	switch active {
	case tabConns:
		return app.conns.generation
	case tabLogs:
		return app.logs.generation
	case tabTraffic:
		return app.traffic.generation
	}
	return 0
}
