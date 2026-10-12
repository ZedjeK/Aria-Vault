package moteur

import (
	"path/filepath"
	"strings"
	"testing"

	"aria-vault/internal/journal"
	"aria-vault/internal/paths"
)

func TestStartSansConfiguration(t *testing.T) {
	p := paths.In(t.TempDir())
	changes := 0
	m := New(p, journal.New(filepath.Join(p.Base, "aria.log")), func() { changes++ })

	if m.Configure() {
		t.Fatal("un dossier vide ne doit pas être considéré comme configuré")
	}
	err := m.Start()
	if err == nil || !strings.Contains(err.Error(), "configuration HTTPS incomplète") {
		t.Fatalf("Start sans configuration : %v", err)
	}
	if s := m.Status(); s.Etat != Arrete {
		t.Errorf("état = %s, attendu arrete", s.Etat)
	}
	if changes != 0 {
		t.Errorf("onChange appelé %d fois sans changement d'état", changes)
	}

	// Stop sans démarrage : sans effet, sans blocage.
	m.Stop()
	if s := m.Status(); s.Etat != Arrete {
		t.Errorf("état après Stop = %s", s.Etat)
	}
}
