package maj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLire(t *testing.T) {
	ok := `{"tag_name":"v0.6.1","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v0.6.1"}`
	info, err := Lire([]byte(ok), "0.6.0")
	if err != nil || !info.Disponible || info.Version != "0.6.1" {
		t.Fatalf("%+v %v", info, err)
	}
	if info, _ := Lire([]byte(ok), "0.6.1"); info.Disponible {
		t.Error("même version annoncée comme nouvelle")
	}
	if info, _ := Lire([]byte(ok), "dev"); !info.Disponible {
		t.Error("une version dev doit voir la version publiée")
	}
	for _, bad := range []string{
		`{"tag_name":"v0.6.1","html_url":"https://exemple.com/piege"}`,
		`{"tag_name":"latest","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/latest"}`,
		`{"tag_name":"v0.6.1","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v0.6.1","draft":true}`,
		`pas du json`,
	} {
		if _, err := Lire([]byte(bad), "0.6.0"); err == nil {
			t.Errorf("accepté : %s", bad)
		}
	}
}

func TestVerifier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent manquant")
		}
		w.Write([]byte(`{"tag_name":"v1.0.0","html_url":"https://github.com/ZedjeK/Aria-Vault/releases/tag/v1.0.0"}`))
	}))
	defer srv.Close()
	v := Nouveau("0.6.0")
	v.adresse = srv.URL
	info, err := v.Verifier(context.Background())
	if err != nil || !info.Disponible || v.Info().Version != "1.0.0" {
		t.Fatalf("%+v %v", info, err)
	}
	// Erreur réseau : le dernier résultat connu est gardé, avec l'erreur.
	srv.Close()
	info, err = v.Verifier(context.Background())
	if err == nil || info.Version != "1.0.0" || info.Erreur == "" {
		t.Errorf("%+v %v", info, err)
	}
}
