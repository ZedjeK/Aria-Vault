package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// Fichier de réglages exportable : domaine et e-mail en clair, jeton DuckDNS
// chiffré par une phrase de passe (scrypt + AES-256-GCM). À l'import, le jeton
// est rechiffré avec DPAPI pour le compte Windows qui l'importe.

const (
	formatExport  = "aria-vault-reglages"
	versionExport = 1
	// Paramètres scrypt recommandés pour un usage interactif (~0,3 s, 128 Mo).
	scryptN, scryptR, scryptP = 1 << 17, 8, 1
	PhraseMin                 = 8
)

// ErrPhrase signale une phrase de passe incorrecte (ou un fichier altéré).
var ErrPhrase = errors.New("phrase de passe incorrecte")

type exportFile struct {
	Format  string        `json:"format"`
	Version int           `json:"version"`
	Domaine string        `json:"domaine"`
	Email   string        `json:"email,omitempty"`
	Jeton   *jetonChiffre `json:"jetonDuckDNS"`
}

type jetonChiffre struct {
	KDF     string `json:"kdf"`
	N       int    `json:"n"`
	R       int    `json:"r"`
	P       int    `json:"p"`
	Sel     []byte `json:"sel"`
	Nonce   []byte `json:"nonce"`
	Chiffre []byte `json:"chiffre"`
}

// Exporter produit le fichier de réglages. La phrase de passe protège le jeton.
func (c *Config) Exporter(phrase string) ([]byte, error) {
	if len([]rune(phrase)) < PhraseMin {
		return nil, fmt.Errorf("la phrase de passe doit faire au moins %d caractères", PhraseMin)
	}
	if err := c.Valider(); err != nil {
		return nil, err
	}
	jeton, err := c.Jeton()
	if err != nil {
		return nil, err
	}
	sel := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := rand.Read(sel); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	gcm, err := aead(phrase, sel, scryptN, scryptR, scryptP)
	if err != nil {
		return nil, err
	}
	// Le domaine est lié au chiffré : on ne peut pas le changer sans casser le déchiffrement.
	chiffre := gcm.Seal(nil, nonce, []byte(jeton), []byte(c.Domaine))
	return json.MarshalIndent(exportFile{
		Format:  formatExport,
		Version: versionExport,
		Domaine: c.Domaine,
		Email:   c.Email,
		Jeton:   &jetonChiffre{KDF: "scrypt", N: scryptN, R: scryptR, P: scryptP, Sel: sel, Nonce: nonce, Chiffre: chiffre},
	}, "", "  ")
}

// Importe est le contenu d'un fichier de réglages déchiffré.
type Importe struct {
	Domaine string
	Email   string
	Jeton   string // vide si aucune phrase n'a été donnée
}

// LireExport lit un fichier de réglages. Avec une phrase vide, seuls le
// domaine et l'e-mail sont rendus (le jeton sera à ressaisir).
func LireExport(data []byte, phrase string) (Importe, error) {
	var f exportFile
	if err := json.Unmarshal(data, &f); err != nil || f.Format != formatExport {
		return Importe{}, errors.New("ce n'est pas un fichier de réglages Aria Vault")
	}
	if f.Version != versionExport {
		return Importe{}, fmt.Errorf("version de fichier %d non prise en charge", f.Version)
	}
	d, err := NormaliserDomaine(f.Domaine)
	if err != nil {
		return Importe{}, err
	}
	out := Importe{Domaine: d, Email: strings.TrimSpace(f.Email)}
	if phrase == "" || f.Jeton == nil {
		return out, nil
	}
	j := f.Jeton
	// Bornes raisonnables : un fichier piégé ne doit pas faire exploser la mémoire.
	if j.KDF != "scrypt" || j.N < 1<<14 || j.N > 1<<20 || j.R < 1 || j.R > 16 || j.P < 1 || j.P > 4 || len(j.Sel) < 16 || len(j.Nonce) != 12 {
		return Importe{}, errors.New("paramètres de chiffrement invalides dans le fichier")
	}
	gcm, err := aead(phrase, j.Sel, j.N, j.R, j.P)
	if err != nil {
		return Importe{}, err
	}
	clair, err := gcm.Open(nil, j.Nonce, j.Chiffre, []byte(f.Domaine))
	if err != nil {
		return Importe{}, ErrPhrase
	}
	out.Jeton = string(clair)
	return out, nil
}

func aead(phrase string, sel []byte, n, r, p int) (cipher.AEAD, error) {
	cle, err := scrypt.Key([]byte(phrase), sel, n, r, p, 32)
	if err != nil {
		return nil, err
	}
	bloc, err := aes.NewCipher(cle)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(bloc)
}
