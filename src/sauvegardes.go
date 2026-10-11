package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aria-vaultwarden/internal/config"
	"aria-vaultwarden/internal/journal"
	"aria-vaultwarden/internal/moteur"
	"aria-vaultwarden/internal/paths"
	"aria-vaultwarden/internal/sauvegarde"
)

// BackupView est l'état de la page Sauvegardes.
type BackupView struct {
	Dossier   string            `json:"dossier"`
	ParDefaut bool              `json:"parDefaut"`
	Auto      bool              `json:"auto"`
	Derniere  time.Time         `json:"derniere"`
	Liste     []sauvegarde.Info `json:"liste"`
	Garder    int               `json:"garder"`
	Erreur    string            `json:"erreur"`
}

// Backups renvoie la configuration et la liste des sauvegardes.
func (s *VaultService) Backups() BackupView {
	cfg, err := s.lireConfig()
	v := BackupView{Garder: sauvegarde.Garder}
	if err != nil {
		v.Erreur = err.Error()
		return v
	}
	v.Dossier, v.ParDefaut = dossierSauvegarde(cfg)
	v.Auto, v.Derniere = !cfg.SansSauvegardeAuto, cfg.DerniereSauvegarde
	if v.Liste, err = sauvegarde.Lister(v.Dossier); err != nil {
		v.Erreur = err.Error()
	}
	return v
}

func dossierSauvegarde(cfg *config.Config) (string, bool) {
	if cfg.DossierSauvegarde != "" {
		return cfg.DossierSauvegarde, false
	}
	return paths.SauvegardesParDefaut(), true
}

// Backup sauvegarde le coffre maintenant. Si le serveur tourne, il est
// arrêté le temps de la copie (quelques secondes) puis relancé.
func (s *VaultService) Backup() (sauvegarde.Info, error) {
	s.svgMu.Lock()
	defer s.svgMu.Unlock()

	cfg, err := s.lireConfig()
	if err != nil {
		return sauvegarde.Info{}, err
	}
	dest, _ := dossierSauvegarde(cfg)

	enLigne := s.pauseServeur("sauvegarde")
	info, err := sauvegarde.Creer(s.p.Donnees, dest, time.Now())
	s.repriseServeur(enLigne)
	if err != nil {
		s.j.Add(journal.Erreur, "Sauvegarde impossible : %v", err)
		return sauvegarde.Info{}, err
	}

	s.j.Add(journal.OK, "Sauvegarde créée : %s (%s)", info.Nom, taille(info.Taille))
	if supprimees, err := sauvegarde.Purger(dest, sauvegarde.Garder); err != nil {
		s.j.Add(journal.Erreur, "Rotation des sauvegardes : %v", err)
	} else if len(supprimees) > 0 {
		s.j.Add(journal.Info, "Rotation : %d ancienne(s) sauvegarde(s) supprimée(s)", len(supprimees))
	}
	if err := s.majConfig(func(c *config.Config) { c.DerniereSauvegarde = info.Date }); err != nil {
		s.j.Add(journal.Erreur, "Date de sauvegarde non enregistrée : %v", err)
	}
	return info, nil
}

// Restore remplace le coffre par une sauvegarde de la liste.
func (s *VaultService) Restore(nom string) error {
	if _, ok := sauvegarde.DateDe(nom); !ok {
		return errors.New("nom de sauvegarde invalide")
	}
	cfg, err := s.lireConfig()
	if err != nil {
		return err
	}
	dest, _ := dossierSauvegarde(cfg)
	return s.remplacerCoffre("Restauration de "+nom, func(now time.Time) (string, error) {
		return sauvegarde.Restaurer(filepath.Join(dest, nom), s.p.Donnees, now)
	})
}

// ImportVault remplace le coffre par un ancien dossier data choisi par l'utilisateur.
// Renvoie false si l'utilisateur a annulé.
func (s *VaultService) ImportVault() (bool, error) {
	src, err := s.app.Dialog.OpenFile().
		SetTitle("Choisis l'ancien dossier data (celui qui contient db.sqlite3)").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		PromptForSingleSelection()
	if err != nil || src == "" {
		return false, err
	}
	return true, s.remplacerCoffre("Import de "+src, func(now time.Time) (string, error) {
		return sauvegarde.Importer(src, s.p.Donnees, now)
	})
}

func (s *VaultService) remplacerCoffre(titre string, fn func(time.Time) (string, error)) error {
	s.svgMu.Lock()
	defer s.svgMu.Unlock()

	enLigne := s.pauseServeur("restauration")
	ancien, err := fn(time.Now())
	if err != nil {
		s.j.Add(journal.Erreur, "%s : %v", titre, err)
		s.repriseServeur(enLigne)
		return err
	}
	if ancien != "" {
		s.j.Add(journal.Info, "Ancien coffre mis de côté : %s", ancien)
	}
	s.j.Add(journal.OK, "%s terminée", titre)
	// Après une restauration, le serveur repart s'il est configuré, même s'il était arrêté.
	s.repriseServeur(enLigne || s.moteur.Configure())
	return nil
}

// pauseServeur arrête le serveur s'il tourne et indique s'il faudra le relancer.
func (s *VaultService) pauseServeur(motif string) bool {
	switch s.moteur.Status().Etat {
	case moteur.Actif, moteur.Demarrage:
		s.j.Add(journal.Info, "Serveur mis en pause pour la %s", motif)
		s.moteur.Stop()
		return true
	}
	return false
}

func (s *VaultService) repriseServeur(relancer bool) {
	if !relancer {
		return
	}
	if err := s.moteur.Start(); err != nil {
		s.j.Add(journal.Erreur, "Relance du serveur : %v", err)
	}
}

