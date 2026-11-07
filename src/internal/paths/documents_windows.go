//go:build windows

// Dossier Documents de l'utilisateur, tel que Windows le connaît (il peut
// avoir été déplacé, par exemple dans OneDrive).

package paths

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// documents interroge Windows (FOLDERID_Documents) plutôt que de deviner le chemin.
func documents() string {
	if d, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0); err == nil && d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents")
}
