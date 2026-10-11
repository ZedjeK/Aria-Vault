package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportImport(t *testing.T) {
	const jeton = "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0"
	c := &Config{Domaine: "vault.exemple.duckdns.org", Email: "moi@exemple.fr"}
	if err := c.SetJeton(jeton); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Exporter("court"); err == nil {
		t.Error("phrase trop courte acceptée")
	}
	data, err := c.Exporter("ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), jeton) {
		t.Fatal("le jeton apparaît en clair dans l'export")
	}

	got, err := LireExport(data, "ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if got.Jeton != jeton || got.Domaine != c.Domaine || got.Email != c.Email {
		t.Errorf("import = %+v", got)
	}

	if _, err := LireExport(data, "mauvaise phrase"); err != ErrPhrase {
		t.Errorf("mauvaise phrase : %v", err)
	}

	// Sans phrase : domaine et e-mail seulement.
	sans, err := LireExport(data, "")
	if err != nil || sans.Jeton != "" || sans.Domaine != c.Domaine {
		t.Errorf("import sans phrase = %+v, %v", sans, err)
	}

	// Domaine modifié dans le fichier : le jeton ne doit plus se déchiffrer.
	var f map[string]any
	json.Unmarshal(data, &f)
	f["domaine"] = "pirate.duckdns.org"
	altere, _ := json.Marshal(f)
	if _, err := LireExport(altere, "ma phrase de passe"); err != ErrPhrase {
		t.Errorf("domaine altéré : %v", err)
	}

	if _, err := LireExport([]byte(`{"format":"autre"}`), "x"); err == nil {
		t.Error("fichier étranger accepté")
	}
}
