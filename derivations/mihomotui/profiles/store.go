// Package profiles owns persisted Mihomo profile files and their metadata.
package profiles

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MaxProfileBytes is the largest profile or metadata document the store accepts.
const MaxProfileBytes int64 = 32 << 20

var (
	// ErrActiveProfile is returned when attempting to delete the active profile.
	ErrActiveProfile = errors.New("active profile cannot be deleted")
	// ErrActiveIDPersistence means active.yaml was replaced but its profile ID was not persisted.
	ErrActiveIDPersistence = errors.New("active profile ID persistence failed")
	// ErrProfileNotFound is returned when an ID does not exist in the store.
	ErrProfileNotFound = errors.New("profile not found")
)

// Kind identifies how a profile entered the store.
type Kind string

const (
	KindFile Kind = "file"
	KindURL  Kind = "url"
)

// Entry is one owned profile and its immutable source metadata.
type Entry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      Kind      `json:"kind"`
	File      string    `json:"file"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Snapshot is the complete persisted profile index.
type Snapshot struct {
	Version  int     `json:"version"`
	ActiveID string  `json:"active_id,omitempty"`
	Profiles []Entry `json:"profiles"`
}

// Downloader is intentionally structural so api.Client can satisfy it directly.
type Downloader interface {
	DownloadProfile(context.Context, string) ([]byte, string, error)
}

// Store keeps profile metadata and content under one application-owned root.
type Store struct {
	root         string
	downloader   Downloader
	stateMu      sync.RWMutex
	mutationMu   sync.Mutex
	state        Snapshot
	now          func() time.Time
	random       io.Reader
	writeIndexFn func(Snapshot) error
	replaceFn    func(string, string) error
}

// Activation holds a staged boot snapshot while Mihomo applies Payload.
type Activation struct {
	ID      string
	Payload []byte

	staged  string
	store   *Store
	release func()
	done    bool
}

// Open creates or opens an application-owned profile store rooted at root.
func Open(root string, downloader Downloader) (*Store, error) {
	if root == "" {
		return nil, errors.New("profile store root is empty")
	}
	root = filepath.Clean(root)
	if err := mkdirPrivate(root); err != nil {
		return nil, fmt.Errorf("create profile store root: %w", err)
	}
	if err := mkdirPrivate(filepath.Join(root, profileDirectoryName)); err != nil {
		return nil, fmt.Errorf("create profiles directory: %w", err)
	}

	store := &Store{
		root:       root,
		downloader: downloader,
		now:        time.Now,
		random:     rand.Reader,
		replaceFn:  replaceFile,
	}

	state, exists, err := store.loadIndex()
	if err != nil {
		return nil, err
	}
	if !exists {
		state = emptySnapshot()
	}
	if err := validateSnapshot(state); err != nil {
		return nil, fmt.Errorf("validate profiles metadata: %w", err)
	}
	if err := store.reconcileOwnedArtifacts(state); err != nil {
		return nil, fmt.Errorf("reconcile profile artifacts: %w", err)
	}
	if err := store.validateReferencedFiles(state); err != nil {
		return nil, fmt.Errorf("validate profile files: %w", err)
	}
	if exists {
		if err := chmodPrivate(store.indexPath()); err != nil {
			return nil, fmt.Errorf("secure profiles metadata: %w", err)
		}
	}

	store.state = cloneSnapshot(state)
	store.writeIndexFn = store.writeIndex
	return store, nil
}

// Snapshot returns a copy of the current in-memory metadata.
func (s *Store) Snapshot() Snapshot {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return cloneSnapshot(s.state)
}

// Read returns a bounded copy of the owned profile content identified by id.
func (s *Store) Read(id string) ([]byte, error) {
	s.stateMu.RLock()
	found := false
	for _, entry := range s.state.Profiles {
		if entry.ID == id {
			found = true
			break
		}
	}
	s.stateMu.RUnlock()
	if !found {
		return nil, ErrProfileNotFound
	}

	profilePath := s.profilePath(id)
	info, err := os.Lstat(profilePath)
	if err != nil {
		return nil, fmt.Errorf("stat profile %q: %w", id, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("profile %q is not a regular file", id)
	}

	file, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("open profile %q: %w", id, err)
	}
	data, readErr := readBounded(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read profile %q: %w", id, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close profile %q: %w", id, closeErr)
	}
	return data, nil
}

func (s *Store) publish(next Snapshot) {
	s.stateMu.Lock()
	s.state = cloneSnapshot(next)
	s.stateMu.Unlock()
}

// PrepareActivation serializes profile mutations until CommitActivation or
// AbortActivation completes the transaction.
func (s *Store) PrepareActivation(id string) (*Activation, error) {
	s.mutationMu.Lock()
	release := true
	defer func() {
		if release {
			s.mutationMu.Unlock()
		}
	}()

	payload, err := s.Read(id)
	if err != nil {
		return nil, err
	}
	staged, err := stage(s.activePath(), payload)
	if err != nil {
		return nil, fmt.Errorf("stage active profile: %w", err)
	}

	release = false
	return &Activation{
		ID:      id,
		Payload: append([]byte(nil), payload...),
		staged:  staged,
		store:   s,
		release: s.mutationMu.Unlock,
	}, nil
}

func (s *Store) validateActivation(txn *Activation) error {
	if txn == nil {
		return errors.New("activation transaction is nil")
	}
	if txn.store != s {
		return errors.New("activation transaction belongs to another store")
	}
	if txn.done || txn.release == nil || txn.staged == "" {
		return errors.New("activation transaction is not pending")
	}
	return nil
}
