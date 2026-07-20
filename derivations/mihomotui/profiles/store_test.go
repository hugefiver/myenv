package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type testDownloader struct {
	mu    sync.Mutex
	calls int
	data  []byte
	final string
	err   error
}

type downloadFunc func(context.Context, string) ([]byte, string, error)

func (fn downloadFunc) DownloadProfile(ctx context.Context, rawURL string) ([]byte, string, error) {
	return fn(ctx, rawURL)
}

func (d *testDownloader) DownloadProfile(_ context.Context, _ string) ([]byte, string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	if d.err != nil {
		return nil, "", d.err
	}
	return append([]byte(nil), d.data...), d.final, nil
}

func (d *testDownloader) callCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

func TestResolveDataDir(t *testing.T) {
	homeErr := errors.New("no home")
	cases := []struct {
		name    string
		goos    string
		env     map[string]string
		home    string
		homeErr error
		want    string
		wantErr bool
	}{
		{name: "override wins", goos: "linux", env: map[string]string{"MIHOMOTUI_DATA_DIR": "D:/profiles", "XDG_DATA_HOME": "ignored"}, want: "D:/profiles"},
		{name: "linux xdg", goos: "linux", env: map[string]string{"XDG_DATA_HOME": "data"}, want: filepath.Join("data", "mihomotui")},
		{name: "linux home", goos: "linux", home: "home", want: filepath.Join("home", ".local", "share", "mihomotui")},
		{name: "linux home error", goos: "linux", homeErr: homeErr, wantErr: true},
		{name: "windows", goos: "windows", env: map[string]string{"LOCALAPPDATA": "AppData"}, want: filepath.Join("AppData", "mihomotui")},
		{name: "windows missing app data", goos: "windows", wantErr: true},
		{name: "darwin", goos: "darwin", home: "home", want: filepath.Join("home", "Library", "Application Support", "mihomotui")},
		{name: "unsupported", goos: "plan9", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveDataDir(tc.goos, func(key string) string { return tc.env[key] }, func() (string, error) {
				if tc.homeErr != nil {
					return "", tc.homeErr
				}
				return tc.home, nil
			})
			if tc.wantErr {
				if err == nil {
					t.Fatal("resolveDataDir() succeeded, want error")
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("resolveDataDir() = %q, %v; want %q, nil", got, err, tc.want)
			}
		})
	}
}