// ChooseBackupFolder change le dossier des sauvegardes. Renvoie false si annulé.
func (s *VaultService) ChooseBackupFolder() (bool, error) {
	cfg, err := s.lireConfig()
	if err != nil {
		return false, err
	}
	actuel, _ := dossierSauvegarde(cfg)
	dir, err := s.app.Dialog.OpenFile().
		SetTitle("Dossier des sauvegardes (idéalement OneDrive ou un autre disque)").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		SetDirectory(actuel).
		PromptForSingleSelection()
	if err != nil || dir == "" {
		return false, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}
	if err := s.majConfig(func(c *config.Config) { c.DossierSauvegarde = dir }); err != nil {
		return false, err
	}
	s.j.Add(journal.Info, "Dossier des sauvegardes : %s", dir)
	return true, nil
}

// OpenBackupFolder ouvre le dossier des sauvegardes dans l'Explorateur.
func (s *VaultService) OpenBackupFolder() error {
	cfg, err := s.lireConfig()
	if err != nil {
		return err
	}
	dir, _ := dossierSauvegarde(cfg)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return s.app.Browser.OpenFile(dir)
}

// SetDailyBackup active ou non la sauvegarde quotidienne.
func (s *VaultService) SetDailyBackup(on bool) error {
	if err := s.majConfig(func(c *config.Config) { c.SansSauvegardeAuto = !on }); err != nil {
		return err
	}
	if on {
		s.j.Add(journal.Info, "Sauvegarde quotidienne activée")
	} else {
		s.j.Add(journal.Info, "Sauvegarde quotidienne désactivée")
	}
	return nil
}

// boucleQuotidienne sauvegarde le coffre une fois par 24 h tant qu'Aria tourne.
func (s *VaultService) boucleQuotidienne(ctx context.Context) {
	verifier := func() {
		cfg, err := s.lireConfig()
		if err != nil || cfg.SansSauvegardeAuto || !s.moteur.Configure() {
			return
		}
		if time.Since(cfg.DerniereSauvegarde) < 24*time.Hour {
			return
		}
		if _, err := os.Stat(filepath.Join(s.p.Donnees, "db.sqlite3")); err != nil {
			return // pas encore de coffre
		}
		s.j.Add(journal.Info, "Sauvegarde quotidienne")
		_, _ = s.Backup()
	}
	// Premier passage une fois le serveur en ligne, puis toutes les 30 min.
	t := time.NewTimer(2 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			verifier()
			t.Reset(30 * time.Minute)
		}
	}
}

// ---------- Réglages d'Aria (domaine, e-mail, jeton) ----------

// ExportSettings enregistre les réglages dans un fichier choisi par l'utilisateur.
// Renvoie false si annulé.
func (s *VaultService) ExportSettings(phrase string) (bool, error) {
	cfg, err := s.lireConfig()
	if err != nil {
		return false, err
	}
	data, err := cfg.Exporter(phrase)
	if err != nil {
		return false, err
	}
	dir, _ := dossierSauvegarde(cfg)
	_ = os.MkdirAll(dir, 0o700)
	dst, err := s.app.Dialog.SaveFile().
		SetMessage("Enregistrer les réglages d'Aria Vault").
		SetDirectory(dir).
		SetFilename("reglages-aria-vault.json").
		AddFilter("Réglages Aria Vault", "*.json").
		PromptForSingleSelection()
	if err != nil || dst == "" {
		return false, err
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return false, err
	}
	s.j.Add(journal.OK, "Réglages exportés : %s (jeton chiffré par ta phrase de passe)", dst)
	return true, nil
}

// ImportResult décrit le résultat d'un import de réglages.
type ImportResult struct {
	Annule  bool   `json:"annule"`
	Domaine string `json:"domaine"`
	Jeton   bool   `json:"jeton"` // le jeton a été importé
	Error   string `json:"error"`
}

// ImportSettings lit un fichier de réglages. Phrase vide : domaine et e-mail
// seulement. Le serveur est relancé s'il tournait.
func (s *VaultService) ImportSettings(phrase string) ImportResult {
	src, err := s.app.Dialog.OpenFile().
		SetTitle("Fichier de réglages Aria Vault").
		AddFilter("Réglages Aria Vault", "*.json").
		PromptForSingleSelection()
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	if src == "" {
		return ImportResult{Annule: true}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	imp, err := config.LireExport(data, phrase)
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	var errJeton error
	err = s.majConfig(func(c *config.Config) {
		c.Domaine, c.Email = imp.Domaine, imp.Email
		if imp.Jeton != "" {
			errJeton = c.SetJeton(imp.Jeton) // rechiffré avec DPAPI pour ce compte Windows
		}
	})
	if err == nil {
		err = errJeton
	}
	if err != nil {
		return ImportResult{Error: err.Error()}
	}
	if imp.Jeton != "" {
		s.j.Add(journal.OK, "Réglages importés pour %s, jeton compris", imp.Domaine)
	} else {
		s.j.Add(journal.OK, "Réglages importés pour %s (jeton à ressaisir sur la page HTTPS)", imp.Domaine)
	}
	if err := s.relancerSiActif(); err != nil {
		return ImportResult{Domaine: imp.Domaine, Jeton: imp.Jeton != "", Error: err.Error()}
	}
	return ImportResult{Domaine: imp.Domaine, Jeton: imp.Jeton != ""}
}

// ---------- Accès à config.json ----------

func (s *VaultService) lireConfig() (*config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	return config.Load(s.p.Config)
}

// majConfig lit, modifie et réenregistre config.json sans course entre appels.
func (s *VaultService) majConfig(fn func(*config.Config)) error {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	cfg, err := config.Load(s.p.Config)
	if err != nil {
		return err
	}
	fn(cfg)
	return cfg.Save(s.p.Config)
}

func taille(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f Mo", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f Ko", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d o", n)
}
