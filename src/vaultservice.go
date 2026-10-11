package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"aria-vaultwarden/internal/config"
	"aria-vaultwarden/internal/journal"
	"aria-vaultwarden/internal/moteur"
	"aria-vaultwarden/internal/paths"
	"aria-vaultwarden/internal/vaultwarden"
)

// VaultService expose au frontend les actions d'Aria Vaultwarden.
type VaultService struct {
	p      paths.Paths
	j      *journal.Journal
	moteur *moteur.Moteur
	app    *application.App // renseigné après application.New

	cfgMu sync.Mutex // protège config.json (lecture-modification-écriture)
	svgMu sync.Mutex // une seule sauvegarde ou restauration à la fois
}

// View est l'état complet affiché par l'interface.
type View struct {
	Moteur          moteur.Status `json:"moteur"`
	Configure       bool          `json:"configure"`
	Domaine         string        `json:"domaine"`
	Email           string        `json:"email"`
	JetonEnregistre bool          `json:"jetonEnregistre"`
	Inscriptions    bool          `json:"inscriptions"` // réglage enregistré
	Autostart       bool          `json:"autostart"`
	Version         string        `json:"version"`
	Vaultwarden     string        `json:"vaultwarden"`
	Donnees         string        `json:"donnees"`
	Base            string        `json:"base"`
	Erreur          string        `json:"erreur"` // config.json illisible
}

// Status renvoie l'état de l'appli.
func (s *VaultService) Status() View {
	v := View{
		Moteur:      s.moteur.Status(),
		Version:     version,
		Vaultwarden: vaultwarden.Version(),
		Donnees:     s.p.Donnees,
		Base:        s.p.Base,
	}
	cfg, err := s.lireConfig()
	if err != nil {
		v.Erreur = err.Error()
	} else {
		v.Configure = cfg.Valider() == nil
		v.Domaine, v.Email, v.Inscriptions = cfg.Domaine, cfg.Email, cfg.Inscriptions
		v.JetonEnregistre = len(cfg.JetonChiffre) > 0
	}
	if s.app != nil {
		v.Autostart, _ = s.app.Autostart.IsEnabled()
	}
	return v
}

// Journal renvoie les derniers événements.
func (s *VaultService) Journal() []journal.Entry { return s.j.Entries() }

// ConfigInput est le formulaire de la page HTTPS.
type ConfigInput struct {
	Domaine string `json:"domaine"`
	Jeton   string `json:"jeton"` // vide = garder l'actuel
	Email   string `json:"email"`
}

// SaveResult renvoie les erreurs par champ, ou une erreur générale.
type SaveResult struct {
	Fields map[string]string `json:"fields"`
	Error  string            `json:"error"`
}

// SaveConfig enregistre la configuration HTTPS et relance le coffre s'il tournait.
func (s *VaultService) SaveConfig(in ConfigInput) SaveResult {
	res := SaveResult{Fields: map[string]string{}}
	s.cfgMu.Lock()
	enregistre := false
	defer func() {
		if !enregistre {
			s.cfgMu.Unlock()
		}
	}()
	cfg, err := config.Load(s.p.Config)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	if d, err := config.NormaliserDomaine(in.Domaine); err != nil {
		res.Fields["domaine"] = err.Error()
	} else {
		cfg.Domaine = d
	}
	if j := strings.TrimSpace(in.Jeton); j != "" {
		if err := cfg.SetJeton(j); err != nil {
			res.Fields["jeton"] = err.Error()
		}
	} else if len(cfg.JetonChiffre) == 0 {
		res.Fields["jeton"] = "le jeton DuckDNS est obligatoire"
	}
	email := strings.TrimSpace(in.Email)
	if email != "" && (!strings.Contains(email, "@") || strings.ContainsAny(email, " \t")) {
		res.Fields["email"] = "adresse e-mail invalide"
	}
	cfg.Email = email
	if len(res.Fields) > 0 {
		res.Error = "vérifie les champs signalés"
		return res
	}

	err = cfg.Save(s.p.Config)
	enregistre = true
	s.cfgMu.Unlock()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	s.j.Add(journal.OK, "Configuration enregistrée pour %s (jeton chiffré pour ton compte Windows)", cfg.Domaine)
	if err := s.relancerSiActif(); err != nil {
		res.Error = err.Error()
	}
	return res
}

// Start démarre le coffre.
func (s *VaultService) Start() error { return s.moteur.Start() }

// Stop arrête le coffre.
func (s *VaultService) Stop() { s.moteur.Stop() }

// Restart relance le coffre.
func (s *VaultService) Restart() error { return s.moteur.Restart() }

// SetInscriptions ouvre ou ferme la création de comptes.
func (s *VaultService) SetInscriptions(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.Inscriptions = on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Inscriptions ouvertes")
	} else {
		s.j.Add(journal.Info, "Inscriptions fermées")
	}
	return s.relancerSiActif()
}

// SetAutostart active ou non le lancement silencieux à l'ouverture de session.
func (s *VaultService) SetAutostart(on bool) error {
	if on {
		if err := s.app.Autostart.EnableWithOptions(application.AutostartOptions{Arguments: []string{argSilencieux}}); err != nil {
			return err
		}
		s.j.Add(journal.Info, "Lancement à l'ouverture de session activé")
		return nil
	}
	if err := s.app.Autostart.Disable(); err != nil {
		return err
	}
	s.j.Add(journal.Info, "Lancement à l'ouverture de session désactivé")
	return nil
}

// OpenVault ouvre le coffre web dans le navigateur.
func (s *VaultService) OpenVault() error {
	cfg, err := s.lireConfig()
	if err != nil {
		return err
	}
	if cfg.Domaine == "" {
		return fmt.Errorf("aucun domaine configuré")
	}
	return s.app.Browser.OpenURL("https://" + cfg.Domaine)
}

// OpenDataFolder ouvre le dossier du coffre dans l'Explorateur.
func (s *VaultService) OpenDataFolder() error { return s.app.Browser.OpenFile(s.p.Donnees) }

// relancerSiActif applique un réglage en relançant le coffre s'il tourne.
func (s *VaultService) relancerSiActif() error {
	switch s.moteur.Status().Etat {
	case moteur.Actif, moteur.Demarrage, moteur.Erreur:
		return s.moteur.Restart()
	}
	return nil
}
