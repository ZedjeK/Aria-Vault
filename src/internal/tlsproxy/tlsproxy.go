// Package tlsproxy sert le coffre en HTTPS : certificat Let's Encrypt obtenu et
// renouvelé par certmagic (le moteur de Caddy) en validation DNS-01 chez DuckDNS,
// puis relais vers Vaultwarden qui n'écoute que sur 127.0.0.1.
//
// Aucun port n'a besoin d'être ouvert sur la box : DNS-01 prouve le contrôle du
// domaine par un enregistrement TXT, sans connexion entrante.
package tlsproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/duckdns"
	"go.uber.org/zap"

	"aria-vault/internal/ports"
)

// Options règle le relais HTTPS.
type Options struct {
	Domaine     string // vault.exemple.duckdns.org
	Jeton       string // jeton DuckDNS
	Email       string // facultatif
	Staging     bool   // CA de test Let's Encrypt (certificat non reconnu, sans limite de fréquence)
	Certificats string // dossier de stockage des certificats
	Ecoute      string // ":443"
	Cible       string // "http://127.0.0.1:8000"
	Logger      *zap.Logger
}

// Proxy est le relais HTTPS en cours d'exécution.
type Proxy struct {
	srv     *http.Server
	cache   *certmagic.Cache
	tlsCfg  *tls.Config
	domaine string
	done    chan error
}

// Start ouvre le port, obtient (ou recharge) le certificat, puis sert le coffre.
// L'obtention d'un premier certificat peut prendre quelques minutes : DuckDNS
// doit publier l'enregistrement TXT avant que Let's Encrypt ne le vérifie.
func Start(ctx context.Context, o Options) (*Proxy, error) {
	target, err := url.Parse(o.Cible)
	if err != nil {
		return nil, err
	}
	logger := o.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Le port est pris d'abord : une erreur claire tout de suite plutôt
	// qu'après plusieurs minutes de demande de certificat.
	ln, err := net.Listen("tcp", o.Ecoute)
	if err != nil {
		return nil, ports.Describe(o.Ecoute)
	}

	var cfg *certmagic.Config
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) { return cfg, nil },
		Logger:           logger,
	})
	cfg = certmagic.New(cache, certmagic.Config{
		Storage: &certmagic.FileStorage{Path: o.Certificats},
		Logger:  logger,
	})
	ca := certmagic.LetsEncryptProductionCA
	if o.Staging {
		ca = certmagic.LetsEncryptStagingCA
	}
	cfg.Issuers = []certmagic.Issuer{certmagic.NewACMEIssuer(cfg, certmagic.ACMEIssuer{
		CA:                      ca,
		Email:                   o.Email,
		Agreed:                  true,
		DisableHTTPChallenge:    true,
		DisableTLSALPNChallenge: true,
		DNS01Solver: &certmagic.DNS01Solver{
			DNSManager: certmagic.DNSManager{
				DNSProvider: &duckdns.Provider{APIToken: o.Jeton},
			},
		},
		Logger: logger,
	})}

	if err := cfg.ManageSync(ctx, []string{o.Domaine}); err != nil {
		ln.Close()
		cache.Stop()
		return nil, err
	}

	tlsCfg := cfg.TLSConfig()
	tlsCfg.NextProtos = []string{"h2", "http/1.1"}
	tlsCfg.MinVersion = tls.VersionTLS12

	p := &Proxy{
		srv: &http.Server{
			Handler:           newHandler(target),
			TLSConfig:         tlsCfg,
			ReadHeaderTimeout: 15 * time.Second,
			IdleTimeout:       2 * time.Minute,
			// Pas de WriteTimeout : WebSockets et pièces jointes peuvent durer.
			ErrorLog: zap.NewStdLog(logger.Named("http")),
		},
		cache:   cache,
		tlsCfg:  tlsCfg,
		domaine: o.Domaine,
		done:    make(chan error, 1),
	}
	go func() {
		err := p.srv.Serve(tls.NewListener(ln, tlsCfg))
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		p.done <- err
	}()
	return p, nil
}

// Expiry renvoie la date de fin du certificat servi (renouvelé automatiquement
// par certmagic environ 30 jours avant), ou zéro s'il est introuvable.
func (p *Proxy) Expiry() time.Time {
	cert, err := p.tlsCfg.GetCertificate(&tls.ClientHelloInfo{ServerName: p.domaine})
	if err != nil || cert == nil {
		return time.Time{}
	}
	leaf := cert.Leaf
	if leaf == nil && len(cert.Certificate) > 0 {
		if leaf, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
			return time.Time{}
		}
	}
	if leaf == nil {
		return time.Time{}
	}
	return leaf.NotAfter
}

// Done reçoit l'erreur du serveur s'il s'arrête de lui-même.
func (p *Proxy) Done() <-chan error { return p.done }

// Stop ferme les connexions et arrête le renouvellement.
func (p *Proxy) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = p.srv.Shutdown(ctx)
	p.cache.Stop()
}

// newHandler relaie vers Vaultwarden en conservant l'hôte demandé et en
// transmettant la vraie IP du client (pour le limiteur de tentatives).
func newHandler(target *url.URL) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = pr.In.Host
			// X-Forwarded-* entrants sont déjà retirés par ReverseProxy.
			pr.SetXForwarded()
			// Vaultwarden lit X-Real-IP par défaut ; une valeur envoyée par le
			// client est écrasée, elle ne peut donc pas tromper le limiteur.
			pr.Out.Header.Del("X-Real-IP")
			if ip, _, err := net.SplitHostPort(pr.In.RemoteAddr); err == nil {
				pr.Out.Header.Set("X-Real-IP", ip)
			}
		},
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "Vaultwarden ne répond pas", http.StatusBadGateway)
		},
	}
}
