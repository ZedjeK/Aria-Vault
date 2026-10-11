package config

import (
	"path/filepath"
	"testing"
)

func TestNormaliserDomaine(t *testing.T) {
	ok := map[string]string{
		"https://Vault.Exemple.DuckDNS.org/": "vault.exemple.duckdns.org",
		"exemple.duckdns.org":                "exemple.duckdns.org",
	}
	for in, want := range ok {
		got, err := NormaliserDomaine(in)
		if err != nil || got != want {
			t.Errorf("%q → %q, %v ; attendu %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "duckdns.org", "vault.exemple.fr", "-x.duckdns.org", "a..duckdns.org"} {
		if _, err := NormaliserDomaine(bad); err == nil {
			t.Errorf("%q accepté", bad)
		}
	}
}

func TestSaveLoadJeton(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	c := &Config{Domaine: "vault.exemple.duckdns.org"}
	if err := c.SetJeton("pas-un-jeton"); err == nil {
		t.Error("jeton mal formé accepté")
	}
	const jeton = "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0"
	if err := c.SetJeton(jeton); err != nil {
		t.Fatal(err)
	}
	if err := c.Save(path); err != nil {
		t.Fatal(err)
	}
	l, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if j, err := l.Jeton(); err != nil || j != jeton {
		t.Errorf("jeton relu = %q, %v", j, err)
	}
	if err := l.Valider(); err != nil {
		t.Error(err)
	}
}
