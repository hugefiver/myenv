package profiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func readBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxProfileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxProfileBytes {
		return nil, exceedsLimitError()
	}
	return data, nil
}

func exceedsLimitError() error {
	return fmt.Errorf("profile exceeds %d bytes", MaxProfileBytes)
}

func stage(target string, data []byte) (temporary string, err error) {
	file, err := os.CreateTemp(filepath.Dir(target), ".mihomotui-*.tmp")
	if err != nil {
		return "", err
	}
	temporary = file.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = file.Close()
			_ = os.Remove(temporary)
		}
	}()

	if runtime.GOOS != "windows" {
		if err := file.Chmod(0o600); err != nil {
			return "", err
		}
	}
	for len(data) > 0 {
		written, writeErr := file.Write(data)
		if writeErr != nil {
			return "", writeErr
		}
		if written == 0 {
			return "", io.ErrShortWrite
		}
		data = data[written:]
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	cleanup = false
	return temporary, nil
}

func atomicWrite(target string, data []byte, replace func(string, string) error) error {
	temporary, err := stage(target, data)
	if err != nil {
		return err
	}
	if err := replace(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func mkdirPrivate(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(dir, 0o700)
	}
	return nil
}

func chmodPrivate(name string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(name, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RefreshURL replaces one URL-backed profile while preserving its identity.
func (s *Store) RefreshURL(ctx context.Context, id string) (Entry, error) {
	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	if err := contextError(ctx); err != nil {
		return Entry{}, err
	}
	current := s.Snapshot()
	position := profilePosition(current, id)
	if position < 0 {
		return Entry{}, ErrProfileNotFound
	}
	entry := current.Profiles[position]
	if entry.Kind != KindURL {
		return Entry{}, fmt.Errorf("profile %q is not URL-backed", id)
	}
	if s.downloader == nil {
		return Entry{}, errors.New("profile downloader is not configured")
	}

	oldData, err := s.Read(id)
	if err != nil {
		return Entry{}, err
	}
	data, _, err := s.downloader.DownloadProfile(ctx, entry.URL)
	if err != nil {
		return Entry{}, fmt.Errorf("download profile: %w", err)
	}
	if err := contextError(ctx); err != nil {
		return Entry{}, err
	}
	if int64(len(data)) > MaxProfileBytes {
		return Entry{}, exceedsLimitError()
	}

	target := s.profilePath(id)
	replacement, err := stage(target, data)
	if err != nil {
		return Entry{}, fmt.Errorf("stage refreshed profile: %w", err)
	}
	rollback, err := stage(target, oldData)
	if err != nil {
		_ = os.Remove(replacement)
		return Entry{}, fmt.Errorf("stage refresh rollback: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(replacement)
		_ = os.Remove(rollback)
	}
	if err := contextError(ctx); err != nil {
		cleanup()
		return Entry{}, err
	}
	if err := s.replaceFn(replacement, target); err != nil {
		cleanup()
		return Entry{}, fmt.Errorf("replace refreshed profile: %w", err)
	}
	if err := contextError(ctx); err != nil {
		if rollbackErr := s.replaceFn(rollback, target); rollbackErr != nil {
			cleanup()
			return Entry{}, fmt.Errorf("refresh canceled: %w; rollback profile: %v", err, rollbackErr)
		}
		cleanup()
		return Entry{}, err
	}

	entry.UpdatedAt = s.now().UTC()
	next := cloneSnapshot(current)
	next.Profiles[position] = entry
	if err := s.writeIndexFn(next); err != nil {
		if rollbackErr := s.replaceFn(rollback, target); rollbackErr != nil {
			cleanup()
			return Entry{}, fmt.Errorf("persist refreshed profile metadata: %w; rollback profile: %v", err, rollbackErr)
		}
		cleanup()
		return Entry{}, fmt.Errorf("persist refreshed profile metadata: %w", err)
	}

	s.publish(next)
	cleanup()
	return entry, nil
}

func profilePosition(state Snapshot, id string) int {
	for i := range state.Profiles {
		if state.Profiles[i].ID == id {
			return i
		}
	}
	return -1
}
