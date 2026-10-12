package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistant(t *testing.T) {
	racine := t.TempDir()
	docs := filepath.Join(racine, "Documents")
	os.MkdirAll(docs, 0o700)

	if got := Existant(filepath.Join(docs, "Aria Vault", "Sauvegardes")); got != docs {
		t.Errorf("dossier absent : %q, attendu le parent existant %q", got, docs)
	}
	if got := Existant(docs); got != docs {
		t.Errorf("dossier existant : %q", got)
	}
	if _, err := os.Stat(filepath.Join(docs, "Aria Vault")); err == nil {
		t.Error("aucun dossier ne doit être créé")
	}
}
