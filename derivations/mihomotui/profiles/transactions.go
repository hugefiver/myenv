package profiles

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ImportFile copies a regular local file into the store.
func (s *Store) ImportFile(ctx context.Context, source string) (Entry, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	if err := contextError(ctx); err != nil {
		return Entry{}, err
	}
	abs, err := filepath.Abs(filepath.Clean(source))
	if err != nil {
		return Entry{}, fmt.Errorf("resolve profile path: %w", err)
	}
	file, err := os.Open(abs)
	if err != nil {
		return Entry{}, fmt.Errorf("open profile source: %w", err)
	}
	info, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return Entry{}, fmt.Errorf("stat profile source: %w", statErr)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return Entry{}, errors.New("profile source is not a regular file")
	}
	data, readErr := readBounded(file)
	closeErr := file.Close()
	if readErr != nil {
		return Entry{}, fmt.Errorf("read profile source: %w", readErr)
	}
	if closeErr != nil {
		return Entry{}, fmt.Errorf("close profile source: %w", closeErr)
	}
	if err := contextError(ctx); err != nil {
		return Entry{}, err
	}
	return s.importData(filepath.Base(abs), KindFile, "", data)
}

// ImportURL downloads a profile once, then stores both its bytes and source URL.
func (s *Store) ImportURL(ctx context.Context, rawURL string) (Entry, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	if err := contextError(ctx); err != nil {
		return Entry{}, err
	}
	if err := validateProfileURL(rawURL); err != nil {
		return Entry{}, err
	}
	if s.downloader == nil {
		return Entry{}, errors.New("profile downloader is not configured")
	}
	data, finalURL, err := s.downloader.DownloadProfile(ctx, rawURL)
	if err != nil {
		return Entry{}, fmt.Errorf("download profile: %w", err)
	}
	if err := contextError(ctx); err != nil {
		return Entry{}, err
	}
	if int64(len(data)) > MaxProfileBytes {
		return Entry{}, exceedsLimitError()
	}
	name, err := profileNameFromURL(finalURL)
	if err != nil {
		return Entry{}, err
	}
	return s.importData(name, KindURL, rawURL, data)
}

// Delete removes an inactive owned profile. Metadata is durable before cleanup.
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	if err := contextError(ctx); err != nil {
		return err
	}
	current := s.Snapshot()
	if current.ActiveID == id {
		return ErrActiveProfile
	}
	position := -1
	for i, entry := range current.Profiles {
		if entry.ID == id {
			position = i
			break
		}
	}
	if position == -1 {
		return ErrProfileNotFound
	}

	target := s.profilePath(id)
	deleteTemp := s.deleteTempPath(id)
	if err := s.replaceFn(target, deleteTemp); err != nil {
		return fmt.Errorf("stage profile deletion: %w", err)
	}
	if err := contextError(ctx); err != nil {
		rollbackErr := s.replaceFn(deleteTemp, target)
		if rollbackErr != nil {
			return fmt.Errorf("cancel profile deletion: %w; rollback profile deletion: %v", err, rollbackErr)
		}
		return err
	}

	next := cloneSnapshot(current)
	next.Profiles = append(next.Profiles[:position:position], next.Profiles[position+1:]...)
	if err := s.writeIndexFn(next); err != nil {
		rollbackErr := s.replaceFn(deleteTemp, target)
		if rollbackErr != nil {
			return fmt.Errorf("persist profile deletion metadata: %w; rollback profile deletion: %v", err, rollbackErr)
		}
		return fmt.Errorf("persist profile deletion metadata: %w", err)
	}

	s.publish(next)
	if err := os.Remove(deleteTemp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("profile metadata deleted; owned cleanup deferred: %w", err)
	}
	return nil
}

func (s *Store) importData(name string, kind Kind, sourceURL string, data []byte) (Entry, error) {
	current := s.Snapshot()
	id, err := s.newID(current)
	if err != nil {
		return Entry{}, err
	}
	target := s.profilePath(id)
	temporary, err := stage(target, data)
	if err != nil {
		return Entry{}, fmt.Errorf("stage profile: %w", err)
	}
	if err := s.replaceFn(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return Entry{}, fmt.Errorf("commit profile: %w", err)
	}

	now := s.now().UTC()
	entry := Entry{
		ID:        id,
		Name:      name,
		Kind:      kind,
		File:      profileFileName(id),
		URL:       sourceURL,
		CreatedAt: now,
		UpdatedAt: now,
	}
	next := cloneSnapshot(current)
	next.Profiles = append(next.Profiles, entry)
	if err := s.writeIndexFn(next); err != nil {
		cleanupErr := os.Remove(target)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			return Entry{}, fmt.Errorf("persist profile metadata: %w; remove unindexed profile: %v", err, cleanupErr)
		}
		return Entry{}, fmt.Errorf("persist profile metadata: %w", err)
	}
	s.publish(next)
	return entry, nil
}

func (s *Store) newID(state Snapshot) (string, error) {
	known := make(map[string]struct{}, len(state.Profiles))
	for _, entry := range state.Profiles {
		known[entry.ID] = struct{}{}
	}
	for {
		bytes := make([]byte, 16)
		if _, err := io.ReadFull(s.random, bytes); err != nil {
			return "", fmt.Errorf("generate profile ID: %w", err)
		}
		id := hex.EncodeToString(bytes)
		if _, exists := known[id]; !exists {
			return id, nil
		}
	}
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("profile context is nil")
	}
	return ctx.Err()
}

// AbortActivation discards a pending boot snapshot without changing state.
func (s *Store) AbortActivation(txn *Activation) error {
	if err := s.validateActivation(txn); err != nil {
		return err
	}
	txn.done = true
	err := os.Remove(txn.staged)
	txn.release()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove staged active profile: %w", err)
	}
	return nil
}

// CommitActivation installs the staged boot snapshot and then persists its ID.
func (s *Store) CommitActivation(txn *Activation) error {
	if err := s.validateActivation(txn); err != nil {
		return err
	}
	txn.done = true
	defer txn.release()

	if err := s.replaceFn(txn.staged, s.activePath()); err != nil {
		_ = os.Remove(txn.staged)
		return fmt.Errorf("replace active profile: %w", err)
	}

	next := s.Snapshot()
	next.ActiveID = txn.ID
	if err := s.writeIndexFn(next); err != nil {
		return fmt.Errorf("%w: %v", ErrActiveIDPersistence, err)
	}
	s.publish(next)
	return nil
}
