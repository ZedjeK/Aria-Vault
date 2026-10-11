// Package sauvegarde crée, vérifie, liste et restaure les sauvegardes du
// coffre Vaultwarden (le dossier data).
//
// Une sauvegarde est un zip horodaté qui ne garde que l'essentiel : la base
// SQLite (avec ses fichiers -wal et -shm), les clés du serveur, les pièces
// jointes, les Sends et config.json. Le cache d'icônes, les fichiers
// temporaires et les journaux sont laissés de côté.
//
// Une restauration ne supprime jamais rien : le coffre en place est renommé
// en data.avant-restauration-<date> avant que la sauvegarde prenne sa place.
package sauvegarde

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	Prefixe   = "aria-vault-"
	Extension = ".zip"
	Garder    = 10 // sauvegardes conservées par la rotation

	formatDate = "2006-01-02_150405"
	manifeste  = "aria-vault.json"
	base       = "db.sqlite3"
)

// Ce qui est sauvegardé, relativement au dossier data.
var (
	fichiers = []string{"db.sqlite3", "db.sqlite3-wal", "db.sqlite3-shm", "config.json"}
	motifs   = []string{"rsa_key*"} // rsa_key.pem, et les anciennes rsa_key.der / .pub.*
	dossiers = []string{"attachments", "sends"}
)

// Info décrit une sauvegarde.
type Info struct {
	Nom    string    `json:"nom"`
	Chemin string    `json:"chemin"`
	Date   time.Time `json:"date"`
	Taille int64     `json:"taille"`
}

type manifest struct {
	Format   string    `json:"format"`
	Version  int       `json:"version"`
	Date     time.Time `json:"date"`
	Fichiers int       `json:"fichiers"`
}

// ErrPasDeCoffre signale un dossier sans base Vaultwarden.
var ErrPasDeCoffre = errors.New("aucun coffre ici : db.sqlite3 introuvable")

