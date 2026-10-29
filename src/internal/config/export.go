package config

import (
	"encoding/json"
	"errors"
	"strings"

	"aria-vault/internal/chiffre"
	"aria-vault/internal/dnsprov"
)

// Fichier de réglages exportable (.ariavault), entièrement chiffré par une
// phrase de passe (scrypt + AES-256-GCM, voir le paquet chiffre) : rien n'y
// est lisible sans elle, ni le domaine, ni l'e-mail, ni les identifiants DNS.
// À l'import, les identifiants sont rechiffrés avec DPAPI pour le compte
// Windows qui l'importe.

const (
	ExtensionExport = ".ariavault"
	PhraseMin       = chiffre.PhraseMin
)

var magie = []byte("ARIAVLT2")

// ErrPhrase signale une phrase de passe incorrecte (ou un fichier altéré).
var ErrPhrase = chiffre.ErrPhrase

// Importe est le contenu d'un fichier de réglages déchiffré.
type Importe struct {
	Domaine      string            `json:"domaine"`
	Email        string            `json:"email,omitempty"`
	Fournisseur  string            `json:"fournisseur,omitempty"`  // absent avant la v0.6 : DuckDNS
	Identifiants map[string]string `json:"identifiants,omitempty"` // v0.6
	Jeton        string            `json:"jeton,omitempty"`        // DuckDNS, lu par les versions précédentes
}

// Exporter produit le fichier de réglages chiffré par la phrase de passe.
func (c *Config) Exporter(phrase string) ([]byte, error) {
	if err := chiffre.VerifierPhrase(phrase); err != nil {
		return nil, err
	}
	if err := c.Valider(); err != nil {
		return nil, err
	}
	ids, err := c.Identifiants()
	if err != nil {
		return nil, err
	}
	imp := Importe{Domaine: c.Domaine, Email: c.Email, Fournisseur: c.FournisseurDNS(), Identifiants: ids}
	if imp.Fournisseur == dnsprov.DuckDNS {
		imp.Jeton = ids["jeton"]
	}
	clair, err := json.Marshal(imp)
	if err != nil {
		return nil, err
	}
	return chiffre.Sceller(phrase, magie, clair)
}

// LireExport déchiffre un fichier de réglages. La phrase est obligatoire.
func LireExport(data []byte, phrase string) (Importe, error) {
	if phrase == "" {
		return Importe{}, errors.New("la phrase de passe du fichier est obligatoire")
	}
	clair, err := chiffre.Ouvrir(phrase, magie, data)
	if errors.Is(err, chiffre.ErrFormat) {
		return Importe{}, errors.New("ce n'est pas un fichier de réglages Aria Vault (.ariavault)")
	}
	if err != nil {
		return Importe{}, err
	}

	var imp Importe
	if err := json.Unmarshal(clair, &imp); err != nil {
		return Importe{}, errors.New("contenu du fichier illisible")
	}
	if imp.Fournisseur == "" {
		imp.Fournisseur = dnsprov.DuckDNS
		imp.Identifiants = map[string]string{"jeton": imp.Jeton}
	}
	if imp.Domaine, err = dnsprov.NormaliserDomaine(imp.Fournisseur, imp.Domaine); err != nil {
		return Importe{}, err
	}
	imp.Email = strings.TrimSpace(imp.Email)
	propres, erreurs := dnsprov.Valider(imp.Fournisseur, imp.Identifiants)
	if len(erreurs) > 0 {
		return Importe{}, errors.New("identifiants DNS invalides dans le fichier")
	}
	imp.Identifiants = propres
	return imp, nil
}
