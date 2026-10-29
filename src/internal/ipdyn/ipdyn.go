// Package ipdyn fait pointer le domaine vers l'adresse IP locale de ce PC.
//
// Toutes les 10 minutes : l'adresse privée de la carte réseau qui mène à
// Internet est comparée à l'adresse publiée chez le fournisseur DNS ; si elles
// diffèrent, l'enregistrement A est mis à jour. Réserver l'adresse du PC sur
// la box reste conseillé : entre deux vérifications, ou le temps que les caches
// DNS suivent, le téléphone chercherait le coffre à l'ancienne adresse.
package ipdyn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"aria-vault/internal/dnsprov"
)

// Intervalle entre deux vérifications.
const Intervalle = 10 * time.Minute

// Locale décrit la carte réseau de ce PC qui mène à Internet.
type Locale struct {
	IP        netip.Addr `json:"ip"`
	Interface string     `json:"interface"`
	MAC       string     `json:"mac"`
}

// TrouverLocale renvoie l'adresse IPv4 utilisée pour sortir vers Internet.
// Aucun paquet n'est envoyé : « se connecter » en UDP choisit seulement la route.
func TrouverLocale() (Locale, error) {
	c, err := net.Dial("udp4", "1.1.1.1:53")
	if err != nil {
		return Locale{}, fmt.Errorf("aucune connexion réseau : %w", err)
	}
	defer c.Close()
	ap, err := netip.ParseAddrPort(c.LocalAddr().String())
	if err != nil {
		return Locale{}, err
	}
	l := Locale{IP: ap.Addr().Unmap()}
	if ifs, err := net.Interfaces(); err == nil {
		for _, i := range ifs {
			addrs, _ := i.Addrs()
			for _, a := range addrs {
				if n, ok := a.(*net.IPNet); ok {
					if ip, ok := netip.AddrFromSlice(n.IP); ok && ip.Unmap() == l.IP {
						l.Interface, l.MAC = i.Name, formatMAC(i.HardwareAddr)
					}
				}
			}
		}
	}
	return l, nil
}

// formatMAC écrit l'adresse comme Windows (ipconfig /all) : 3C-52-82-AA-BB-CC.
func formatMAC(hw net.HardwareAddr) string {
	if len(hw) == 0 {
		return ""
	}
	s := ""
	for i, b := range hw {
		if i > 0 {
			s += "-"
		}
		s += fmt.Sprintf("%02X", b)
	}
	return s
}

// Publiable indique si l'adresse peut servir au coffre : une adresse privée
// du réseau local (192.168.x.x, 10.x.x.x, 172.16-31.x.x).
func Publiable(ip netip.Addr) bool { return ip.Is4() && ip.IsPrivate() }

// LookupPublic interroge des résolveurs publics (et non la box, qui peut
// filtrer les réponses en adresse privée).
func LookupPublic(ctx context.Context, host string) ([]netip.Addr, error) {
	var dernier error
	for _, srv := range []string{"1.1.1.1:53", "9.9.9.9:53", "8.8.8.8:53"} {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, network, srv)
			},
		}
		ips, err := r.LookupNetIP(ctx, "ip4", host)
		if err == nil {
			for i := range ips {
				ips[i] = ips[i].Unmap()
			}
			return ips, nil
		}
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return nil, nil // pas encore d'adresse publiée
		}
		dernier = err
	}
	return nil, dernier
}

// Etat est affiché par la page HTTPS.
type Etat struct {
	Actif         bool       `json:"actif"`         // publication automatique activée
	Locale        Locale     `json:"locale"`        // carte réseau de ce PC
	Publiee       netip.Addr `json:"publiee"`       // adresse enregistrée chez le fournisseur
	Verifie       time.Time  `json:"verifie"`       // dernière vérification réussie
	Modifie       time.Time  `json:"modifie"`       // dernière mise à jour faite par Aria
	Erreur        string     `json:"erreur"`        // dernière erreur
	Avertissement string     `json:"avertissement"` // adresse non publiable, etc.
}

// Veilleur garde l'état entre deux vérifications.
type Veilleur struct {
	mu   sync.Mutex
	etat Etat
	// Après une mise à jour, la lecture DNS peut renvoyer l'ancienne adresse
	// le temps du cache : on ne republie pas la même adresse pendant ce délai.
	derniereIP netip.Addr
	// injectables pour les tests
	locale func() (Locale, error)
	lookup func(ctx context.Context, host string) ([]netip.Addr, error)
	now    func() time.Time
}

// NouveauVeilleur prépare un veilleur branché sur le vrai réseau.
func NouveauVeilleur() *Veilleur {
	return &Veilleur{locale: TrouverLocale, lookup: LookupPublic, now: time.Now}
}

// Etat renvoie l'état courant.
func (v *Veilleur) Etat() Etat {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.etat
}

// Locale rafraîchit et renvoie la carte réseau (même publication désactivée).
func (v *Veilleur) Locale() Locale {
	l, err := v.locale()
	v.mu.Lock()
	defer v.mu.Unlock()
	if err == nil {
		v.etat.Locale = l
	}
	return v.etat.Locale
}

// Verifier fait une passe : lit les deux adresses et publie si besoin.
// Le message renvoyé décrit une mise à jour faite ("" sinon).
func (v *Veilleur) Verifier(ctx context.Context, p *dnsprov.Publieur, actif bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	l, errLocale := v.locale()
	v.mu.Lock()
	v.etat.Actif = actif
	if errLocale == nil {
		v.etat.Locale = l
	}
	v.mu.Unlock()
	if p == nil {
		return "", nil // pas encore configuré
	}

	publiee, err := p.Publiee(ctx, v.lookup)
	if err != nil {
		return "", v.echec(fmt.Errorf("lecture de l'adresse publiée : %w", err))
	}
	v.mu.Lock()
	v.etat.Publiee, v.etat.Verifie, v.etat.Erreur, v.etat.Avertissement = publiee, v.now(), "", ""
	v.mu.Unlock()

	switch {
	case !actif:
		return "", nil
	case errLocale != nil:
		return "", v.echec(errLocale)
	case !Publiable(l.IP):
		v.mu.Lock()
		v.etat.Avertissement = fmt.Sprintf("%s n'est pas une adresse de réseau local : rien n'est publié", l.IP)
		v.mu.Unlock()
		return "", nil
	case publiee == l.IP:
		return "", nil
	}
	v.mu.Lock()
	recent := v.derniereIP == l.IP && v.now().Sub(v.etat.Modifie) < 2*Intervalle
	v.mu.Unlock()
	if recent {
		return "", nil // déjà publiée, les caches DNS n'ont pas encore suivi
	}

	if err := p.Publier(ctx, l.IP); err != nil {
		return "", v.echec(fmt.Errorf("mise à jour de l'adresse chez le fournisseur : %w", err))
	}
	v.mu.Lock()
	v.derniereIP, v.etat.Publiee, v.etat.Modifie = l.IP, l.IP, v.now()
	v.mu.Unlock()
	if publiee.IsValid() {
		return fmt.Sprintf("Adresse de %s mise à jour : %s → %s", p.Domaine, publiee, l.IP), nil
	}
	return fmt.Sprintf("Adresse de %s publiée : %s", p.Domaine, l.IP), nil
}

func (v *Veilleur) echec(err error) error {
	v.mu.Lock()
	v.etat.Erreur = err.Error()
	v.mu.Unlock()
	return err
}
