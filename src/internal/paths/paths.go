// Package paths centralise les emplacements d'Aria Vault.
//
// Tout vit dans %LOCALAPPDATA%\AriaVault, propre au compte Windows :
// Aria tourne avec les droits de l'utilisateur, jamais en administrateur.
// Jusqu'à la v0.3 (sous le nom Aria Vaultwarden), le dossier s'appelait
// AriaVaultwarden : il est renommé automatiquement au premier lancement.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Paths regroupe les dossiers et fichiers utilisés par Aria.
type Paths struct {
	Base        string // %LOCALAPPDATA%\AriaVault
	Moteur      string // vaultwarden.exe et web-vault, réextraits à chaque démarrage
	Donnees     string // le coffre : base SQLite, clés RSA, pièces jointes
	Certificats string // certificat Let's Encrypt et compte ACME
	Config      string // réglages d'Aria (jeton DuckDNS chiffré)
}

const (
	Dossier       = "AriaVault"
	AncienDossier = "AriaVaultwarden" // v0.3 et avant
)

// Racine renvoie le dossier parent (%LOCALAPPDATA% sous Windows).
func Racine() (string, error) {
	// Sous Windows, UserCacheDir correspond à %LOCALAPPDATA%.
	return os.UserCacheDir()
}

// Migrer renvoie les emplacements d'Aria, en renommant au besoin l'ancien
// dossier AriaVaultwarden en AriaVault. Si le renommage échoue (un fichier
// encore ouvert), Aria continue sur l'ancien dossier et réessaiera au
// prochain lancement. msg décrit la migration effectuée, s'il y en a eu une.
func Migrer(racine string) (p Paths, msg string, err error) {
	return migrer(racine, 20, 500*time.Millisecond)
}

func migrer(racine string, essais int, pause time.Duration) (Paths, string, error) {
	nouveau := filepath.Join(racine, Dossier)
	ancien := filepath.Join(racine, AncienDossier)
	if existe(nouveau) || !existe(ancien) {
		return In(nouveau), "", nil
	}
	var err error
	for i := 0; i < essais; i++ {
		if err = os.Rename(ancien, nouveau); err == nil {
			return In(nouveau), fmt.Sprintf("Dossier d'Aria renommé : %s → %s", ancien, nouveau), nil
		}
		time.Sleep(pause) // l'ancienne version libère encore ses fichiers
	}
	return In(ancien), "", fmt.Errorf("renommage de %s impossible, Aria continue dessus pour cette fois : %w", ancien, err)
}

func existe(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Documents renvoie le dossier Documents de l'utilisateur (redirection
// OneDrive comprise) : point de départ des boîtes de dialogue.
func Documents() string { return documents() }

// SurOneDrive indique si p est dans un dossier synchronisé par OneDrive.
func SurOneDrive(p string) bool {
	for _, v := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		racine := os.Getenv(v)
		if racine == "" {
			continue
		}
		if rel, err := filepath.Rel(racine, p); err == nil && filepath.IsLocal(rel) || filepath.Clean(p) == filepath.Clean(racine) {
			return true
		}
	}
	return false
}

// MemeDisque indique si a et b sont sur le même lecteur (C:, D:…).
func MemeDisque(a, b string) bool {
	va, vb := filepath.VolumeName(a), filepath.VolumeName(b)
	return va != "" && strings.EqualFold(va, vb)
}

// In construit les emplacements à partir d'un dossier de base.
func In(base string) Paths {
	return Paths{
		Base:        base,
		Moteur:      filepath.Join(base, "moteur"),
		Donnees:     filepath.Join(base, "data"),
		Certificats: filepath.Join(base, "certificats"),
		Config:      filepath.Join(base, "config.json"),
	}
}

// Existant renvoie p, ou son plus proche parent qui existe ("" si aucun).
// Utile pour ouvrir une boîte de dialogue Windows, qui refuse un dossier absent.
func Existant(p string) string {
	for {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return ""
		}
		p = parent
	}
}
