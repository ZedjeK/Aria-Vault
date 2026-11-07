//go:build !windows

package paths

import (
	"os"
	"path/filepath"
)

// documents : hors Windows (tests), ~/Documents.
func documents() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents")
}
