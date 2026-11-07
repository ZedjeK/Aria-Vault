package vaultwarden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFermerAdmin(t *testing.T) {
	dir := t.TempDir()
	chemin := filepath.Join(dir, "config.json")
	// Tel qu'un panneau /admin l'écrit, avec une casse inattendue en plus.
	ecrit := `{
  "domain": "https://ancien.exemple.fr",
  "admin_token": "$argon2id$v=19$m=65540,t=3,p=4$c2Vs$aGFjaGU",
  "DISABLE_ADMIN_TOKEN": true,
  "signups_allowed": false,
  "smtp_port": 587
}`
	if err := os.WriteFile(chemin, []byte(ecrit), 0o600); err != nil {
		t.Fatal(err)
	}
	retirees, err := FermerAdmin(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"DISABLE_ADMIN_TOKEN", "admin_token"}; !slices.Equal(retirees, want) {
		t.Errorf("clés retirées = %v, attendu %v", retirees, want)
	}
	data, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	var reste map[string]any
	if err := json.Unmarshal(data, &reste); err != nil {
		t.Fatalf("config.json réécrit illisible : %v", err)
	}
	for k := range reste {
		if strings.Contains(strings.ToLower(k), "admin") {
			t.Errorf("%s toujours présent : /admin pourrait se rouvrir", k)
		}
	}
	if reste["domain"] != "https://ancien.exemple.fr" || reste["signups_allowed"] != false || reste["smtp_port"] != float64(587) {
		t.Errorf("les autres réglages doivent être gardés : %v", reste)
	}
	msg := MessageAdminFerme(retirees)
	if !strings.Contains(msg, "/admin") || !strings.Contains(msg, "admin_token") {
		t.Errorf("message peu clair : %q", msg)
	}

	// Deuxième passage : rien à faire, le fichier n'est pas réécrit.
	avant, _ := os.Stat(chemin)
	if retirees, err := FermerAdmin(dir); err != nil || len(retirees) != 0 {
		t.Errorf("second passage = %v, %v", retirees, err)
	}
	if apres, _ := os.Stat(chemin); !apres.ModTime().Equal(avant.ModTime()) {
		t.Error("config.json sain réécrit pour rien")
	}
}

func TestFermerAdminSansFichier(t *testing.T) {
	if retirees, err := FermerAdmin(t.TempDir()); err != nil || retirees != nil {
		t.Errorf("sans config.json = %v, %v", retirees, err)
	}
}

func TestFermerAdminSansJeton(t *testing.T) {
	dir := t.TempDir()
	chemin := filepath.Join(dir, "config.json")
	ecrit := `{"domain": "https://ancien.exemple.fr"}`
	if err := os.WriteFile(chemin, []byte(ecrit), 0o600); err != nil {
		t.Fatal(err)
	}
	if retirees, err := FermerAdmin(dir); err != nil || len(retirees) != 0 {
		t.Errorf("config.json sans jeton = %v, %v", retirees, err)
	}
	if b, _ := os.ReadFile(chemin); string(b) != ecrit {
		t.Errorf("config.json sans jeton modifié : %q", b)
	}
}

func TestFermerAdminIllisible(t *testing.T) {
	dir := t.TempDir()
	// Un fichier qu'Aria ne sait pas relire ne doit pas passer en silence.
	for _, contenu := range []string{`{"admin_token": "x"`, `["admin_token"]`, "\ufeff{\"admin_token\": \"x\"}"} {
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(contenu), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := FermerAdmin(dir); err == nil {
			t.Errorf("%q accepté sans vérification", contenu)
		}
	}
}
