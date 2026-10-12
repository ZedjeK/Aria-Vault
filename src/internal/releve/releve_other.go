//go:build !windows

package releve

import (
	"errors"
	"os"
	"time"
)

// Hors Windows (tests uniquement) : pas de relève entre instances.

func EcouterFermeture(func()) error        { return nil }
func demanderFermeture() error             { return errors.New("non pris en charge") }
func vivante(uint32, int64) bool           { return false }
func attendre(uint32, time.Duration) bool  { return true }
func terminer(uint32, time.Duration) error { return nil }
func anciennes03() []ancienne              { return nil }
func creationDe(pid uint32) (int64, error) {
	if pid != uint32(os.Getpid()) {
		return 0, errors.New("non pris en charge")
	}
	return 1, nil
}
