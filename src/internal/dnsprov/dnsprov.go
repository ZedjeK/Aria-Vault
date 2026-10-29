// Package dnsprov décrit les fournisseurs DNS gérés par Aria Vault et
// fabrique leur client libdns : DuckDNS, Cloudflare, OVH et deSEC.
//
// Un même client sert à deux choses : publier la preuve DNS-01 pour le
// certificat Let's Encrypt, et faire pointer le domaine vers l'IP du PC.
package dnsprov

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/libdns/cloudflare"
	"github.com/libdns/desec"
	"github.com/libdns/duckdns"
	"github.com/libdns/libdns"
	"github.com/libdns/ovh"
)

// Identifiants des fournisseurs (enregistrés dans config.json).
const (
	DuckDNS    = "duckdns"
	Cloudflare = "cloudflare"
	OVH        = "ovh"
	DeSEC      = "desec"
)

// Champ est un identifiant à saisir pour un fournisseur.
type Champ struct {
	Cle         string   `json:"cle"`
	Libelle     string   `json:"libelle"`
	Secret      bool     `json:"secret"`
	Placeholder string   `json:"placeholder"`
	Choix       []string `json:"choix,omitempty"` // liste fermée (point d'accès OVH)
	Defaut      string   `json:"defaut,omitempty"`
}

// Etape d'un tutoriel. Lien est facultatif.
type Etape struct {
	Texte string `json:"texte"`
	Lien  string `json:"lien,omitempty"`
}

// Fournisseur décrit un fournisseur et son tutoriel, affiché par l'interface.
type Fournisseur struct {
	ID       string  `json:"id"`
	Nom      string  `json:"nom"`
	Resume   string  `json:"resume"`
	Exemple  string  `json:"exemple"` // domaine d'exemple
	Suffixe  string  `json:"suffixe"` // domaine imposé (".duckdns.org"), vide sinon
	Champs   []Champ `json:"champs"`
	Tutoriel []Etape `json:"tutoriel"`
	Note     string  `json:"note,omitempty"`
}

