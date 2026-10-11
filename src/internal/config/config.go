// Package config lit et enregistre les réglages d'Aria Vaultwarden.
// Le jeton DuckDNS est chiffré avec DPAPI : seul le compte Windows qui l'a
// enregistré peut le relire.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config est le contenu de config.json.
type Config struct {
	Domaine      string `json:"domaine"`
	JetonChiffre []byte `json:"jetonDuckDNS,omitempty"` // DPAPI, encodé en base64 dans le JSON
	Email        string `json:"email,omitempty"`        // facultatif : avertissements Let's Encrypt
}

// Load lit config.json. Un fichier absent donne une configuration vide.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config.json illisible : %w", err)
	}
	return &c, nil
}

// Save enregistre config.json (écriture atomique).
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SetJeton chiffre et mémorise le jeton DuckDNS.
func (c *Config) SetJeton(jeton string) error {
	jeton = strings.TrimSpace(jeton)
	if !jetonRe.MatchString(jeton) {
		return errors.New("le jeton DuckDNS doit ressembler à xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	}
	enc, err := protect([]byte(jeton))
	if err != nil {
		return fmt.Errorf("chiffrement du jeton : %w", err)
	}
	c.JetonChiffre = enc
	return nil
}

// Jeton déchiffre le jeton DuckDNS.
func (c *Config) Jeton() (string, error) {
	if len(c.JetonChiffre) == 0 {
		return "", errors.New("aucun jeton DuckDNS enregistré")
	}
	plain, err := unprotect(c.JetonChiffre)
	if err != nil {
		return "", fmt.Errorf("jeton DuckDNS indéchiffrable (enregistré par un autre compte Windows ?) : %w", err)
	}
	return string(plain), nil
}

var (
	jetonRe   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	domaineRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.duckdns\.org$`)
	labelRe   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// NormaliserDomaine accepte « https://vault.exemple.duckdns.org/ » comme
// « vault.exemple.duckdns.org » et vérifie qu'il s'agit d'un sous-domaine DuckDNS.
func NormaliserDomaine(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	// DuckDNS gère aussi les sous-sous-domaines (vault.exemple.duckdns.org) :
	// seul le dernier niveau avant duckdns.org est vérifié par l'expression.
	parts := strings.Split(s, ".")
	if len(parts) < 3 || !domaineRe.MatchString(strings.Join(parts[len(parts)-3:], ".")) {
		return "", fmt.Errorf("%q n'est pas un sous-domaine DuckDNS (attendu : exemple.duckdns.org)", s)
	}
	for _, p := range parts[:len(parts)-3] {
		if !labelRe.MatchString(p) {
			return "", fmt.Errorf("%q contient un nom invalide", s)
		}
	}
	return s, nil
}

// Valider vérifie que tout ce qu'il faut pour démarrer est présent.
func (c *Config) Valider() error {
	if _, err := NormaliserDomaine(c.Domaine); err != nil {
		return err
	}
	_, err := c.Jeton()
	return err
}
