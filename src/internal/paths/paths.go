// Package paths centralise les emplacements d'Aria Vaultwarden.
//
// Tout vit dans %LOCALAPPDATA%\AriaVaultwarden, propre au compte Windows :
// Aria tourne avec les droits de l'utilisateur, jamais en administrateur.
package paths

import (
	"os"
	"path/filepath"
)

// Paths regroupe les dossiers et fichiers utilisés par Aria.
type Paths struct {
	Base        string // %LOCALAPPDATA%\AriaVaultwarden
	Moteur      string // vaultwarden.exe et web-vault, réextraits à chaque démarrage
	Donnees     string // le coffre : base SQLite, clés RSA, pièces jointes
	Certificats string // certificat Let's Encrypt et compte ACME
	Config      string // réglages d'Aria (jeton DuckDNS chiffré)
}

// Default renvoie les emplacements standard.
func Default() (Paths, error) {
	// Sous Windows, UserCacheDir correspond à %LOCALAPPDATA%.
	root, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, err
	}
	return In(filepath.Join(root, "AriaVaultwarden")), nil
}

// SauvegardesParDefaut est le dossier proposé pour les sauvegardes :
// Documents\Aria Vault\Sauvegardes (Documents suit une redirection OneDrive).
func SauvegardesParDefaut() string {
	return filepath.Join(documents(), "Aria Vault", "Sauvegardes")
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