var liste = []Fournisseur{
	{
		ID:      DuckDNS,
		Nom:     "DuckDNS",
		Resume:  "Gratuit, sans nom de domaine à acheter : un sous-domaine en .duckdns.org en deux minutes.",
		Exemple: "exemple.duckdns.org",
		Suffixe: ".duckdns.org",
		Champs: []Champ{
			{Cle: "jeton", Libelle: "Jeton DuckDNS", Secret: true, Placeholder: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"},
		},
		Tutoriel: []Etape{
			{Texte: "Ouvre duckdns.org et connecte-toi avec un des comptes proposés en haut de la page (GitHub, Google, Reddit…). Aucun compte DuckDNS à créer.", Lien: "https://www.duckdns.org"},
			{Texte: "Dans le champ « sub domain », saisis le nom de ton choix (par exemple « exemple ») puis clique sur « add domain ». Ton domaine devient exemple.duckdns.org."},
			{Texte: "Recopie le « token » affiché en haut de la page : c'est le jeton à saisir dans le formulaire. Garde-le pour toi, il permet de modifier ton domaine."},
			{Texte: "Pas besoin de saisir l'IP sur le site : Aria fait pointer le domaine vers l'IP de ce PC tout seul (réglage « Publier l'IP de ce PC »)."},
		},
	},
	{
		ID:      Cloudflare,
		Nom:     "Cloudflare",
		Resume:  "Pour un nom de domaine à toi dont le DNS est géré par Cloudflare (offre gratuite suffisante).",
		Exemple: "vault.exemple.fr",
		Champs: []Champ{
			{Cle: "jeton", Libelle: "Jeton d'API Cloudflare", Secret: true, Placeholder: "Jeton créé dans « API Tokens »"},
		},
		Tutoriel: []Etape{
			{Texte: "Ton domaine doit déjà utiliser les serveurs DNS de Cloudflare (il apparaît comme « Active » dans ton tableau de bord)."},
			{Texte: "Ouvre la page des jetons : My Profile → API Tokens, puis « Create Token ».", Lien: "https://dash.cloudflare.com/profile/api-tokens"},
			{Texte: "Choisis le modèle « Edit zone DNS ». Dans « Permissions », garde Zone · DNS · Edit et ajoute une ligne Zone · Zone · Read (Aria en a besoin pour trouver ta zone)."},
			{Texte: "Dans « Zone Resources », choisis Include · Specific zone · ton domaine : le jeton ne pourra toucher à rien d'autre."},
			{Texte: "« Continue to summary » puis « Create Token ». Copie le jeton tout de suite : Cloudflare ne l'affiche qu'une fois."},
			{Texte: "Aria crée l'enregistrement A en « DNS only » (nuage gris) : c'est obligatoire, Cloudflare ne peut pas relayer vers une adresse privée."},
		},
	},
	{
		ID:      OVH,
		Nom:     "OVHcloud",
		Resume:  "Pour un nom de domaine à toi dont la zone DNS est hébergée chez OVHcloud.",
		Exemple: "vault.exemple.fr",
		Champs: []Champ{
			{Cle: "endpoint", Libelle: "Région du compte", Choix: []string{"ovh-eu", "ovh-ca", "ovh-us"}, Defaut: "ovh-eu"},
			{Cle: "applicationKey", Libelle: "Application key", Placeholder: "Clé d'application"},
			{Cle: "applicationSecret", Libelle: "Application secret", Secret: true, Placeholder: "Secret d'application"},
			{Cle: "consumerKey", Libelle: "Consumer key", Secret: true, Placeholder: "Clé consommateur"},
		},
		Tutoriel: []Etape{
			{Texte: "Ta zone DNS doit être chez OVHcloud : ton domaine apparaît dans « Web Cloud → Noms de domaine » avec un onglet « Zone DNS »."},
			{Texte: "Ouvre la page de création de clés d'API de ta région (le lien ci-dessous est celui de l'Europe, il préremplit les droits nécessaires) et connecte-toi.", Lien: OVHLienCles},
			{Texte: "Donne un nom (« Aria Vault ») et une description, puis choisis la validité « Unlimited » : sinon les clés expirent et le certificat ne se renouvelle plus."},
			{Texte: "Vérifie les droits : GET, POST, PUT et DELETE sur /domain/zone/*. Si le formulaire est vide, ajoute ces quatre lignes à la main."},
			{Texte: "Clique sur « Create keys » et recopie les trois valeurs : Application key, Application secret et Consumer key."},
		},
		Note: "Compte OVHcloud Canada ou États-Unis : choisis la région ovh-ca ou ovh-us, et crée les clés sur api.ca.ovhcloud.com ou api.us.ovhcloud.com.",
	},
	{
		ID:      DeSEC,
		Nom:     "deSEC",
		Resume:  "Hébergeur DNS gratuit et associatif : ton propre domaine, ou un sous-domaine gratuit en .dedyn.io.",
		Exemple: "exemple.dedyn.io",
		Champs: []Champ{
			{Cle: "jeton", Libelle: "Jeton deSEC", Secret: true, Placeholder: "Jeton créé dans « Token Management »"},
		},
		Tutoriel: []Etape{
			{Texte: "Crée un compte sur desec.io. À l'inscription, tu peux demander un domaine gratuit en .dedyn.io, ou ajouter ensuite ton propre domaine (en le faisant pointer vers les serveurs de deSEC).", Lien: "https://desec.io"},
			{Texte: "Une fois connecté, ouvre « Token Management » et crée un nouveau jeton (bouton +), nommé par exemple « Aria Vault ». Il n'a pas besoin du droit de gérer les autres jetons."},
			{Texte: "Copie le jeton tout de suite : deSEC ne le montre qu'une fois."},
		},
		Note: "deSEC impose une durée de cache d'une heure : après un changement d'IP, ton téléphone peut mettre jusqu'à une heure à suivre. Réserver l'IP de ce PC sur ta box évite le problème.",
	},
}

// OVHLienCles ouvre le formulaire de clés d'API OVHcloud (Europe) avec les
// droits nécessaires préremplis.
const OVHLienCles = "https://api.eu.ovhcloud.com/createToken/?GET=/domain/zone/*&POST=/domain/zone/*&PUT=/domain/zone/*&DELETE=/domain/zone/*"

// Liste renvoie les fournisseurs gérés, DuckDNS en premier.
func Liste() []Fournisseur { return liste }

// Trouver renvoie un fournisseur par son identifiant.
func Trouver(id string) (Fournisseur, bool) {
	for _, f := range liste {
		if f.ID == id {
			return f, true
		}
	}
	return Fournisseur{}, false
}

var (
	uuidRe  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	cleRe   = regexp.MustCompile(`^[!-~]{8,300}$`) // caractères imprimables, sans espace
	labelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	tldRe   = regexp.MustCompile(`^([a-z]{2,63}|xn--[a-z0-9-]{1,59})$`)
)

// NormaliserDomaine accepte « https://Vault.Exemple.fr/ » comme « vault.exemple.fr »
// et vérifie que le domaine convient au fournisseur.
func NormaliserDomaine(id, s string) (string, error) {
	f, ok := Trouver(id)
	if !ok {
		return "", fmt.Errorf("fournisseur DNS inconnu : %q", id)
	}
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(strings.TrimSuffix(s, "/"), ".")
	if s == "" {
		return "", errors.New("le domaine est obligatoire")
	}
	if len(s) > 253 {
		return "", errors.New("domaine trop long")
	}
	parts := strings.Split(s, ".")
	for _, p := range parts {
		if !labelRe.MatchString(p) {
			return "", fmt.Errorf("%q n'est pas un nom de domaine valide (exemple : %s)", s, f.Exemple)
		}
	}
	if len(parts) < 2 || !tldRe.MatchString(parts[len(parts)-1]) {
		return "", fmt.Errorf("%q n'est pas un nom de domaine complet (exemple : %s)", s, f.Exemple)
	}
	if f.Suffixe != "" && (!strings.HasSuffix(s, f.Suffixe) || len(parts) < 3) {
		return "", fmt.Errorf("%q n'est pas un sous-domaine %s (exemple : %s)", s, f.Nom, f.Exemple)
	}
	return s, nil
}

// Valider nettoie les identifiants saisis et renvoie les erreurs par champ.
func Valider(id string, ids map[string]string) (map[string]string, map[string]string) {
	f, ok := Trouver(id)
	if !ok {
		return nil, map[string]string{"fournisseur": "fournisseur DNS inconnu"}
	}
	propres := map[string]string{}
	erreurs := map[string]string{}
	for _, c := range f.Champs {
		v := strings.TrimSpace(ids[c.Cle])
		if v == "" {
			v = c.Defaut
		}
		switch {
		case v == "":
			erreurs[c.Cle] = c.Libelle + " : obligatoire"
		case len(c.Choix) > 0:
			if !contient(c.Choix, v) {
				erreurs[c.Cle] = c.Libelle + " : valeur inconnue"
			}
		case id == DuckDNS && c.Cle == "jeton":
			if !uuidRe.MatchString(v) {
				erreurs[c.Cle] = "le jeton DuckDNS doit ressembler à xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
			}
		default:
			if !cleRe.MatchString(v) {
				erreurs[c.Cle] = c.Libelle + " : valeur invalide (vérifie le copier-coller, sans espace)"
			}
		}
		propres[c.Cle] = v
	}
	return propres, erreurs
}

func contient(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// Client regroupe les opérations libdns utilisées par Aria.
type Client interface {
	libdns.RecordGetter
	libdns.RecordAppender
	libdns.RecordSetter
	libdns.RecordDeleter
}

// Nouveau fabrique le client libdns du fournisseur. Les identifiants doivent
// avoir été validés.
func Nouveau(id string, ids map[string]string) (Client, error) {
	switch id {
	case DuckDNS:
		return &duckdns.Provider{APIToken: ids["jeton"]}, nil
	case Cloudflare:
		return &cloudflare.Provider{APIToken: ids["jeton"]}, nil
	case OVH:
		return &ovh.Provider{
			Endpoint:          ids["endpoint"],
			ApplicationKey:    ids["applicationKey"],
			ApplicationSecret: ids["applicationSecret"],
			ConsumerKey:       ids["consumerKey"],
		}, nil
	case DeSEC:
		return &desec.Provider{Token: ids["jeton"]}, nil
	}
	return nil, fmt.Errorf("fournisseur DNS inconnu : %q", id)
}

// Publieur fait pointer un domaine vers une IPv4 chez son fournisseur.
type Publieur struct {
	ID      string
	Domaine string
	Client  Client
	// Zone trouve la zone DNS d'un nom (certmagic.FindZoneByFQDN en vrai).
	Zone func(ctx context.Context, fqdn string) (string, error)
}

// zone renvoie la zone et le nom relatif du domaine. DuckDNS n'a pas de vraie
// zone par utilisateur : le domaine lui-même sert de zone.
func (p Publieur) zone(ctx context.Context) (string, string, error) {
	if p.ID == DuckDNS {
		return p.Domaine + ".", "@", nil
	}
	z, err := p.Zone(ctx, p.Domaine+".")
	if err != nil {
		return "", "", fmt.Errorf("zone DNS de %s introuvable : %w", p.Domaine, err)
	}
	return z, libdns.RelativeName(p.Domaine+".", z), nil
}

// Publiee renvoie l'IPv4 actuellement enregistrée pour le domaine. Pour
// DuckDNS, la lecture passe par une requête DNS publique (lire n'est pas
// possible par son API sans modifier).
func (p Publieur) Publiee(ctx context.Context, lookup func(ctx context.Context, host string) ([]netip.Addr, error)) (netip.Addr, error) {
	if p.ID == DuckDNS {
		ips, err := lookup(ctx, p.Domaine)
		if err != nil {
			return netip.Addr{}, err
		}
		for _, ip := range ips {
			if ip.Is4() {
				return ip, nil
			}
		}
		return netip.Addr{}, nil
	}
	zone, nom, err := p.zone(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	recs, err := p.Client.GetRecords(ctx, zone)
	if err != nil {
		return netip.Addr{}, err
	}
	for _, r := range recs {
		rr := r.RR()
		if rr.Type != "A" || !strings.EqualFold(strings.TrimSuffix(rr.Name, "."), nom) {
			continue
		}
		if a, err := rr.Parse(); err == nil {
			if addr, ok := a.(libdns.Address); ok {
				return addr.IP, nil
			}
		}
	}
	return netip.Addr{}, nil // pas encore d'enregistrement A
}

// Publier remplace l'enregistrement A du domaine par ip.
func (p Publieur) Publier(ctx context.Context, ip netip.Addr) error {
	if !ip.Is4() {
		return errors.New("seules les adresses IPv4 sont publiées")
	}
	zone, nom, err := p.zone(ctx)
	if err != nil {
		return err
	}
	// 60 s de cache : un changement d'IP est suivi vite (deSEC impose lui-même son minimum d'une heure).
	_, err = p.Client.SetRecords(ctx, zone, []libdns.Record{libdns.Address{Name: nom, IP: ip, TTL: 60 * time.Second}})
	return err
}
