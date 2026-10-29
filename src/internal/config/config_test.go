package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aria-vault/internal/dnsprov"
)

const jeton = "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0"

func TestSaveLoadIdentifiants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	c := &Config{Domaine: "vault.exemple.duckdns.org"}
	if errs := c.SetIdentifiants(dnsprov.DuckDNS, map[string]string{"jeton": "pas-un-jeton"}); errs["jeton"] == "" {
		t.Error("jeton mal formé accepté")
	}
	if errs := c.SetIdentifiants(dnsprov.DuckDNS, map[string]string{"jeton": " " + jeton + " "}); errs != nil {
		t.Fatal(errs)
	}
	if err := c.Save(path); err != nil {
		t.Fatal(err)
	}
	l, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if ids, err := l.Identifiants(); err != nil || ids["jeton"] != jeton {
		t.Errorf("identifiants relus = %v, %v", ids, err)
	}
	if err := l.Valider(); err != nil {
		t.Error(err)
	}
}

// Les réglages écrits par la v0.5 (jetonDuckDNS seul) restent valides.
func TestAncienFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	// Comme la v0.5 : le jeton seul, chiffré avec DPAPI sous Windows.
	enc, err := protect([]byte(jeton))
	if err != nil {
		t.Fatal(err)
	}
	ancien, _ := json.Marshal(map[string]any{"domaine": "exemple.duckdns.org", "jetonDuckDNS": enc})
	if err := os.WriteFile(path, ancien, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.FournisseurDNS() != dnsprov.DuckDNS || !c.AIdentifiants() {
		t.Fatalf("fournisseur %q", c.FournisseurDNS())
	}
	if err := c.Valider(); err != nil {
		t.Fatal(err)
	}
	// Un nouvel enregistrement passe au format générique.
	if errs := c.SetIdentifiants(dnsprov.Cloudflare, map[string]string{"jeton": "abcdefghijklmnopqrstuvwxyz0123456789ABCD"}); errs != nil {
		t.Fatal(errs)
	}
	if c.JetonChiffre != nil || c.Fournisseur != dnsprov.Cloudflare {
		t.Error("ancien jeton conservé")
	}
}

func TestPhraseSauvegarde(t *testing.T) {
	var c Config
	if p, err := c.PhraseSauvegarde(); p != "" || err != nil {
		t.Fatal(p, err)
	}
	if err := c.SetPhraseSauvegarde("ma phrase secrète"); err != nil {
		t.Fatal(err)
	}
	if p, _ := c.PhraseSauvegarde(); p != "ma phrase secrète" {
		t.Error(p)
	}
	_ = c.SetPhraseSauvegarde("")
	if c.PhraseChiffree != nil {
		t.Error("phrase non effacée")
	}
}
