//go:build windows

package demarrage

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const cleRun = `Software\Microsoft\Windows\CurrentVersion\Run`

// Aligner migre l'ancienne entrée (v0.3) et fait pointer l'entrée active vers
// exe. Ne crée rien si le lancement à l'ouverture de session est désactivé.
// Renvoie une description de ce qui a changé ("" si rien).
func Aligner(exe string, args ...string) (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, cleRun, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer k.Close()

	_, _, errAncien := k.GetStringValue(Ancien)
	actuel, _, errActuel := k.GetStringValue(Actuel)
	aAncien, aActuel := errAncien == nil, errActuel == nil
	if !aAncien && !aActuel {
		return "", nil // démarrage automatique désactivé : on n'y touche pas
	}

	var faits []string
	if aAncien {
		if err := k.DeleteValue(Ancien); err != nil {
			return "", err
		}
		faits = append(faits, "ancienne entrée Aria Vaultwarden retirée")
	}
	if !aActuel || !strings.EqualFold(Exe(actuel), exe) {
		if err := k.SetStringValue(Actuel, Commande(exe, args...)); err != nil {
			return "", err
		}
		faits = append(faits, "pointe vers "+exe)
	}
	if len(faits) == 0 {
		return "", nil
	}
	return fmt.Sprintf("Lancement à l'ouverture de session : %s", strings.Join(faits, ", ")), nil
}