func TestImportFilePersistsOwnedCopy(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "任意-プロフィール")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "配置-文件.yaml")
	want := []byte("mixed-port: 7890\n# 保留\n")
	if err := os.WriteFile(source, want, 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportFile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Name != filepath.Base(source) || entry.Kind != KindFile || entry.URL != "" || entry.File != profileFileName(entry.ID) {
		t.Fatalf("ImportFile() entry = %+v", entry)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Read(entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("Read() = %q, want %q", got, want)
	}
	metadata, err := os.ReadFile(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(metadata), source) {
		t.Fatalf("metadata persisted source path %q", source)
	}
}

func TestImportURLStoresSubmittedURLAndDoesNotRedownload(t *testing.T) {
	root := t.TempDir()
	original := "https://origin.example/subscription"
	download := &testDownloader{data: []byte("proxies: {}\n"), final: "https://cdn.example/files/final.yaml?token=1"}
	store, err := Open(root, download)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportURL(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Kind != KindURL || entry.URL != original || entry.Name != "final.yaml" || download.callCount() != 1 {
		t.Fatalf("ImportURL() entry = %+v, calls = %d", entry, download.callCount())
	}

	noRedownload := &testDownloader{err: errors.New("must not download while reopening")}
	reopened, err := Open(root, noRedownload)
	if err != nil {
		t.Fatal(err)
	}
	if noRedownload.callCount() != 0 {
		t.Fatal("Open() redownloaded a persisted profile")
	}
	if got := reopened.Snapshot().Profiles[0]; got.URL != original || got.Name != "final.yaml" {
		t.Fatalf("reopened metadata = %+v", got)
	}
}

func TestImportURLCancellationAfterDownloadLeavesNoArtifacts(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	downloader := downloadFunc(func(context.Context, string) ([]byte, string, error) {
		close(started)
		<-release
		return []byte("proxies: {}\n"), "https://cdn.example/final.yaml", nil
	})
	root := t.TempDir()
	store, err := Open(root, downloader)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errs := make(chan error, 1)
	go func() {
		_, err := store.ImportURL(ctx, "https://origin.example/subscription")
		errs <- err
	}()
	<-started
	cancel()
	close(release)
	if err := <-errs; !errors.Is(err, context.Canceled) {
		t.Fatalf("ImportURL() cancellation error = %v", err)
	}
	if got := store.Snapshot(); len(got.Profiles) != 0 {
		t.Fatalf("state published after cancellation: %+v", got)
	}
	entries, err := os.ReadDir(filepath.Join(root, "profiles"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("owned profile remains after cancellation: %v", entries)
	}
	assertNoOwnedTemps(t, root)
}

func TestImportFileRejectsInvalidSourcesAndOversize(t *testing.T) {
	store, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportFile(context.Background(), filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("ImportFile() missing source succeeded")
	}
	if _, err := store.ImportFile(context.Background(), t.TempDir()); err == nil {
		t.Fatal("ImportFile() directory source succeeded")
	}
	if _, err := store.ImportFile(context.Background(), "bad\x00path"); err == nil {
		t.Fatal("ImportFile() invalid open path succeeded")
	}

	over := filepath.Join(t.TempDir(), "large.yaml")
	file, err := os.Create(over)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxProfileBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportFile(context.Background(), over); err == nil || !strings.Contains(err.Error(), "exceeds 33554432 bytes") {
		t.Fatalf("ImportFile() oversized error = %v", err)
	}

	tooLarge := &testDownloader{data: make([]byte, MaxProfileBytes+1), final: "https://cdn.example/large.yaml"}
	store, err = Open(t.TempDir(), tooLarge)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportURL(context.Background(), "https://origin.example/large"); err == nil || !strings.Contains(err.Error(), "exceeds 33554432 bytes") {
		t.Fatalf("ImportURL() oversized error = %v", err)
	}
}

func TestImportFailureLeavesNoOwnedArtifacts(t *testing.T) {
	t.Run("metadata failure", func(t *testing.T) {
		root := t.TempDir()
		store, err := Open(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		store.writeIndexFn = func(Snapshot) error { return errors.New("metadata disk full") }
		source := filepath.Join(t.TempDir(), "source.yaml")
		if err := os.WriteFile(source, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ImportFile(context.Background(), source); err == nil {
			t.Fatal("ImportFile() succeeded when metadata write failed")
		}
		if got := store.Snapshot(); len(got.Profiles) != 0 {
			t.Fatalf("state published after metadata failure: %+v", got)
		}
		assertNoOwnedTemps(t, root)
		entries, err := os.ReadDir(filepath.Join(root, "profiles"))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("unindexed profile remains: %v", entries)
		}
	})

	t.Run("download failure and cancellation", func(t *testing.T) {
		root := t.TempDir()
		failed, err := Open(root, &testDownloader{err: errors.New("network down")})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := failed.ImportURL(context.Background(), "https://origin.example/failure"); err == nil {
			t.Fatal("ImportURL() download failure succeeded")
		}
		cancelled, err := Open(t.TempDir(), &testDownloader{data: []byte("x"), final: "https://cdn.example/x.yaml"})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := cancelled.ImportURL(ctx, "https://origin.example/cancel"); !errors.Is(err, context.Canceled) {
			t.Fatalf("ImportURL() cancellation error = %v", err)
		}
		assertNoOwnedTemps(t, root)
		assertNoOwnedTemps(t, cancelled.root)
	})
}

func TestOpenUsesSchemaAndPrivateModes(t *testing.T) {
	root := t.TempDir()
	store, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC) }
	source := filepath.Join(t.TempDir(), "source.yaml")
	if err := os.WriteFile(source, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportFile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(entry.ID) {
		t.Fatalf("profile ID %q is not lowercase hexadecimal", entry.ID)
	}

	data, err := os.ReadFile(filepath.Join(root, "profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Version  int `json:"version"`
		Profiles []struct {
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Version != 1 || len(raw.Profiles) != 1 {
		t.Fatalf("metadata schema = %+v", raw)
	}
	for _, timestamp := range []string{raw.Profiles[0].CreatedAt, raw.Profiles[0].UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
			t.Fatalf("timestamp %q is not RFC3339: %v", timestamp, err)
		}
	}
	if !strings.HasSuffix(string(data), "\n") || !strings.Contains(string(data), "\n  \"profiles\"") {
		t.Fatalf("metadata is not indented JSON with a trailing newline: %q", data)
	}

	if runtime.GOOS != "windows" {
		for _, name := range []string{root, filepath.Join(root, "profiles")} {
			info, err := os.Stat(name)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0o700 {
				t.Fatalf("directory %q mode = %#o, want 0700", name, got)
			}
		}
		for _, name := range []string{filepath.Join(root, "profiles.json"), store.profilePath(entry.ID)} {
			info, err := os.Stat(name)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("file %q mode = %#o, want 0600", name, got)
			}
		}
	}
}

func TestOpenReconcilesOnlyOwnedArtifacts(t *testing.T) {
	t.Run("restores referenced delete temp", func(t *testing.T) {
		root := t.TempDir()
		id := testID(1)
		entry := validEntry(id, KindFile)
		writeMetadata(t, root, Snapshot{Version: 1, Profiles: []Entry{entry}})
		deleteTemp := filepath.Join(root, "profiles", ".mihomotui-delete-"+id+".tmp")
		if err := os.WriteFile(deleteTemp, []byte("recovered"), 0o600); err != nil {
			t.Fatal(err)
		}
		store, err := Open(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := store.Read(id)
		if err != nil || string(got) != "recovered" {
			t.Fatalf("recovered profile = %q, %v", got, err)
		}
		assertNotExists(t, deleteTemp)
	})

	t.Run("removes stale owned files but preserves unrelated files", func(t *testing.T) {
		root := t.TempDir()
		writeMetadata(t, root, emptySnapshot())
		profiles := filepath.Join(root, "profiles")
		owned := filepath.Join(profiles, testID(2)+".yaml")
		unrelatedYAML := filepath.Join(profiles, "unrelated.yaml")
		notes := filepath.Join(profiles, "notes.tmp")
		staleDelete := filepath.Join(profiles, ".mihomotui-delete-"+testID(3)+".tmp")
		for name, data := range map[string]string{
			owned:         "owned",
			unrelatedYAML: "unrelated",
			notes:         "notes",
			staleDelete:   "delete",
			filepath.Join(profiles, ".mihomotui-stage.tmp"): "stage",
			filepath.Join(root, ".mihomotui-index.tmp"):     "index",
		} {
			if err := os.WriteFile(name, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Open(root, nil); err != nil {
			t.Fatal(err)
		}
		assertNotExists(t, owned)
		assertNotExists(t, staleDelete)
		assertNotExists(t, filepath.Join(profiles, ".mihomotui-stage.tmp"))
		assertNotExists(t, filepath.Join(root, ".mihomotui-index.tmp"))
		assertExists(t, unrelatedYAML)
		assertExists(t, notes)
	})
}

func TestOpenRejectsInvalidMetadata(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{name: "unsupported version", mutate: func(s *Snapshot) { s.Version = 2 }},
		{name: "invalid ID", mutate: func(s *Snapshot) { s.Profiles[0].ID = "bad" }},
		{name: "duplicate ID", mutate: func(s *Snapshot) { s.Profiles = append(s.Profiles, s.Profiles[0]) }},
		{name: "empty name", mutate: func(s *Snapshot) { s.Profiles[0].Name = "" }},
		{name: "invalid kind", mutate: func(s *Snapshot) { s.Profiles[0].Kind = "other" }},
		{name: "zero created at", mutate: func(s *Snapshot) { s.Profiles[0].CreatedAt = time.Time{} }},
		{name: "updated before created", mutate: func(s *Snapshot) { s.Profiles[0].UpdatedAt = s.Profiles[0].CreatedAt.Add(-time.Second) }},
		{name: "traversal file", mutate: func(s *Snapshot) { s.Profiles[0].File = "profiles/../other.yaml" }},
		{name: "absolute file", mutate: func(s *Snapshot) { s.Profiles[0].File = "/profiles/other.yaml" }},
		{name: "mismatched file", mutate: func(s *Snapshot) { s.Profiles[0].File = "profiles/" + testID(99) + ".yaml" }},
		{name: "file source URL", mutate: func(s *Snapshot) { s.Profiles[0].URL = "https://origin.example/x" }},
		{name: "invalid URL profile", mutate: func(s *Snapshot) { s.Profiles[0].Kind, s.Profiles[0].URL = KindURL, "ftp://origin.example/x" }},
		{name: "relative URL profile", mutate: func(s *Snapshot) { s.Profiles[0].Kind, s.Profiles[0].URL = KindURL, "/x" }},
		{name: "userinfo URL profile", mutate: func(s *Snapshot) { s.Profiles[0].Kind, s.Profiles[0].URL = KindURL, "https://user@origin.example/x" }},
		{name: "unknown active ID", mutate: func(s *Snapshot) { s.ActiveID = testID(100) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			entry := validEntry(testID(10), KindFile)
			writeOwnedProfile(t, root, entry.ID, "valid")
			state := Snapshot{Version: 1, Profiles: []Entry{entry}}
			tc.mutate(&state)
			writeMetadata(t, root, state)
			if _, err := Open(root, nil); err == nil {
				t.Fatal("Open() accepted invalid metadata")
			}
		})
	}

	t.Run("trailing JSON", func(t *testing.T) {
		root := t.TempDir()
		entry := validEntry(testID(11), KindFile)
		writeOwnedProfile(t, root, entry.ID, "valid")
		state, err := json.Marshal(Snapshot{Version: 1, Profiles: []Entry{entry}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "profiles.json"), append(state, []byte("\n{}\n")...), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(root, nil); err == nil {
			t.Fatal("Open() accepted trailing JSON")
		}
	})

	t.Run("missing and nonregular referenced profile", func(t *testing.T) {
		for _, directory := range []bool{false, true} {
			root := t.TempDir()
			entry := validEntry(testID(12+boolToInt(directory)), KindFile)
			writeMetadata(t, root, Snapshot{Version: 1, Profiles: []Entry{entry}})
			if directory {
				if err := os.Mkdir(filepath.Join(root, "profiles", entry.ID+".yaml"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Open(root, nil); err == nil {
				t.Fatal("Open() accepted a missing or nonregular referenced profile")
			}
		}
	})

	t.Run("invalid metadata cannot trigger cleanup", func(t *testing.T) {
		root := t.TempDir()
		stale := filepath.Join(root, "profiles", testID(13)+".yaml")
		writeOwnedProfile(t, root, testID(13), "must survive")
		writeMetadata(t, root, Snapshot{Version: 2, Profiles: []Entry{}})
		if _, err := Open(root, nil); err == nil {
			t.Fatal("Open() accepted unsupported metadata")
		}
		assertExists(t, stale)
	})
}

func TestDeleteTransactionsAndRecovery(t *testing.T) {
	t.Run("active nonactive and unknown", func(t *testing.T) {
		store := mustStore(t, t.TempDir())
		first := mustImport(t, store, "first")
		second := mustImport(t, store, "second")
		setActiveID(store, first.ID)
		if err := store.Delete(context.Background(), first.ID); !errors.Is(err, ErrActiveProfile) {
			t.Fatalf("Delete(active) error = %v", err)
		}
		setActiveID(store, "")
		if err := store.Delete(context.Background(), testID(404)); !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("Delete(unknown) error = %v", err)
		}
		if err := store.Delete(context.Background(), second.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Read(second.ID); !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("Read(deleted) error = %v", err)
		}
		assertNotExists(t, store.profilePath(second.ID))
	})

	t.Run("metadata failure rolls back profile rename", func(t *testing.T) {
		store := mustStore(t, t.TempDir())
		entry := mustImport(t, store, "rollback")
		store.writeIndexFn = func(Snapshot) error { return errors.New("write failure") }
		if err := store.Delete(context.Background(), entry.ID); err == nil {
			t.Fatal("Delete() succeeded when metadata write failed")
		}
		if _, err := store.Read(entry.ID); err != nil {
			t.Fatalf("profile was not rolled back: %v", err)
		}
		assertNotExists(t, store.deleteTempPath(entry.ID))
	})

	t.Run("open restores crash before index", func(t *testing.T) {
		root := t.TempDir()
		store := mustStore(t, root)
		entry := mustImport(t, store, "before")
		if err := os.Rename(store.profilePath(entry.ID), store.deleteTempPath(entry.ID)); err != nil {
			t.Fatal(err)
		}
		reopened, err := Open(root, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reopened.Read(entry.ID); err != nil {
			t.Fatalf("Open() did not restore profile: %v", err)
		}
	})

	t.Run("open finishes crash after index", func(t *testing.T) {
		root := t.TempDir()
		store := mustStore(t, root)
		entry := mustImport(t, store, "after")
		if err := store.writeIndex(emptySnapshot()); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(store.profilePath(entry.ID), store.deleteTempPath(entry.ID)); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(root, nil); err != nil {
			t.Fatal(err)
		}
		assertNotExists(t, store.deleteTempPath(entry.ID))
	})

	t.Run("final cleanup is deferred then recovered", func(t *testing.T) {
		root := t.TempDir()
		store := mustStore(t, root)
		entry := mustImport(t, store, "deferred")
		deleteTemp := store.deleteTempPath(entry.ID)
		store.writeIndexFn = func(next Snapshot) error {
			if err := store.writeIndex(next); err != nil {
				return err
			}
			if err := os.Remove(deleteTemp); err != nil {
				return err
			}
			if err := os.Mkdir(deleteTemp, 0o700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(deleteTemp, "blocker"), []byte("x"), 0o600)
		}
		err := store.Delete(context.Background(), entry.ID)
		if err == nil || !strings.Contains(err.Error(), "profile metadata deleted; owned cleanup deferred:") {
			t.Fatalf("Delete() cleanup error = %v", err)
		}
		if _, err := Open(root, nil); err != nil {
			t.Fatalf("Open() did not complete deferred cleanup: %v", err)
		}
		assertNotExists(t, deleteTemp)
	})
}

func TestDeleteCancellationBeforeMetadataRollsBack(t *testing.T) {
	store := mustStore(t, t.TempDir())
	entry := mustImport(t, store, "rollback on cancellation")
	before := store.Snapshot()
	beforeIndex, err := os.ReadFile(store.indexPath())
	if err != nil {
		t.Fatal(err)
	}
	staged := make(chan struct{})
	release := make(chan struct{})
	originalReplace := store.replaceFn
	store.replaceFn = func(source, destination string) error {
		err := originalReplace(source, destination)
		if err == nil && destination == store.deleteTempPath(entry.ID) {
			close(staged)
			<-release
		}
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deleter := contextDeleterForTest(t, store)
	done := make(chan error, 1)
	go func() { done <- deleter.Delete(ctx, entry.ID) }()
	<-staged
	cancel()
	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() cancellation error = %v", err)
	}
	assertProfileUnchanged(t, store, entry.ID, "rollback on cancellation", before)
	afterIndex, err := os.ReadFile(store.indexPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(afterIndex) != string(beforeIndex) {
		t.Fatalf("profiles index changed after cancellation: got %q, want %q", afterIndex, beforeIndex)
	}
	assertNoOwnedTemps(t, store.root)
}

func TestDeleteCancellationAlreadyCanceledLeavesProfileUnchanged(t *testing.T) {
	store := mustStore(t, t.TempDir())
	entry := mustImport(t, store, "already cancelled")
	before := store.Snapshot()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := contextDeleterForTest(t, store).Delete(ctx, entry.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() cancellation error = %v", err)
	}
	assertProfileUnchanged(t, store, entry.ID, "already cancelled", before)
	assertNoOwnedTemps(t, store.root)
}

func TestConcurrentImportsPreserveAllEntries(t *testing.T) {
	store := mustStore(t, t.TempDir())
	const count = 12
	sources := make([]string, count)
	contents := make([]string, count)
	for i := range sources {
		contents[i] = fmt.Sprintf("profile-%d", i)
		sources[i] = filepath.Join(t.TempDir(), fmt.Sprintf("source-%d.yaml", i))
		if err := os.WriteFile(sources[i], []byte(contents[i]), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	start := make(chan struct{})
	var ready, done sync.WaitGroup
	ready.Add(count)
	done.Add(count)
	entries := make(chan Entry, count)
	errs := make(chan error, count)
	for _, source := range sources {
		source := source
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			entry, err := store.ImportFile(context.Background(), source)
			if err != nil {
				errs <- err
				return
			}
			entries <- entry
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(entries)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent ImportFile() error: %v", err)
	}
	if t.Failed() {
		return
	}
	if got := store.Snapshot(); len(got.Profiles) != count {
		t.Fatalf("Snapshot() retained %d profiles, want %d", len(got.Profiles), count)
	}
	seen := make(map[string]bool, count)
	for entry := range entries {
		data, err := store.Read(entry.ID)
		if err != nil {
			t.Fatal(err)
		}
		seen[string(data)] = true
	}
	for _, content := range contents {
		if !seen[content] {
			t.Errorf("lost concurrent profile %q", content)
		}
	}
}

func TestActivationStageCommit(t *testing.T) {
	root := t.TempDir()
	store := mustStore(t, root)
	entry := mustImport(t, store, "mixed-port: 7890\n")

	txn, err := store.PrepareActivation(entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if txn.ID != entry.ID || string(txn.Payload) != "mixed-port: 7890\n" {
		t.Fatalf("PrepareActivation() = %+v", txn)
	}
	if _, err := os.Stat(txn.staged); err != nil {
		t.Fatalf("activation stage does not exist: %v", err)
	}
	if err := store.CommitActivation(txn); err != nil {
		t.Fatal(err)
	}

	active, err := os.ReadFile(filepath.Join(root, "active.yaml"))
	if err != nil || string(active) != "mixed-port: 7890\n" {
		t.Fatalf("active.yaml = %q, %v", active, err)
	}
	if got := store.Snapshot().ActiveID; got != entry.ID {
		t.Fatalf("active ID = %q, want %q", got, entry.ID)
	}
	reopened, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Snapshot().ActiveID; got != entry.ID {
		t.Fatalf("persisted active ID = %q, want %q", got, entry.ID)
	}
	assertNotExists(t, txn.staged)
}

func TestAbortActivationPreservesOldSnapshot(t *testing.T) {
	root := t.TempDir()
	store := mustStore(t, root)
	first := mustImport(t, store, "first")
	second := mustImport(t, store, "second")
	activateForTest(t, store, first.ID)

	txn, err := store.PrepareActivation(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AbortActivation(txn); err != nil {
		t.Fatal(err)
	}
	active, err := os.ReadFile(filepath.Join(root, "active.yaml"))
	if err != nil || string(active) != "first" {
		t.Fatalf("active.yaml after abort = %q, %v", active, err)
	}
	if got := store.Snapshot().ActiveID; got != first.ID {
		t.Fatalf("active ID after abort = %q, want %q", got, first.ID)
	}
	assertNotExists(t, txn.staged)
}

func TestCommitActivationReplaceFailurePreservesMetadata(t *testing.T) {
	root := t.TempDir()
	store := mustStore(t, root)
	first := mustImport(t, store, "current")
	second := mustImport(t, store, "candidate")
	activateForTest(t, store, first.ID)
	txn, err := store.PrepareActivation(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	store.replaceFn = func(source, destination string) error {
		if destination == filepath.Join(root, "active.yaml") {
			return errors.New("active disk full")
		}
		return replaceFile(source, destination)
	}

	if err := store.CommitActivation(txn); err == nil || !strings.Contains(err.Error(), "replace active profile") {
		t.Fatalf("CommitActivation() error = %v", err)
	}
	if got := store.Snapshot().ActiveID; got != first.ID {
		t.Fatalf("active ID changed to %q, want %q", got, first.ID)
	}
	active, readErr := os.ReadFile(filepath.Join(root, "active.yaml"))
	if readErr != nil || string(active) != "current" {
		t.Fatalf("active.yaml after replace failure = %q, %v", active, readErr)
	}
	assertNotExists(t, txn.staged)
	if err := store.AbortActivation(txn); err == nil {
		t.Fatal("AbortActivation() accepted a completed transaction")
	}
}

func TestCommitActivationMetadataFailureLeavesRuntimeSnapshotOnly(t *testing.T) {
	root := t.TempDir()
	store := mustStore(t, root)
	first := mustImport(t, store, "first")
	second := mustImport(t, store, "second")
	activateForTest(t, store, first.ID)
	txn, err := store.PrepareActivation(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	store.writeIndexFn = func(Snapshot) error { return errors.New("metadata disk full") }

	err = store.CommitActivation(txn)
	if !errors.Is(err, ErrActiveIDPersistence) {
		t.Fatalf("CommitActivation() error = %v, want ErrActiveIDPersistence", err)
	}
	active, readErr := os.ReadFile(filepath.Join(root, "active.yaml"))
	if readErr != nil || string(active) != "second" {
		t.Fatalf("active.yaml = %q, %v", active, readErr)
	}
	if got := store.Snapshot().ActiveID; got != first.ID {
		t.Fatalf("in-memory active ID = %q, want %q", got, first.ID)
	}
	reopened, openErr := Open(root, nil)
	if openErr != nil {
		t.Fatal(openErr)
	}
	if got := reopened.Snapshot().ActiveID; got != first.ID {
		t.Fatalf("persisted active ID = %q, want %q", got, first.ID)
	}
	assertNotExists(t, txn.staged)
}

func TestActivationRejectsInvalidForeignAndReusedTransactions(t *testing.T) {
	firstStore := mustStore(t, t.TempDir())
	secondStore := mustStore(t, t.TempDir())
	entry := mustImport(t, firstStore, "candidate")
	if _, err := firstStore.PrepareActivation(testID(404)); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("PrepareActivation(unknown) error = %v", err)
	}
	if err := firstStore.CommitActivation(nil); err == nil {
		t.Fatal("CommitActivation(nil) succeeded")
	}
	txn, err := firstStore.PrepareActivation(entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := secondStore.CommitActivation(txn); err == nil {
		t.Fatal("foreign CommitActivation() succeeded")
	}
	if err := firstStore.AbortActivation(txn); err != nil {
		t.Fatal(err)
	}
	if err := firstStore.AbortActivation(txn); err == nil {
		t.Fatal("reused AbortActivation() succeeded")
	}
	if err := firstStore.CommitActivation(txn); err == nil {
		t.Fatal("reused CommitActivation() succeeded")
	}
}

func TestRefreshURLReplacesSameProfile(t *testing.T) {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	downloader := &testDownloader{data: []byte("old"), final: "https://cdn.example/old.yaml"}
	store, err := Open(t.TempDir(), downloader)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now }
	entry, err := store.ImportURL(context.Background(), "https://origin.example/profile")
	if err != nil {
		t.Fatal(err)
	}
	downloader.data = []byte("new")
	downloader.final = "https://cdn.example/new.yaml"
	now = now.Add(time.Hour)

	refreshed, err := store.RefreshURL(context.Background(), entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.ID != entry.ID || refreshed.CreatedAt != entry.CreatedAt || !refreshed.UpdatedAt.Equal(now) {
		t.Fatalf("RefreshURL() = %+v, original = %+v", refreshed, entry)
	}
	if got := store.Snapshot(); len(got.Profiles) != 1 || got.Profiles[0].ID != entry.ID {
		t.Fatalf("Snapshot() = %+v", got)
	}
	data, err := store.Read(entry.ID)
	if err != nil || string(data) != "new" {
		t.Fatalf("refreshed bytes = %q, %v", data, err)
	}
	fileEntry := mustImport(t, store, "local")
	if _, err := store.RefreshURL(context.Background(), fileEntry.ID); err == nil {
		t.Fatal("RefreshURL(FILE) succeeded")
	}
}

func TestRefreshDownloadFailurePreservesProfile(t *testing.T) {
	downloader := &testDownloader{data: []byte("old"), final: "https://cdn.example/old.yaml"}
	store, err := Open(t.TempDir(), downloader)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportURL(context.Background(), "https://origin.example/profile")
	if err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	downloader.err = errors.New("network down")
	if _, err := store.RefreshURL(context.Background(), entry.ID); err == nil {
		t.Fatal("RefreshURL() succeeded after download failure")
	}
	assertProfileUnchanged(t, store, entry.ID, "old", before)
}

func TestRefreshMetadataFailureRollsBackBytesAndTimestamp(t *testing.T) {
	downloader := &testDownloader{data: []byte("old"), final: "https://cdn.example/old.yaml"}
	store, err := Open(t.TempDir(), downloader)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportURL(context.Background(), "https://origin.example/profile")
	if err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	downloader.data = []byte("new")
	store.writeIndexFn = func(Snapshot) error { return errors.New("metadata disk full") }
	if _, err := store.RefreshURL(context.Background(), entry.ID); err == nil {
		t.Fatal("RefreshURL() succeeded after metadata failure")
	}
	assertProfileUnchanged(t, store, entry.ID, "old", before)
	assertNoOwnedTemps(t, store.root)
}

func TestRefreshRejectsOversizeAndCancellation(t *testing.T) {
	var response []byte
	downloader := downloadFunc(func(ctx context.Context, _ string) ([]byte, string, error) {
		return append([]byte(nil), response...), "https://cdn.example/profile.yaml", nil
	})
	store, err := Open(t.TempDir(), downloader)
	if err != nil {
		t.Fatal(err)
	}
	response = []byte("old")
	entry, err := store.ImportURL(context.Background(), "https://origin.example/profile")
	if err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	response = make([]byte, MaxProfileBytes+1)
	if _, err := store.RefreshURL(context.Background(), entry.ID); err == nil || !strings.Contains(err.Error(), "exceeds 33554432 bytes") {
		t.Fatalf("RefreshURL() oversized error = %v", err)
	}
	assertProfileUnchanged(t, store, entry.ID, "old", before)

	ctx, cancel := context.WithCancel(context.Background())
	originalReplace := store.replaceFn
	replaced := false
	store.replaceFn = func(source, destination string) error {
		err := originalReplace(source, destination)
		if err == nil && destination == store.profilePath(entry.ID) && !replaced {
			replaced = true
			cancel()
		}
		return err
	}
	response = []byte("cancelled replacement")
	if _, err := store.RefreshURL(ctx, entry.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("RefreshURL() cancellation error = %v", err)
	}
	if !replaced {
		t.Fatal("RefreshURL() did not reach replacement cancellation boundary")
	}
	assertProfileUnchanged(t, store, entry.ID, "old", before)
}

func TestActivationSerializesRefreshAndImport(t *testing.T) {
	downloadStarted := make(chan struct{}, 1)
	downloader := downloadFunc(func(context.Context, string) ([]byte, string, error) {
		downloadStarted <- struct{}{}
		return []byte("refreshed"), "https://cdn.example/profile.yaml", nil
	})
	store, err := Open(t.TempDir(), downloader)
	if err != nil {
		t.Fatal(err)
	}
	urlEntry, err := store.ImportURL(context.Background(), "https://origin.example/profile")
	if err != nil {
		t.Fatal(err)
	}
	<-downloadStarted
	activationEntry := mustImport(t, store, "activation")
	source := filepath.Join(t.TempDir(), "blocked.yaml")
	if err := os.WriteFile(source, []byte("imported"), 0o600); err != nil {
		t.Fatal(err)
	}
	txn, err := store.PrepareActivation(activationEntry.ID)
	if err != nil {
		t.Fatal(err)
	}

	refreshDone := make(chan error, 1)
	go func() {
		_, err := store.RefreshURL(context.Background(), urlEntry.ID)
		refreshDone <- err
	}()
	importDone := make(chan error, 1)
	go func() {
		_, err := store.ImportFile(context.Background(), source)
		importDone <- err
	}()
	select {
	case <-downloadStarted:
		t.Fatal("refresh passed pending activation")
	case err := <-importDone:
		t.Fatalf("import passed pending activation: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := store.AbortActivation(txn); err != nil {
		t.Fatal(err)
	}
	for name, done := range map[string]<-chan error{"refresh": refreshDone, "import": importDone} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("%s after abort: %v", name, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s remained blocked after abort", name)
		}
	}
}

func activateForTest(t *testing.T, store *Store, id string) {
	t.Helper()
	txn, err := store.PrepareActivation(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CommitActivation(txn); err != nil {
		t.Fatal(err)
	}
}

func assertProfileUnchanged(t *testing.T, store *Store, id, wantData string, want Snapshot) {
	t.Helper()
	data, err := store.Read(id)
	if err != nil || string(data) != wantData {
		t.Fatalf("profile bytes = %q, %v; want %q", data, err, wantData)
	}
	got := store.Snapshot()
	if !snapshotsEqual(got, want) {
		t.Fatalf("Snapshot() = %+v, want %+v", got, want)
	}
}

func snapshotsEqual(left, right Snapshot) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func validEntry(id string, kind Kind) Entry {
	timestamp := time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC)
	entry := Entry{
		ID:        id,
		Name:      "profile.yaml",
		Kind:      kind,
		File:      profileFileName(id),
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}
	if kind == KindURL {
		entry.URL = "https://origin.example/profile.yaml"
	}
	return entry
}

func writeMetadata(t *testing.T, root string, state Snapshot) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "profiles"), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profiles.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeOwnedProfile(t *testing.T, root, id, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "profiles"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profiles", id+".yaml"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustStore(t *testing.T, root string) *Store {
	t.Helper()
	store, err := Open(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func mustImport(t *testing.T, store *Store, data string) Entry {
	t.Helper()
	source := filepath.Join(t.TempDir(), "source.yaml")
	if err := os.WriteFile(source, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	entry, err := store.ImportFile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func setActiveID(store *Store, id string) {
	store.stateMu.Lock()
	store.state.ActiveID = id
	store.stateMu.Unlock()
}

func assertNoOwnedTemps(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{root, filepath.Join(root, "profiles")} {
		matches, err := filepath.Glob(filepath.Join(dir, ".mihomotui-*.tmp"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("owned temporary artifacts remain in %q: %v", dir, matches)
		}
	}
}

func assertExists(t *testing.T, name string) {
	t.Helper()
	if _, err := os.Lstat(name); err != nil {
		t.Fatalf("expected %q to exist: %v", name, err)
	}
}

func assertNotExists(t *testing.T, name string) {
	t.Helper()
	if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be absent, got %v", name, err)
	}
}

func testID(number int) string {
	return fmt.Sprintf("%032x", number)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type contextDeleter interface {
	Delete(context.Context, string) error
}

func contextDeleterForTest(t *testing.T, store *Store) contextDeleter {
	t.Helper()
	deleter, ok := any(store).(contextDeleter)
	if !ok {
		t.Fatal("Store.Delete does not accept context.Context")
	}
	return deleter
}
