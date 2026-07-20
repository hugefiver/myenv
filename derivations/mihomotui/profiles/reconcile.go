package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ownedProfilePattern = regexp.MustCompile(`^([0-9a-f]{32})\.yaml$`)
	deleteTempPattern   = regexp.MustCompile(`^\.mihomotui-delete-([0-9a-f]{32})\.tmp$`)
)

func (s *Store) reconcileOwnedArtifacts(state Snapshot) error {
	referenced := make(map[string]struct{}, len(state.Profiles))
	for _, entry := range state.Profiles {
		referenced[entry.ID] = struct{}{}
	}

	profilesDir := filepath.Join(s.root, profileDirectoryName)
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if matches := deleteTempPattern.FindStringSubmatch(name); matches != nil {
			id := matches[1]
			temporary := filepath.Join(profilesDir, name)
			if _, keep := referenced[id]; keep {
				canonical := s.profilePath(id)
				_, statErr := os.Lstat(canonical)
				if errors.Is(statErr, os.ErrNotExist) {
					if err := s.replaceFn(temporary, canonical); err != nil {
						return fmt.Errorf("restore profile %q: %w", id, err)
					}
				} else if statErr != nil {
					return statErr
				} else if err := removeOwnedPath(temporary); err != nil {
					return err
				}
			} else if err := removeOwnedPath(temporary); err != nil {
				return err
			}
			continue
		}
		if isStageTemp(name) {
			if err := removeOwnedPath(filepath.Join(profilesDir, name)); err != nil {
				return err
			}
		}
	}

	rootEntries, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, entry := range rootEntries {
		if isStageTemp(entry.Name()) {
			if err := removeOwnedPath(filepath.Join(s.root, entry.Name())); err != nil {
				return err
			}
		}
	}

	entries, err = os.ReadDir(profilesDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		matches := ownedProfilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		if _, keep := referenced[matches[1]]; !keep {
			if err := removeOwnedPath(filepath.Join(profilesDir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) validateReferencedFiles(state Snapshot) error {
	for _, entry := range state.Profiles {
		profilePath := s.profilePath(entry.ID)
		info, err := os.Lstat(profilePath)
		if err != nil {
			return fmt.Errorf("profile %q: %w", entry.ID, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("profile %q is not a regular file", entry.ID)
		}
		if err := chmodPrivate(profilePath); err != nil {
			return err
		}
	}
	return nil
}

func isStageTemp(name string) bool {
	return strings.HasPrefix(name, ".mihomotui-") && strings.HasSuffix(name, ".tmp")
}

func removeOwnedPath(name string) error {
	return os.RemoveAll(name)
}
