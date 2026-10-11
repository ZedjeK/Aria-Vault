package sauvegarde

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// fauxCoffre crée un dossier data réaliste.
func fauxCoffre(t *testing.T, dir, marque string) {
	t.Helper()
	ecrire := func(rel, contenu string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ecrire("db.sqlite3", "base "+marque)
	ecrire("db.sqlite3-wal", "wal "+marque)
	ecrire("db.sqlite3-shm", "shm")
	ecrire("rsa_key.pem", "cle "+marque)
	ecrire("config.json", "{}")
	ecrire("attachments/abc/def.bin", "piece jointe "+marque)
	ecrire("sends/s1/f.txt", "send")
	ecrire("icon_cache/site.png", "icone")
	ecrire("tmp/x", "temporaire")
	ecrire("vaultwarden.log", "journal")
}

func entrees(t *testing.T, chemin string) []string {
	t.Helper()
	zr, err := zip.OpenReader(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var noms []string
	for _, f := range zr.File {
		noms = append(noms, f.Name)
	}
	slices.Sort(noms)
	return noms
}

func TestCreerNeGardeQueLEssentiel(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	fauxCoffre(t, data, "A")
	dest := filepath.Join(t.TempDir(), "Sauvegardes")
	now := time.Date(2026, 10, 12, 21, 30, 0, 0, time.Local)

	info, err := Creer(data, dest, now)
	if err != nil {
		t.Fatal(err)
	}
	if info.Nom != "aria-vault-2026-10-12_213000.zip" {
		t.Errorf("nom = %q", info.Nom)
	}
	want := []string{"aria-vault.json", "attachments/abc/def.bin", "config.json", "db.sqlite3", "db.sqlite3-shm", "db.sqlite3-wal", "rsa_key.pem", "sends/s1/f.txt"}
	if got := entrees(t, info.Chemin); !slices.Equal(got, want) {
		t.Errorf("contenu =\n%v\nattendu\n%v", got, want)
	}
	if _, err := os.Stat(info.Chemin + ".tmp"); err == nil {
		t.Error("le fichier temporaire doit disparaître")
	}
}

func TestCreerSansCoffre(t *testing.T) {
	if _, err := Creer(t.TempDir(), t.TempDir(), time.Now()); err != ErrPasDeCoffre {
		t.Errorf("err = %v", err)
	}
}

func TestListerEtPurger(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	fauxCoffre(t, data, "A")
	dest := t.TempDir()
	debut := time.Date(2026, 10, 1, 3, 0, 0, 0, time.Local)
	for i := 0; i < 12; i++ {
		if _, err := Creer(data, dest, debut.Add(time.Duration(i)*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	// Un fichier étranger ne doit jamais être touché.
	os.WriteFile(filepath.Join(dest, "photo.zip"), []byte("x"), 0o600)

	supprimees, err := Purger(dest, Garder)
	if err != nil {
		t.Fatal(err)
	}
	if len(supprimees) != 2 || supprimees[1] != "aria-vault-2026-10-01_030000.zip" {
		t.Errorf("supprimées = %v", supprimees)
	}
	liste, _ := Lister(dest)
	if len(liste) != Garder || liste[0].Nom != "aria-vault-2026-10-12_030000.zip" {
		t.Errorf("liste = %d, première %q", len(liste), liste[0].Nom)
	}
	if _, err := os.Stat(filepath.Join(dest, "photo.zip")); err != nil {
		t.Error("photo.zip a été supprimé")
	}
}

func TestRestaurerMetLAncienDeCote(t *testing.T) {
	racine := t.TempDir()
	data := filepath.Join(racine, "data")
	fauxCoffre(t, data, "ANCIEN")
	dest := filepath.Join(racine, "svg")
	info, err := Creer(data, dest, time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}

	// Le coffre change après la sauvegarde…
	os.RemoveAll(data)
	fauxCoffre(t, data, "NOUVEAU")

	// …puis on restaure.
	now := time.Date(2026, 10, 12, 10, 0, 0, 0, time.Local)
	ancien, err := Restaurer(info.Chemin, data, now)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(data, "db.sqlite3")); string(b) != "base ANCIEN" {
		t.Errorf("base restaurée = %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(data, "attachments", "abc", "def.bin")); string(b) != "piece jointe ANCIEN" {
		t.Errorf("pièce jointe restaurée = %q", b)
	}
	if _, err := os.Stat(filepath.Join(data, "aria-vault.json")); err == nil {
		t.Error("le manifeste ne doit pas atterrir dans le coffre")
	}
	if ancien != data+".avant-restauration-2026-10-12_100000" {
		t.Errorf("ancien = %q", ancien)
	}
	if b, _ := os.ReadFile(filepath.Join(ancien, "db.sqlite3")); string(b) != "base NOUVEAU" {
		t.Errorf("le coffre remplacé doit être conservé intact, lu %q", b)
	}
}

func TestVerifierRefuse(t *testing.T) {
	dir := t.TempDir()
	creerZip := func(nom string, fichiers ...string) string {
		p := filepath.Join(dir, nom)
		f, _ := os.Create(p)
		zw := zip.NewWriter(f)
		for _, n := range fichiers {
			w, _ := zw.Create(n)
			w.Write([]byte("x"))
		}
		zw.Close()
		f.Close()
		return p
	}
	if err := Verifier(creerZip("sansbase.zip", "rsa_key.pem")); err == nil {
		t.Error("archive sans db.sqlite3 acceptée")
	}
	if err := Verifier(creerZip("evasion.zip", "db.sqlite3", "../../evasion.txt")); err == nil {
		t.Error("chemin d'évasion accepté")
	}
	os.WriteFile(filepath.Join(dir, "pasunzip.zip"), []byte("bonjour"), 0o600)
	if err := Verifier(filepath.Join(dir, "pasunzip.zip")); err == nil {
		t.Error("fichier non zip accepté")
	}
}

func TestImporter(t *testing.T) {
	racine := t.TempDir()
	ancienVW := filepath.Join(racine, "Vaultwarden", "data")
	fauxCoffre(t, ancienVW, "VW")
	data := filepath.Join(racine, "AriaVault", "data")
	fauxCoffre(t, data, "ARIA")

	mis, err := Importer(ancienVW, data, time.Date(2026, 10, 12, 11, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(data, "db.sqlite3")); string(b) != "base VW" {
		t.Errorf("base importée = %q", b)
	}
	if _, err := os.Stat(filepath.Join(data, "icon_cache")); err == nil {
		t.Error("le cache d'icônes ne doit pas être importé")
	}
	if b, _ := os.ReadFile(filepath.Join(mis, "db.sqlite3")); string(b) != "base ARIA" {
		t.Errorf("ancien coffre conservé = %q", b)
	}
	if _, err := Importer(data, data, time.Now()); err == nil {
		t.Error("importer le coffre sur lui-même doit être refusé")
	}
	if _, err := Importer(t.TempDir(), data, time.Now()); err != ErrPasDeCoffre {
		t.Errorf("dossier vide : %v", err)
	}
}
