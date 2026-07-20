//go:build !windows

package profiles

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
