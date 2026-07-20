package ui

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"mihomotui/api"
	profileStore "mihomotui/profiles"
)

func TestProfileInputUsesWholePaste(t *testing.T) {
	model, _, _, closeServer := newProfileTestModel(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer closeServer()
	model.importing = true
	cmd, status, isErr := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q16東京"), Paste: true})
	if cmd != nil || status != "" || isErr || model.input != "q16東京" || !model.importing {
		t.Fatalf("pasted input state = input %q, importing %v, cmd %v, status %q, err %v", model.input, model.importing, cmd != nil, status, isErr)
	}
	model.input = "first\nsecond"
	cmd, status, isErr = model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || !isErr || status != "profile source must be a single line" || !model.importing {
		t.Fatalf("invalid submitted input = importing %v, cmd %v, status %q, error %v", model.importing, cmd != nil, status, isErr)
	}
}

func TestProfileStaleResultPreservesRequestStatusAndCursor(t *testing.T) {
	model, store, _, closeServer := newProfileTestModel(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer closeServer()
	first := importProfileFile(t, store, "first")
	model.syncFromStore()
	request, ok := model.beginProfileRequest(profileRequestActivation, first.ID)
	if !ok {
		t.Fatal("beginProfileRequest() rejected an idle model")
	}
	cursor := model.cursor
	second := importProfileFile(t, store, "second")
	stale := profileResultMsg{
		request:   profileRequest{ID: request.ID + 1, Target: second.ID, Kind: profileRequestImportFile},
		entry:     second,
		persisted: true,
	}
	cmd, status, isErr := model.Update(stale)
	if cmd != nil || status != "" || isErr || !model.busy || model.request != request || model.cursor != cursor {
		t.Fatalf("stale result changed state: busy %v, request %+v, cursor %d, status %q, error %v", model.busy, model.request, model.cursor, status, isErr)
	}
	if len(model.entries) != 2 {
		t.Fatalf("persisted stale result did not synchronize entries: %+v", model.entries)
	}
}

func TestProfileLocalImportPersistsWithoutAPIOrActivation(t *testing.T) {
	var requests atomic.Int32
	model, store, root, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer closeServer()
	source := filepath.Join(t.TempDir(), "本地.yaml")
	payload := []byte("mixed-port: 7890\n")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := submitProfileImport(t, model, source)
	status, isErr := executeProfileCommand(t, model, cmd)
	if isErr || !strings.HasPrefix(status, "imported: ") {
		t.Fatalf("import status = %q, error = %v", status, isErr)
	}
	snapshot := store.Snapshot()
	if len(snapshot.Profiles) != 1 || snapshot.Profiles[0].Kind != profileStore.KindFile || snapshot.ActiveID != "" {
		t.Fatalf("Snapshot() = %+v", snapshot)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("local import made %d API requests", got)
	}
	if _, err := os.Stat(filepath.Join(root, "active.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("active.yaml exists after import: %v", err)
	}
	if got, err := store.Read(snapshot.Profiles[0].ID); err != nil || string(got) != string(payload) {
		t.Fatalf("stored payload = %q, %v", got, err)
	}
}

func TestProfileURLImportPersistsWithoutAuthorization(t *testing.T) {
	var authorization string
	var configCalls atomic.Int32
	var profileURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/profile.yaml":
			authorization = r.Header.Get("Authorization")
			_, _ = io.WriteString(w, "proxies: {}\n")
		case "/configs":
			configCalls.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
	model, store, root, closeServer := newProfileTestModel(t, handler)
	defer closeServer()
	profileURL = model.cliURLForTest("/profile.yaml")

	status, isErr := executeProfileCommand(t, model, submitProfileImport(t, model, profileURL))
	if isErr || status != "imported: profile.yaml" {
		t.Fatalf("import status = %q, error = %v", status, isErr)
	}
	snapshot := store.Snapshot()
	if len(snapshot.Profiles) != 1 || snapshot.Profiles[0].Kind != profileStore.KindURL || snapshot.Profiles[0].URL != profileURL || snapshot.ActiveID != "" {
		t.Fatalf("Snapshot() = %+v", snapshot)
	}
	if authorization != "" || configCalls.Load() != 0 {
		t.Fatalf("URL import authorization = %q, config calls = %d", authorization, configCalls.Load())
	}
	if _, err := os.Stat(filepath.Join(root, "active.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("active.yaml exists after URL import: %v", err)
	}
}

func TestProfileActivationCommitsOnlyAfterNoContent(t *testing.T) {
	requestSeen := make(chan string, 1)
	release := make(chan struct{})
	model, store, root, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/configs" || r.Method != http.MethodPut {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Payload string `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requestSeen <- body.Payload
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer closeServer()
	entry := importProfileFile(t, store, "exact payload\n")
	model.syncFromStore()
	model.cursor = profileIndex(model, entry.ID)
	cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("activation command is nil")
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	if got := <-requestSeen; got != "exact payload\n" {
		t.Fatalf("runtime payload = %q", got)
	}
	if got := store.Snapshot().ActiveID; got != "" {
		t.Fatalf("active ID committed before 204: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "active.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("active.yaml committed before 204: %v", err)
	}
	close(release)
	status, isErr := updateProfileResult(t, model, <-result)
	if isErr || status != "activated: source.yaml" {
		t.Fatalf("activation status = %q, error = %v", status, isErr)
	}
	assertActiveProfile(t, store, root, entry.ID, "exact payload\n")
}

func TestProfileViewMarksActiveEntry(t *testing.T) {
	model, store, _, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer closeServer()
	entry := importProfileFile(t, store, "active profile")
	activateStoreProfile(t, store, entry.ID)
	model.syncFromStore()
	model.width, model.height = 80, 20

	view := model.View()
	if !strings.Contains(view, "●") {
		t.Fatalf("active profile view does not contain literal ● marker: %q", view)
	}
	if strings.Contains(view, "*") {
		t.Fatalf("active profile view still uses * marker: %q", view)
	}
}

func TestProfileActivationRejectPreservesPreviousActive(t *testing.T) {
	model, store, root, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "invalid profile")
	}))
	defer closeServer()
	first := importProfileFile(t, store, "first")
	second := importProfileFile(t, store, "second")
	activateStoreProfile(t, store, first.ID)
	model.syncFromStore()
	model.cursor = profileIndex(model, second.ID)

	cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	status, isErr := executeProfileCommand(t, model, cmd)
	if !isErr || !strings.HasPrefix(status, "activation failed: ") {
		t.Fatalf("activation status = %q, error = %v", status, isErr)
	}
	assertActiveProfile(t, store, root, first.ID, "first")
	assertNoProfileStages(t, root)
}

func TestProfileActivationPartialSuccessStatuses(t *testing.T) {
	t.Run("active metadata failure", func(t *testing.T) {
		model, store, root, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer closeServer()
		first := importProfileFile(t, store, "first")
		second := importProfileFile(t, store, "second")
		activateStoreProfile(t, store, first.ID)
		model.syncFromStore()
		model.cursor = profileIndex(model, second.ID)
		if err := os.Remove(filepath.Join(root, "profiles.json")); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(root, "profiles.json"), 0o700); err != nil {
			t.Fatal(err)
		}

		cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
		status, isErr := executeProfileCommand(t, model, cmd)
		if !isErr || !strings.HasPrefix(status, "runtime and boot snapshot updated; active metadata persistence failed: ") {
			t.Fatalf("partial activation status = %q, error = %v", status, isErr)
		}
		if strings.Contains(status, "config applied") {
			t.Fatalf("partial status claims config applied: %q", status)
		}
		if got := store.Snapshot().ActiveID; got != first.ID {
			t.Fatalf("active ID = %q, want %q", got, first.ID)
		}
		data, err := os.ReadFile(filepath.Join(root, "active.yaml"))
		if err != nil || string(data) != "second" {
			t.Fatalf("active.yaml = %q, %v", data, err)
		}
	})

	t.Run("boot snapshot replace failure", func(t *testing.T) {
		var root string
		model, store, openedRoot, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			active := filepath.Join(root, "active.yaml")
			if err := os.Remove(active); err != nil {
				t.Error(err)
			}
			if err := os.Mkdir(active, 0o700); err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		root = openedRoot
		defer closeServer()
		first := importProfileFile(t, store, "first")
		second := importProfileFile(t, store, "second")
		activateStoreProfile(t, store, first.ID)
		model.syncFromStore()
		model.cursor = profileIndex(model, second.ID)

		cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
		status, isErr := executeProfileCommand(t, model, cmd)
		if !isErr || !strings.HasPrefix(status, "runtime activation succeeded; boot snapshot update failed: ") {
			t.Fatalf("partial activation status = %q, error = %v", status, isErr)
		}
		if strings.Contains(status, "config applied") {
			t.Fatalf("partial status claims config applied: %q", status)
		}
		if got := store.Snapshot().ActiveID; got != first.ID {
			t.Fatalf("active ID = %q, want %q", got, first.ID)
		}
		assertNoProfileStages(t, root)
	})
}

