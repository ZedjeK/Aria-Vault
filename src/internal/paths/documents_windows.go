//go:build windows

package paths

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func documents() string {
	if d, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0); err == nil && d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents")
}
