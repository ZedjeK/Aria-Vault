package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestJournalEtCertmagic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aria.log")
	j := New(path)
	j.Add(OK, "Coffre en ligne : %s", "https://vault.exemple.duckdns.org")

	log := zap.New(j.Core()).With(zap.String("identifier", "vault.exemple.duckdns.org"))
	log.Debug("ignoré")
	log.Info("obtaining certificate", zap.String("interne", "omis"))
	log.Error("could not get certificate", zap.Error(errors.New("rate limited")))

	e := j.Entries()
	if len(e) != 3 {
		t.Fatalf("%d entrées, attendu 3 : %+v", len(e), e)
	}
	if e[1].Texte != "certificat : obtaining certificate · identifier=vault.exemple.duckdns.org" {
		t.Errorf("entrée certmagic = %q", e[1].Texte)
	}
	if e[2].Niveau != Erreur || !strings.Contains(e[2].Texte, "error=rate limited") {
		t.Errorf("erreur certmagic = %+v", e[2])
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "\n") != 3 {
		t.Errorf("aria.log :\n%s", data)
	}
}

func TestJournalBorne(t *testing.T) {
	j := New("")
	for i := 0; i < maxEntries+50; i++ {
		j.Add(Info, "ligne %d", i)
	}
	e := j.Entries()
	if len(e) != maxEntries || e[0].Texte != "ligne 50" {
		t.Errorf("%d entrées, première %q", len(e), e[0].Texte)
	}
}
