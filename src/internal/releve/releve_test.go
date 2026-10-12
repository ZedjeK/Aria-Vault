package releve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlusRecente(t *testing.T) {
	cas := []struct {
		a, b string
		want bool
	}{
		{"0.4.0", "0.3.2", true},
		{"0.4.10", "0.4.9", true},
		{"1.0.0", "0.99.99", true},
		{"0.4.0", "0.4.0", false},
		{"0.3.2", "0.4.0", false},
		{"v0.4.1", "0.4.0", true},
		{"dev", "0.4.0", false}, // une compilation de développement ne prend jamais la place
		{"0.4.0", "dev", true},
		{"0.4", "0.3.0", false}, // format invalide
	}
	for _, c := range cas {
		if got := PlusRecente(c.a, c.b); got != c.want {
			t.Errorf("PlusRecente(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestEnregistrerLireEffacer(t *testing.T) {
	base := t.TempDir()
	if err := Enregistrer(base, "0.4.0"); err != nil {
		t.Fatal(err)
	}
	inst, err := lire(base)
	if err != nil || inst.PID != uint32(os.Getpid()) || inst.Version != "0.4.0" || inst.Creation == 0 {
		t.Fatalf("instance relue = %+v, %v", inst, err)
	}
	// Une instance qui se décrit elle-même n'est jamais fermée par Reprendre.
	if msgs := Reprendre(base, "9.9.9"); len(msgs) != 0 {
		t.Errorf("Reprendre sur soi-même : %v", msgs)
	}
	Effacer(base)
	if _, err := os.Stat(filepath.Join(base, fichier)); err == nil {
		t.Error("instance.json devrait être effacé")
	}
}

func TestEffacerNeTouchePasAUneAutreInstance(t *testing.T) {
	base := t.TempDir()
	os.WriteFile(filepath.Join(base, fichier), []byte(`{"pid": 999999, "creation": 1, "version": "0.4.0"}`), 0o600)
	Effacer(base)
	if _, err := os.Stat(filepath.Join(base, fichier)); err != nil {
		t.Error("instance.json d'une autre instance supprimé")
	}
	// Instance périmée (PID inexistant) : aucune action.
	if msgs := Reprendre(base, "0.5.0"); len(msgs) != 0 {
		t.Errorf("instance périmée : %v", msgs)
	}
}
