//go:build !windows

package paths

import (
	"os"
	"path/filepath"
)

func documents() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents")
}
