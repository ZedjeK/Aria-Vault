package config

import (
	"bytes"
	"testing"

	"aria-vault/internal/dnsprov"
)

func TestExportImport(t *testing.T) {
	c := &Config{Domaine: "vault.exemple.duckdns.org", Email: "moi@exemple.fr"}
	if errs := c.SetIdentifiants(dnsprov.DuckDNS, map[string]string{"jeton": jeton}); errs != nil {
		t.Fatal(errs)
	}

	if _, err := c.Exporter("court"); err == nil {
		t.Error("phrase trop courte acceptée")
	}
	data, err := c.Exporter("ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	// Rien de lisible dans le fichier : ni jeton, ni domaine, ni e-mail.
	for _, secret := range []string{jeton, "vault.exemple", "duckdns", "moi@exemple.fr"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Errorf("%q apparaît en clair dans l'export", secret)
		}
	}

	got, err := LireExport(data, "ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if got.Identifiants["jeton"] != jeton || got.Jeton != jeton || got.Fournisseur != dnsprov.DuckDNS || got.Domaine != c.Domaine || got.Email != c.Email {
		t.Errorf("import = %+v", got)
	}

	if _, err := LireExport(data, "mauvaise phrase"); err != ErrPhrase {
		t.Errorf("mauvaise phrase : %v", err)
	}
	if _, err := LireExport(data, ""); err == nil {
		t.Error("import sans phrase accepté")
	}

	// Un seul octet modifié, dans l'en-tête ou le chiffré : refusé.
	for _, i := range []int{12, len(data) - 1} {
		altere := bytes.Clone(data)
		altere[i] ^= 1
		if _, err := LireExport(altere, "ma phrase de passe"); err != ErrPhrase {
			t.Errorf("octet %d modifié : %v", i, err)
		}
	}

	// Paramètres scrypt démesurés : refusés avant tout calcul.
	piege := bytes.Clone(data)
	piege[8] = 30
	if _, err := LireExport(piege, "ma phrase de passe"); err == nil || err == ErrPhrase {
		t.Errorf("paramètres piégés : %v", err)
	}

	if _, err := LireExport([]byte(`{"format":"aria-vault-reglages"}`), "x"); err == nil {
		t.Error("ancien format ou fichier étranger accepté")
	}

	// Deux exports de la même config ne se ressemblent pas (sel et nonce aléatoires).
	data2, _ := c.Exporter("ma phrase de passe")
	if bytes.Equal(data, data2) {
		t.Error("deux exports identiques")
	}
}

func TestExportOVH(t *testing.T) {
	c := &Config{Domaine: "vault.exemple.fr"}
	ids := map[string]string{"applicationKey": "ak123456789", "applicationSecret": "as123456789", "consumerKey": "ck123456789"}
	if errs := c.SetIdentifiants(dnsprov.OVH, ids); errs != nil {
		t.Fatal(errs)
	}
	data, err := c.Exporter("ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	got, err := LireExport(data, "ma phrase de passe")
	if err != nil {
		t.Fatal(err)
	}
	if got.Fournisseur != dnsprov.OVH || got.Identifiants["endpoint"] != "ovh-eu" || got.Identifiants["consumerKey"] != "ck123456789" || got.Jeton != "" {
		t.Errorf("import = %+v", got)
	}
}