// Creer sauvegarde le dossier donnees dans dest. Vaultwarden doit être arrêté
// pour que la base soit copiée dans un état cohérent.
func Creer(donnees, dest string, now time.Time) (Info, error) {
	if _, err := os.Stat(filepath.Join(donnees, base)); err != nil {
		return Info{}, ErrPasDeCoffre
	}
	elements, err := aSauvegarder(donnees)
	if err != nil {
		return Info{}, err
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return Info{}, err
	}

	nom := Prefixe + now.Format(formatDate) + Extension
	final := filepath.Join(dest, nom)
	tmp := final + ".tmp"
	if err := ecrireZip(tmp, donnees, elements, now); err != nil {
		os.Remove(tmp)
		return Info{}, err
	}
	// Relue intégralement (CRC de chaque fichier) avant d'être considérée valide.
	if err := Verifier(tmp); err != nil {
		os.Remove(tmp)
		return Info{}, fmt.Errorf("sauvegarde illisible après écriture : %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return Info{}, err
	}
	st, err := os.Stat(final)
	if err != nil {
		return Info{}, err
	}
	return Info{Nom: nom, Chemin: final, Date: now, Taille: st.Size()}, nil
}

// aSauvegarder liste les fichiers à copier (chemins relatifs, séparateur /).
func aSauvegarder(donnees string) ([]string, error) {
	var out []string
	for _, f := range fichiers {
		if st, err := os.Lstat(filepath.Join(donnees, f)); err == nil && st.Mode().IsRegular() {
			out = append(out, f)
		}
	}
	for _, m := range motifs {
		trouves, _ := filepath.Glob(filepath.Join(donnees, m))
		for _, t := range trouves {
			if st, err := os.Lstat(t); err == nil && st.Mode().IsRegular() {
				out = append(out, filepath.Base(t))
			}
		}
	}
	for _, d := range dossiers {
		racine := filepath.Join(donnees, d)
		if _, err := os.Lstat(racine); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		err := filepath.WalkDir(racine, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.Type().IsRegular() {
				rel, err := filepath.Rel(donnees, p)
				if err != nil {
					return err
				}
				out = append(out, filepath.ToSlash(rel))
			}
			return nil // liens et fichiers spéciaux ignorés
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func ecrireZip(chemin, donnees string, elements []string, now time.Time) error {
	f, err := os.OpenFile(chemin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	for _, rel := range elements {
		if err := ajouter(zw, donnees, rel); err != nil {
			zw.Close()
			f.Close()
			return fmt.Errorf("%s : %w", rel, err)
		}
	}
	m, _ := json.MarshalIndent(manifest{Format: "aria-vault-sauvegarde", Version: 1, Date: now, Fichiers: len(elements)}, "", "  ")
	w, err := zw.Create(manifeste)
	if err == nil {
		_, err = w.Write(m)
	}
	if err != nil {
		zw.Close()
		f.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func ajouter(zw *zip.Writer, donnees, rel string) error {
	src, err := os.Open(filepath.Join(donnees, filepath.FromSlash(rel)))
	if err != nil {
		return err
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil {
		return err
	}
	h, err := zip.FileInfoHeader(st)
	if err != nil {
		return err
	}
	h.Name = rel
	h.Method = zip.Deflate
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

// Verifier relit toute l'archive : chemins sûrs, base présente, CRC corrects.
func Verifier(chemin string) error {
	zr, err := zip.OpenReader(chemin)
	if err != nil {
		return fmt.Errorf("archive illisible : %w", err)
	}
	defer zr.Close()
	aBase := false
	for _, f := range zr.File {
		if !cheminSur(f.Name) {
			return fmt.Errorf("chemin refusé dans l'archive : %s", f.Name)
		}
		if f.Name == base {
			aBase = true
		}
		if f.FileInfo().IsDir() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(io.Discard, r) // le lecteur zip vérifie le CRC en fin de flux
		r.Close()
		if err != nil {
			return fmt.Errorf("%s : %w", f.Name, err)
		}
	}
	if !aBase {
		return errors.New("ce n'est pas une sauvegarde de coffre : db.sqlite3 absent")
	}
	return nil
}

func cheminSur(nom string) bool {
	return nom != "" && !strings.Contains(nom, `\`) && filepath.IsLocal(filepath.FromSlash(nom))
}

// Lister renvoie les sauvegardes de dest, de la plus récente à la plus ancienne.
func Lister(dest string) ([]Info, error) {
	entrees, err := os.ReadDir(dest)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, e := range entrees {
		date, ok := DateDe(e.Name())
		if !ok || !e.Type().IsRegular() {
			continue
		}
		st, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Info{Nom: e.Name(), Chemin: filepath.Join(dest, e.Name()), Date: date, Taille: st.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.After(out[j].Date) })
	return out, nil
}

// DateDe lit la date dans le nom d'une sauvegarde (aria-vault-2026-10-12_213000.zip).
func DateDe(nom string) (time.Time, bool) {
	if !strings.HasPrefix(nom, Prefixe) || !strings.HasSuffix(nom, Extension) || filepath.Base(nom) != nom {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(formatDate, strings.TrimSuffix(strings.TrimPrefix(nom, Prefixe), Extension), time.Local)
	return t, err == nil
}

// Purger ne garde que les `garder` sauvegardes les plus récentes de dest.
// Seuls les fichiers nommés par Aria sont concernés.
func Purger(dest string, garder int) ([]string, error) {
	liste, err := Lister(dest)
	if err != nil || len(liste) <= garder {
		return nil, err
	}
	var supprimees []string
	for _, s := range liste[garder:] {
		if err := os.Remove(s.Chemin); err != nil {
			return supprimees, err
		}
		supprimees = append(supprimees, s.Nom)
	}
	return supprimees, nil
}

// Restaurer remplace le coffre par le contenu de l'archive. Vaultwarden doit
// être arrêté. Renvoie le dossier où l'ancien coffre a été mis de côté ("" s'il
// n'y en avait pas).
func Restaurer(archive, donnees string, now time.Time) (string, error) {
	if err := Verifier(archive); err != nil {
		return "", err
	}
	tmp := donnees + ".restauration"
	if err := os.RemoveAll(tmp); err != nil {
		return "", err
	}
	if err := extraire(archive, tmp); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	return remplacer(tmp, donnees, now)
}

// Importer remplace le coffre par un autre dossier data (par exemple l'ancien
// C:\Vaultwarden\data). Seuls les éléments utiles sont copiés.
func Importer(source, donnees string, now time.Time) (string, error) {
	src, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	dst, err := filepath.Abs(donnees)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dst)) {
		return "", errors.New("c'est déjà le dossier du coffre actuel")
	}
	if _, err := os.Stat(filepath.Join(src, base)); err != nil {
		return "", ErrPasDeCoffre
	}
	elements, err := aSauvegarder(src)
	if err != nil {
		return "", err
	}
	tmp := donnees + ".restauration"
	if err := os.RemoveAll(tmp); err != nil {
		return "", err
	}
	for _, rel := range elements {
		if err := copier(filepath.Join(src, filepath.FromSlash(rel)), filepath.Join(tmp, filepath.FromSlash(rel))); err != nil {
			os.RemoveAll(tmp)
			return "", fmt.Errorf("%s : %w", rel, err)
		}
	}
	return remplacer(tmp, donnees, now)
}

// remplacer met le coffre actuel de côté puis installe tmp à sa place.
func remplacer(tmp, donnees string, now time.Time) (string, error) {
	ancien := ""
	if _, err := os.Stat(donnees); err == nil {
		ancien = donnees + ".avant-restauration-" + now.Format(formatDate)
		if err := os.Rename(donnees, ancien); err != nil {
			os.RemoveAll(tmp)
			return "", fmt.Errorf("impossible de mettre le coffre actuel de côté (un fichier est-il ouvert ?) : %w", err)
		}
	}
	if err := os.Rename(tmp, donnees); err != nil {
		if ancien != "" {
			_ = os.Rename(ancien, donnees) // on remet l'ancien coffre en place
		}
		os.RemoveAll(tmp)
		return "", err
	}
	return ancien, nil
}

func extraire(archive, dst string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name == manifeste || f.FileInfo().IsDir() {
			continue
		}
		if !cheminSur(f.Name) {
			return fmt.Errorf("chemin refusé dans l'archive : %s", f.Name)
		}
		cible := filepath.Join(dst, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(cible), 0o700); err != nil {
			return err
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(cible, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			r.Close()
			return err
		}
		_, err = io.Copy(w, r)
		r.Close()
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
		_ = os.Chtimes(cible, f.Modified, f.Modified)
	}
	return nil
}

func copier(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chtimes(dst, st.ModTime(), st.ModTime())
}
