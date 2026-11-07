package vaultwarden

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// clesAdmin sont les réglages de data\config.json qui ouvriraient le panneau
// /admin : admin_token le protège par un jeton, disable_admin_token l'ouvre
// sans aucune authentification. Aria n'en fixe aucun, le panneau reste fermé.
var clesAdmin = []string{"admin_token", "disable_admin_token"}

// FermerAdmin retire de donnees\config.json les réglages qui rouvriraient le
// panneau /admin (un ancien coffre importé, une sauvegarde d'avant Aria, un
// fichier copié à la main). Les autres réglages sont gardés tels quels.
// Renvoie les clés retirées (aucune si le fichier est absent ou déjà sain).
// Un config.json illisible est une erreur : Vaultwarden ne doit pas démarrer
// sur un fichier qu'Aria n'a pas pu vérifier.
func FermerAdmin(donnees string) ([]string, error) {
	chemin := filepath.Join(donnees, "config.json")
	data, err := os.ReadFile(chemin)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var reglages map[string]json.RawMessage
	if err := json.Unmarshal(data, &reglages); err != nil {
		return nil, fmt.Errorf("data\\config.json illisible, impossible de vérifier que le panneau /admin reste fermé (corrige ou supprime ce fichier) : %w", err)
	}
	var retirees []string
	for k := range reglages {
		for _, a := range clesAdmin {
			// Toutes les casses : rien ne doit pouvoir passer entre les mailles.
			if strings.EqualFold(k, a) {
				retirees = append(retirees, k)
				delete(reglages, k)
			}
		}
	}
	if len(retirees) == 0 {
		return nil, nil
	}
	sort.Strings(retirees)
	propre, err := json.MarshalIndent(reglages, "", "  ")
	if err != nil {
		return nil, err
	}
	tmp := chemin + ".tmp"
	if err := os.WriteFile(tmp, propre, 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, chemin); err != nil {
		os.Remove(tmp)
		return nil, err
	}
	return retirees, nil
}

// MessageAdminFerme explique à l'utilisateur ce que FermerAdmin a retiré.
func MessageAdminFerme(retirees []string) string {
	return fmt.Sprintf("Panneau /admin de Vaultwarden laissé fermé : %s retiré de data\\config.json (Aria Vault ne l'expose jamais à travers le HTTPS)", strings.Join(retirees, " et "))
}
