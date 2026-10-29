package dnsprov

import (
	"context"
	"net/netip"
	"testing"

	"github.com/libdns/libdns"
)

func TestNormaliserDomaine(t *testing.T) {
	ok := []struct{ f, in, want string }{
		{DuckDNS, "https://Vault.Exemple.DuckDNS.org/", "vault.exemple.duckdns.org"},
		{DuckDNS, "exemple.duckdns.org", "exemple.duckdns.org"},
		{Cloudflare, "vault.exemple.fr.", "vault.exemple.fr"},
		{OVH, "exemple.fr", "exemple.fr"},
		{DeSEC, "exemple.dedyn.io", "exemple.dedyn.io"},
	}
	for _, c := range ok {
		got, err := NormaliserDomaine(c.f, c.in)
		if err != nil || got != c.want {
			t.Errorf("%s %q → %q, %v ; attendu %q", c.f, c.in, got, err, c.want)
		}
	}
	bad := []struct{ f, in string }{
		{DuckDNS, ""}, {DuckDNS, "duckdns.org"}, {DuckDNS, "vault.exemple.fr"}, {DuckDNS, "-x.duckdns.org"},
		{DuckDNS, "a..duckdns.org"}, {Cloudflare, "localhost"}, {Cloudflare, "exemple.123"}, {OVH, "*.exemple.fr"},
		{"inconnu", "exemple.fr"},
	}
	for _, c := range bad {
		if _, err := NormaliserDomaine(c.f, c.in); err == nil {
			t.Errorf("%s %q accepté", c.f, c.in)
		}
	}
}

func TestValider(t *testing.T) {
	if _, errs := Valider(DuckDNS, map[string]string{"jeton": "0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0"}); len(errs) > 0 {
		t.Error(errs)
	}
	if _, errs := Valider(DuckDNS, map[string]string{"jeton": "abc"}); errs["jeton"] == "" {
		t.Error("jeton DuckDNS invalide accepté")
	}
	propres, errs := Valider(OVH, map[string]string{"applicationKey": "ak123456789", "applicationSecret": "as123456789", "consumerKey": "ck123456789"})
	if len(errs) > 0 || propres["endpoint"] != "ovh-eu" {
		t.Error(propres, errs)
	}
	if _, errs := Valider(OVH, map[string]string{"endpoint": "ovh-mars"}); errs["endpoint"] == "" || errs["consumerKey"] == "" {
		t.Error(errs)
	}
	if _, errs := Valider(Cloudflare, map[string]string{"jeton": "avec des espaces dedans"}); errs["jeton"] == "" {
		t.Error("espaces acceptés")
	}
	for _, f := range Liste() {
		if len(f.Tutoriel) < 3 {
			t.Errorf("%s : tutoriel trop court", f.ID)
		}
		if _, err := Nouveau(f.ID, map[string]string{}); err != nil {
			t.Error(err)
		}
	}
}

// faux fournisseur en mémoire
type faux struct {
	zone string
	recs []libdns.Record
}

func (f *faux) GetRecords(_ context.Context, zone string) ([]libdns.Record, error) {
	f.zone = zone
	return f.recs, nil
}
func (f *faux) AppendRecords(context.Context, string, []libdns.Record) ([]libdns.Record, error) {
	return nil, nil
}
func (f *faux) DeleteRecords(context.Context, string, []libdns.Record) ([]libdns.Record, error) {
	return nil, nil
}
func (f *faux) SetRecords(_ context.Context, zone string, r []libdns.Record) ([]libdns.Record, error) {
	f.zone, f.recs = zone, r
	return r, nil
}

func TestPublieur(t *testing.T) {
	f := &faux{recs: []libdns.Record{libdns.TXT{Name: "vault", Text: "x"}, libdns.Address{Name: "www", IP: netip.MustParseAddr("1.2.3.4")}}}
	p := Publieur{ID: Cloudflare, Domaine: "vault.exemple.fr", Client: f,
		Zone: func(context.Context, string) (string, error) { return "exemple.fr.", nil }}
	ctx := context.Background()
	if ip, err := p.Publiee(ctx, nil); err != nil || ip.IsValid() {
		t.Fatalf("aucun A attendu : %v %v", ip, err)
	}
	ip := netip.MustParseAddr("192.168.1.20")
	if err := p.Publier(ctx, ip); err != nil {
		t.Fatal(err)
	}
	if f.zone != "exemple.fr." || f.recs[0].RR().Name != "vault" {
		t.Errorf("zone %q, nom %q", f.zone, f.recs[0].RR().Name)
	}
	if got, err := p.Publiee(ctx, nil); err != nil || got != ip {
		t.Errorf("publiée = %v, %v", got, err)
	}
	if err := p.Publier(ctx, netip.MustParseAddr("fe80::1")); err == nil {
		t.Error("IPv6 publiée")
	}

	// DuckDNS : zone = le domaine, lecture par DNS.
	d := Publieur{ID: DuckDNS, Domaine: "exemple.duckdns.org", Client: f}
	if err := d.Publier(ctx, ip); err != nil || f.zone != "exemple.duckdns.org." || f.recs[0].RR().Name != "@" {
		t.Errorf("duckdns : %v %q %q", err, f.zone, f.recs[0].RR().Name)
	}
	lu, _ := d.Publiee(ctx, func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("::1"), ip}, nil
	})
	if lu != ip {
		t.Errorf("duckdns lu %v", lu)
	}
}
