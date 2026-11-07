//go:build !windows

package releve

import (
	"errors"
	"os"
	"time"
)

// Hors Windows (tests uniquement) : pas de relève entre instances.

// EcouterFermeture : hors Windows (tests), aucune autre instance à écouter.
func EcouterFermeture(func()) error { return nil }

// demanderFermeture : hors Windows, rien à signaler.
func demanderFermeture() error { return errors.New("non pris en charge") }

// vivante : hors Windows, aucun processus n'est suivi.
func vivante(uint32, int64) bool { return false }

// attendre : hors Windows, rien à attendre.
func attendre(uint32, time.Duration) bool { return true }

// terminer : hors Windows, rien à arrêter.
func terminer(uint32, time.Duration) error { return nil }

// anciennes03 : hors Windows, pas d'Aria Vaultwarden à chercher.
func anciennes03() []ancienne { return nil }

// creationDe : hors Windows, date de création inconnue.
func creationDe(pid uint32) (int64, error) {
	if pid != uint32(os.Getpid()) {
		return 0, errors.New("non pris en charge")
	}
	return 1, nil
}
