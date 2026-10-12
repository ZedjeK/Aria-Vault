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

func TestSurOneDrive(t *testing.T) {
	racine := t.TempDir()
	t.Setenv("OneDrive", filepath.Join(racine, "OneDrive"))
	t.Setenv("OneDriveConsumer", "")
	t.Setenv("OneDriveCommercial", "")
	if !SurOneDrive(filepath.Join(racine, "OneDrive", "Aria Vault")) {
		t.Error("sous-dossier OneDrive non reconnu")
	}
	if SurOneDrive(filepath.Join(racine, "OneDriveFaux", "x")) || SurOneDrive(filepath.Join(racine, "Documents")) {
		t.Error("faux positif OneDrive")
	}
}

func TestMigrer(t *testing.T) {
	racine := t.TempDir()
	ancien := filepath.Join(racine, AncienDossier)
	os.MkdirAll(filepath.Join(ancien, "data"), 0o700)
	os.WriteFile(filepath.Join(ancien, "data", "db.sqlite3"), []byte("coffre"), 0o600)

	p, msg, err := migrer(racine, 1, 0)
	if err != nil || msg == "" {
		t.Fatalf("migration : %v (msg %q)", err, msg)
	}
	if p.Base != filepath.Join(racine, Dossier) {
		t.Errorf("base = %q", p.Base)
	}
	if b, _ := os.ReadFile(filepath.Join(p.Donnees, "db.sqlite3")); string(b) != "coffre" {
		t.Errorf("coffre migré = %q", b)
	}
	if existe(ancien) {
		t.Error("l'ancien dossier doit avoir été renommé")
	}

	// Deuxième lancement : rien à faire.
	if p2, msg2, err := migrer(racine, 1, 0); err != nil || msg2 != "" || p2.Base != p.Base {
		t.Errorf("relance : %+v %q %v", p2, msg2, err)
	}
}

func TestMigrerNeToucheJamaisAuNouveau(t *testing.T) {
	racine := t.TempDir()
	os.MkdirAll(filepath.Join(racine, AncienDossier), 0o700)
	os.MkdirAll(filepath.Join(racine, Dossier), 0o700)
	p, msg, err := migrer(racine, 1, 0)
	if err != nil || msg != "" || p.Base != filepath.Join(racine, Dossier) {
		t.Errorf("%+v %q %v", p, msg, err)
	}
	if !existe(filepath.Join(racine, AncienDossier)) {
		t.Error("l'ancien dossier ne doit pas être supprimé quand le nouveau existe déjà")
	}
}

func TestMigrerPremierLancement(t *testing.T) {
	racine := t.TempDir()
	if p, msg, err := migrer(racine, 1, 0); err != nil || msg != "" || p.Base != filepath.Join(racine, Dossier) {
		t.Errorf("%+v %q %v", p, msg, err)
	}
}
