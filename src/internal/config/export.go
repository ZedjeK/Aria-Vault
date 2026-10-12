package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// Fichier de réglages exportable (.ariavault), entièrement chiffré par une
// phrase de passe : rien n'y est lisible sans elle, ni le domaine, ni
// l'e-mail, ni le jeton DuckDNS. À l'import, le jeton est rechiffré avec
// DPAPI pour le compte Windows qui l'importe.
//
// Format binaire :
//
//	"ARIAVLT2" | log2(N) | r | p | sel (16 o) | nonce (12 o) | chiffré AES-256-GCM
//
// L'en-tête (tout ce qui précède le chiffré) est authentifié avec le contenu :
// le moindre octet modifié fait échouer le déchiffrement.

const (
	ExtensionExport = ".ariavault"
	PhraseMin       = 8

	// Paramètres scrypt recommandés pour un usage interactif (~0,3 s, 128 Mo).
	scryptLogN, scryptR, scryptP = 17, 8, 1
)

var magie = []byte("ARIAVLT2")

const tailleEntete = 8 + 3 + 16 + 12

// ErrPhrase signale une phrase de passe incorrecte (ou un fichier altéré).
var ErrPhrase = errors.New("phrase de passe incorrecte (ou fichier modifié)")

// Importe est le contenu d'un fichier de réglages déchiffré.
type Importe struct {
	Domaine string `json:"domaine"`
	Email   string `json:"email,omitempty"`
	Jeton   string `json:"jeton"`
}

// Exporter produit le fichier de réglages chiffré par la phrase de passe.
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
	clair, err := json.Marshal(Importe{Domaine: c.Domaine, Email: c.Email, Jeton: jeton})
	if err != nil {
		return nil, err
	}

	entete := make([]byte, 0, tailleEntete)
	entete = append(entete, magie...)
	entete = append(entete, scryptLogN, scryptR, scryptP)
	alea := make([]byte, 16+12)
	if _, err := rand.Read(alea); err != nil {
		return nil, err
	}
	entete = append(entete, alea...)
	sel, nonce := entete[11:27], entete[27:39]

	gcm, err := aead(phrase, sel, 1<<scryptLogN, scryptR, scryptP)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(entete, nonce, clair, entete), nil
}

// LireExport déchiffre un fichier de réglages. La phrase est obligatoire.
func LireExport(data []byte, phrase string) (Importe, error) {
	if phrase == "" {
		return Importe{}, errors.New("la phrase de passe du fichier est obligatoire")
	}
	if len(data) <= tailleEntete || !bytes.Equal(data[:8], magie) {
		return Importe{}, errors.New("ce n'est pas un fichier de réglages Aria Vault (.ariavault)")
	}
	logN, r, p := int(data[8]), int(data[9]), int(data[10])
	// Bornes raisonnables : un fichier piégé ne doit pas faire exploser la mémoire.
	if logN < 14 || logN > 20 || r < 1 || r > 16 || p < 1 || p > 4 {
		return Importe{}, errors.New("paramètres de chiffrement invalides dans le fichier")
	}
	entete, chiffre := data[:tailleEntete], data[tailleEntete:]
	sel, nonce := entete[11:27], entete[27:39]

	gcm, err := aead(phrase, sel, 1<<logN, r, p)
	if err != nil {
		return Importe{}, err
	}
	clair, err := gcm.Open(nil, nonce, chiffre, entete)
	if err != nil {
		return Importe{}, ErrPhrase
	}

	var imp Importe
	if err := json.Unmarshal(clair, &imp); err != nil {
		return Importe{}, errors.New("contenu du fichier illisible")
	}
	if imp.Domaine, err = NormaliserDomaine(imp.Domaine); err != nil {
		return Importe{}, err
	}
	imp.Email = strings.TrimSpace(imp.Email)
	if !jetonRe.MatchString(imp.Jeton) {
		return Importe{}, errors.New("jeton DuckDNS invalide dans le fichier")
	}
	return imp, nil
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
