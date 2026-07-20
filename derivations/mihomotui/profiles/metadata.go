package profiles

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

var profileIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func (s *Store) loadIndex() (Snapshot, bool, error) {
	index := s.indexPath()
	info, err := os.Lstat(index)
	if errors.Is(err, os.ErrNotExist) {
		return Snapshot{}, false, nil
	}
	if err != nil {
		return Snapshot{}, false, fmt.Errorf("stat profiles metadata: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Snapshot{}, false, errors.New("profiles metadata is not a regular file")
	}

	file, err := os.Open(index)
	if err != nil {
		return Snapshot{}, false, fmt.Errorf("open profiles metadata: %w", err)
	}
	data, readErr := readBounded(file)
	closeErr := file.Close()
	if readErr != nil {
		return Snapshot{}, false, fmt.Errorf("read profiles metadata: %w", readErr)
	}
	if closeErr != nil {
		return Snapshot{}, false, fmt.Errorf("close profiles metadata: %w", closeErr)
	}

	state, err := decodeSnapshot(data)
	if err != nil {
		return Snapshot{}, false, fmt.Errorf("decode profiles metadata: %w", err)
	}
	return state, true, nil
}

func (s *Store) writeIndex(state Snapshot) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profiles metadata: %w", err)
	}
	data = append(data, '\n')
	return atomicWrite(s.indexPath(), data, s.replaceFn)
}

func emptySnapshot() Snapshot {
	return Snapshot{Version: 1, Profiles: []Entry{}}
}

func cloneSnapshot(state Snapshot) Snapshot {
	copy := state
	copy.Profiles = append([]Entry(nil), state.Profiles...)
	if copy.Profiles == nil {
		copy.Profiles = []Entry{}
	}
	return copy
}

func validateSnapshot(state Snapshot) error {
	if state.Version != 1 {
		return fmt.Errorf("unsupported profiles metadata version %d", state.Version)
	}
	known := make(map[string]struct{}, len(state.Profiles))
	for _, entry := range state.Profiles {
		if !profileIDPattern.MatchString(entry.ID) {
			return fmt.Errorf("invalid profile ID %q", entry.ID)
		}
		if _, exists := known[entry.ID]; exists {
			return fmt.Errorf("duplicate profile ID %q", entry.ID)
		}
		known[entry.ID] = struct{}{}
		if entry.Name == "" {
			return fmt.Errorf("profile %q has an empty name", entry.ID)
		}
		if entry.Kind != KindFile && entry.Kind != KindURL {
			return fmt.Errorf("profile %q has invalid kind %q", entry.ID, entry.Kind)
		}
		if entry.File != profileFileName(entry.ID) {
			return fmt.Errorf("profile %q has invalid owned path %q", entry.ID, entry.File)
		}
		if entry.CreatedAt.IsZero() || entry.UpdatedAt.IsZero() {
			return fmt.Errorf("profile %q has zero timestamps", entry.ID)
		}
		if entry.UpdatedAt.Before(entry.CreatedAt) {
			return fmt.Errorf("profile %q updated before creation", entry.ID)
		}
		switch entry.Kind {
		case KindFile:
			if entry.URL != "" {
				return fmt.Errorf("file profile %q has a URL", entry.ID)
			}
		case KindURL:
			if err := validateProfileURL(entry.URL); err != nil {
				return fmt.Errorf("URL profile %q: %w", entry.ID, err)
			}
		}
	}
	if state.ActiveID != "" {
		if _, exists := known[state.ActiveID]; !exists {
			return fmt.Errorf("active profile %q does not exist", state.ActiveID)
		}
	}
	return nil
}

func decodeSnapshot(data []byte) (Snapshot, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var state Snapshot
	if err := decoder.Decode(&state); err != nil {
		return Snapshot{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Snapshot{}, errors.New("trailing JSON in profiles metadata")
		}
		return Snapshot{}, fmt.Errorf("trailing data in profiles metadata: %w", err)
	}
	return state, nil
}

func validateProfileURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid profile URL: %w", err)
	}
	if !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("invalid profile URL %q", rawURL)
	}
	return nil
}

func profileNameFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid final profile URL: %w", err)
	}
	name := path.Base(strings.Trim(parsed.Path, "/"))
	if name == "." || name == "" {
		name = parsed.Hostname()
	}
	if name == "" {
		return "", fmt.Errorf("invalid final profile URL %q", rawURL)
	}
	return name, nil
}

func profileFileName(id string) string {
	return profileDirectoryName + "/" + id + ".yaml"
}
