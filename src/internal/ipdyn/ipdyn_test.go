package ipdyn

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"

	"aria-vault/internal/dnsprov"
)

type faux struct {
	recs   []libdns.Record
	appels int
}

func (f *faux) GetRecords(context.Context, string) ([]libdns.Record, error) { return f.recs, nil }
func (f *faux) AppendRecords(context.Context, string, []libdns.Record) ([]libdns.Record, error) {
	return nil, nil
}
func (f *faux) DeleteRecords(context.Context, string, []libdns.Record) ([]libdns.Record, error) {
	return nil, nil
}
func (f *faux) SetRecords(_ context.Context, _ string, r []libdns.Record) ([]libdns.Record, error) {
	f.appels++
	f.recs = r
	return r, nil
}

func TestVerifier(t *testing.T) {
	ip := netip.MustParseAddr("192.168.1.20")
	locale := Locale{IP: ip, Interface: "Ethernet", MAC: "3C-52-82-AA-BB-CC"}
	horloge := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	v := &Veilleur{
		locale: func() (Locale, error) { return locale, nil },
		now:    func() time.Time { return horloge },
	}
	f := &faux{recs: []libdns.Record{libdns.Address{Name: "vault", IP: netip.MustParseAddr("192.168.1.5")}}}
	p := &dnsprov.Publieur{ID: dnsprov.OVH, Domaine: "vault.exemple.fr", Client: f,
		Zone: func(context.Context, string) (string, error) { return "exemple.fr.", nil }}
	ctx := context.Background()

	// Publication désactivée : on lit, on ne touche à rien.
	if msg, err := v.Verifier(ctx, p, false); err != nil || msg != "" || f.appels != 0 {
		t.Fatalf("désactivé : %q %v %d", msg, err, f.appels)
	}
	if e := v.Etat(); e.Publiee.String() != "192.168.1.5" || e.Locale.MAC != locale.MAC {
		t.Errorf("état %+v", e)
	}

	msg, err := v.Verifier(ctx, p, true)
	if err != nil || f.appels != 1 || msg == "" {
		t.Fatalf("activé : %q %v %d", msg, err, f.appels)
	}
	if v.Etat().Publiee != ip {
		t.Error("adresse publiée non mise à jour")
	}
	// Déjà à jour : aucun appel.
	if _, err := v.Verifier(ctx, p, true); err != nil || f.appels != 1 {
		t.Errorf("republication inutile (%d)", f.appels)
	}

	// Adresse non privée (VPN, IPv4 publique) : rien n'est publié.
	locale.IP = netip.MustParseAddr("100.64.0.3")
	if _, err := v.Verifier(ctx, p, true); err != nil || f.appels != 1 || v.Etat().Avertissement == "" {
		t.Errorf("adresse non privée publiée : %d %q", f.appels, v.Etat().Avertissement)
	}

	// Pas encore configuré : seule la carte réseau est relevée.
	if _, err := v.Verifier(ctx, nil, true); err != nil {
		t.Error(err)
	}
}

// DuckDNS : le cache DNS peut montrer l'ancienne adresse juste après la mise à jour.
func TestCacheDuckDNS(t *testing.T) {
	ip := netip.MustParseAddr("192.168.1.20")
	horloge := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	v := &Veilleur{
		locale: func() (Locale, error) { return Locale{IP: ip}, nil },
		lookup: func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("192.168.1.5")}, nil // cache périmé
		},
		now: func() time.Time { return horloge },
	}
	f := &faux{}
	p := &dnsprov.Publieur{ID: dnsprov.DuckDNS, Domaine: "exemple.duckdns.org", Client: f}
	ctx := context.Background()
	v.Verifier(ctx, p, true)
	v.Verifier(ctx, p, true)
	if f.appels != 1 {
		t.Errorf("republié pendant le cache : %d appels", f.appels)
	}
	horloge = horloge.Add(3 * Intervalle)
	v.Verifier(ctx, p, true)
	if f.appels != 2 {
		t.Errorf("pas de nouvel essai après le délai : %d appels", f.appels)
	}
}

func TestPubliable(t *testing.T) {
	for s, want := range map[string]bool{"192.168.1.2": true, "10.0.0.8": true, "172.20.1.1": true, "100.64.0.1": false, "8.8.8.8": false, "fd00::1": false} {
		if Publiable(netip.MustParseAddr(s)) != want {
			t.Errorf("%s", s)
		}
	}
}
