package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const profileDirectoryName = "profiles"

// ResolveDataDir returns the platform-specific application data directory.
func ResolveDataDir() (string, error) {
	return resolveDataDir(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

func resolveDataDir(goos string, getenv func(string) string, home func() (string, error)) (string, error) {
	if root := getenv("MIHOMOTUI_DATA_DIR"); root != "" {
		return root, nil
	}

	switch goos {
	case "linux":
		if root := getenv("XDG_DATA_HOME"); root != "" {
			return filepath.Join(root, "mihomotui"), nil
		}
		root, err := home()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(root, ".local", "share", "mihomotui"), nil
	case "windows":
		root := getenv("LOCALAPPDATA")
		if root == "" {
			return "", errors.New("LOCALAPPDATA is required on windows")
		}
		return filepath.Join(root, "mihomotui"), nil
	case "darwin":
		root, err := home()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(root, "Library", "Application Support", "mihomotui"), nil
	default:
		return "", fmt.Errorf("unsupported platform %q", goos)
	}
}

func (s *Store) indexPath() string {
	return filepath.Join(s.root, "profiles.json")
}

func (s *Store) profilePath(id string) string {
	return filepath.Join(s.root, profileDirectoryName, id+".yaml")
}

func (s *Store) activePath() string {
	return filepath.Join(s.root, "active.yaml")
}

func (s *Store) deleteTempPath(id string) string {
	return filepath.Join(s.root, profileDirectoryName, ".mihomotui-delete-"+id+".tmp")
}
