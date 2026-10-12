// Package demarrage garde l'entrée de lancement à l'ouverture de session
// (HKCU\...\Run) alignée sur l'exe en cours : après un renommage (Aria
// Vaultwarden → Aria Vault) ou une relève par une version placée ailleurs,
// Windows lance toujours la bonne version.
package demarrage

import "strings"

const (
	// Noms des valeurs du registre, tels que Wails les dérive du nom de l'appli.
	Ancien = "aria-vaultwarden" // v0.3 et avant
	Actuel = "aria-vault"
)

// Commande construit la ligne de commande enregistrée dans le registre.
func Commande(exe string, args ...string) string {
	out := `"` + exe + `"`
	for _, a := range args {
		out += " " + a
	}
	return out
}

// Exe extrait le chemin de l'exe d'une ligne de commande (guillemets ou non).
func Exe(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		if fin := strings.IndexByte(cmd[1:], '"'); fin >= 0 {
			return cmd[1 : 1+fin]
		}
		return cmd[1:]
	}
	if i := strings.IndexAny(cmd, " \t"); i >= 0 {
		return cmd[:i]
	}
	return cmd
}