func TestProfileRefreshUpdatesAndActivatesWithFailureBoundaries(t *testing.T) {
	var mu sync.Mutex
	downloadBody := "old URL"
	downloadStatus := http.StatusOK
	configStatus := http.StatusNoContent
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		body, getStatus, putStatus := downloadBody, downloadStatus, configStatus
		mu.Unlock()
		switch r.URL.Path {
		case "/subscription.yaml":
			w.WriteHeader(getStatus)
			_, _ = io.WriteString(w, body)
		case "/configs":
			w.WriteHeader(putStatus)
			if putStatus != http.StatusNoContent {
				_, _ = io.WriteString(w, "rejected")
			}
		default:
			http.NotFound(w, r)
		}
	})
	model, store, root, closeServer := newProfileTestModel(t, handler)
	defer closeServer()
	urlEntry, err := store.ImportURL(context.Background(), model.cliURLForTest("/subscription.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	prior := importProfileFile(t, store, "prior active")
	activateStoreProfile(t, store, prior.ID)
	model.syncFromStore()

	mu.Lock()
	downloadBody = "refreshed URL"
	mu.Unlock()
	model.cursor = profileIndex(model, urlEntry.ID)
	cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	status, isErr := executeProfileCommand(t, model, cmd)
	if isErr || status != "refreshed and activated: subscription.yaml" {
		t.Fatalf("refresh status = %q, error = %v", status, isErr)
	}
	assertActiveProfile(t, store, root, urlEntry.ID, "refreshed URL")
	if got := store.Snapshot(); len(got.Profiles) != 2 {
		t.Fatalf("refresh appended an entry: %+v", got)
	}

	beforeFailure := store.Snapshot()
	mu.Lock()
	downloadStatus = http.StatusBadGateway
	downloadBody = "network failure"
	mu.Unlock()
	model.cursor = profileIndex(model, urlEntry.ID)
	cmd, _, _ = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	status, isErr = executeProfileCommand(t, model, cmd)
	if !isErr || !strings.HasPrefix(status, "refresh failed: ") {
		t.Fatalf("download failure status = %q, error = %v", status, isErr)
	}
	if got := store.Snapshot(); !profileSnapshotsEqual(got, beforeFailure) {
		t.Fatalf("download failure changed snapshot: %+v, want %+v", got, beforeFailure)
	}
	if got, err := store.Read(urlEntry.ID); err != nil || string(got) != "refreshed URL" {
		t.Fatalf("download failure changed bytes: %q, %v", got, err)
	}

	activateStoreProfile(t, store, prior.ID)
	model.syncFromStore()
	mu.Lock()
	downloadStatus = http.StatusOK
	downloadBody = "stored despite reject"
	configStatus = http.StatusBadRequest
	mu.Unlock()
	model.cursor = profileIndex(model, urlEntry.ID)
	cmd, _, _ = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	status, isErr = executeProfileCommand(t, model, cmd)
	if !isErr || !strings.HasPrefix(status, "refreshed content stored; activation failed: ") {
		t.Fatalf("activation reject status = %q, error = %v", status, isErr)
	}
	if got, err := store.Read(urlEntry.ID); err != nil || string(got) != "stored despite reject" {
		t.Fatalf("rejected activation did not retain refresh: %q, %v", got, err)
	}
	assertActiveProfile(t, store, root, prior.ID, "prior active")
}

func TestProfileDeleteActiveAndInactive(t *testing.T) {
	model, store, _, closeServer := newProfileTestModel(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer closeServer()
	active := importProfileFile(t, store, "active")
	inactive := importProfileFile(t, store, "inactive")
	activateStoreProfile(t, store, active.ID)
	model.syncFromStore()

	model.cursor = profileIndex(model, active.ID)
	cmd, status, isErr := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil || status == "cannot delete active profile" || isErr {
		t.Fatalf("delete start = cmd %v, status %q, error %v", cmd != nil, status, isErr)
	}
	status, isErr = executeProfileCommand(t, model, cmd)
	if !isErr || status != "cannot delete active profile" {
		t.Fatalf("active delete status = %q, error = %v", status, isErr)
	}

	model.cursor = profileIndex(model, inactive.ID)
	cmd, _, _ = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	status, isErr = executeProfileCommand(t, model, cmd)
	if isErr || status != "deleted: source.yaml" {
		t.Fatalf("inactive delete status = %q, error = %v", status, isErr)
	}
	if got := store.Snapshot(); len(got.Profiles) != 1 || got.Profiles[0].ID != active.ID {
		t.Fatalf("Snapshot() after delete = %+v", got)
	}
}

func TestProfileSingleFlightBlocksThenRetries(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var calls atomic.Int32
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var mu sync.Mutex
	var payloads []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)
		for {
			maximum := maxConcurrent.Load()
			if current <= maximum || maxConcurrent.CompareAndSwap(maximum, current) {
				break
			}
		}
		var body struct {
			Payload string `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		mu.Lock()
		payloads = append(payloads, body.Payload)
		mu.Unlock()
		if calls.Add(1) == 1 {
			close(firstStarted)
			<-releaseFirst
		}
		w.WriteHeader(http.StatusNoContent)
	})
	model, store, root, closeServer := newProfileTestModel(t, handler)
	defer closeServer()
	first := importNamedProfileFile(t, store, "first.yaml", "first payload")
	second := importNamedProfileFile(t, store, "second.yaml", "second payload")
	model.syncFromStore()
	model.width, model.height = 80, 20
	model.cursor = profileIndex(model, first.ID)
	firstCmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	firstResult := make(chan tea.Msg, 1)
	go func() { firstResult <- firstCmd() }()
	<-firstStarted

	model.cursor = profileIndex(model, second.ID)
	secondCmd, status, isErr := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if secondCmd != nil || status != "profile operation already in progress" || !isErr {
		t.Fatalf("blocked second operation = cmd %v, status %q, error %v", secondCmd != nil, status, isErr)
	}
	close(releaseFirst)
	status, isErr = updateProfileResult(t, model, <-firstResult)
	if isErr || status != "activated: first.yaml" {
		t.Fatalf("first activation status = %q, error = %v", status, isErr)
	}

	model.cursor = profileIndex(model, second.ID)
	secondCmd, _, _ = model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	status, isErr = executeProfileCommand(t, model, secondCmd)
	if isErr || status != "activated: second.yaml" {
		t.Fatalf("second activation status = %q, error = %v", status, isErr)
	}
	if got := maxConcurrent.Load(); got > 1 {
		t.Fatalf("maximum concurrent API requests = %d", got)
	}
	mu.Lock()
	gotPayloads := append([]string(nil), payloads...)
	mu.Unlock()
	if len(gotPayloads) != 2 || gotPayloads[1] != "second payload" {
		t.Fatalf("runtime payloads = %q", gotPayloads)
	}
	assertActiveProfile(t, store, root, second.ID, "second payload")
	if model.activeID != second.ID || !strings.Contains(model.View(), "second.yaml") {
		t.Fatalf("model active ID/view = %q / %q", model.activeID, model.View())
	}
}

func newProfileTestModel(t *testing.T, handler http.Handler) (*profilesModel, *profileStore.Store, string, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	client, err := api.New(server.URL, "controller-secret")
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	root := t.TempDir()
	store, err := profileStore.Open(root, client)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	model := newProfilesModel(context.Background(), client, store)
	registerProfileTestURL(model, server.URL)
	return model, store, root, func() {
		profileTestURLs.Delete(model)
		server.Close()
	}
}

var profileTestURLs sync.Map

func registerProfileTestURL(model *profilesModel, base string) {
	profileTestURLs.Store(model, base)
}

func (m *profilesModel) cliURLForTest(path string) string {
	base, _ := profileTestURLs.Load(m)
	return base.(string) + path
}

func submitProfileImport(t *testing.T, model *profilesModel, input string) tea.Cmd {
	t.Helper()
	model.importing = true
	model.input = ""
	if cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(input), Paste: true}); cmd != nil {
		t.Fatal("pasted input returned a command")
	}
	cmd, _, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitted import returned nil command")
	}
	return cmd
}

func executeProfileCommand(t *testing.T, model *profilesModel, cmd tea.Cmd) (string, bool) {
	t.Helper()
	if cmd == nil {
		t.Fatal("profile command is nil")
	}
	return updateProfileResult(t, model, cmd())
}

func updateProfileResult(t *testing.T, model *profilesModel, msg tea.Msg) (string, bool) {
	t.Helper()
	follow, status, isErr := model.Update(msg)
	if follow != nil {
		t.Fatal("profile result unexpectedly returned a follow-up command")
	}
	return status, isErr
}

func importProfileFile(t *testing.T, store *profileStore.Store, data string) profileStore.Entry {
	t.Helper()
	return importNamedProfileFile(t, store, "source.yaml", data)
}

func importNamedProfileFile(t *testing.T, store *profileStore.Store, name, data string) profileStore.Entry {
	t.Helper()
	source := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(source, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportFile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func activateStoreProfile(t *testing.T, store *profileStore.Store, id string) {
	t.Helper()
	txn, err := store.PrepareActivation(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CommitActivation(txn); err != nil {
		t.Fatal(err)
	}
}

func profileIndex(model *profilesModel, id string) int {
	for i := range model.entries {
		if model.entries[i].ID == id {
			return i
		}
	}
	return -1
}

func assertActiveProfile(t *testing.T, store *profileStore.Store, root, id, data string) {
	t.Helper()
	if got := store.Snapshot().ActiveID; got != id {
		t.Fatalf("active ID = %q, want %q", got, id)
	}
	active, err := os.ReadFile(filepath.Join(root, "active.yaml"))
	if err != nil || string(active) != data {
		t.Fatalf("active.yaml = %q, %v; want %q", active, err, data)
	}
}

func assertNoProfileStages(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{root, filepath.Join(root, "profiles")} {
		matches, err := filepath.Glob(filepath.Join(dir, ".mihomotui-*.tmp"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("staged files remain: %v", matches)
		}
	}
}

func profileSnapshotsEqual(left, right profileStore.Snapshot) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}
