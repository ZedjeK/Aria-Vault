// Aria Vaultwarden : Vaultwarden et son HTTPS dans un seul exe Windows.
//
// Deux modes :
//   - silencieux (--silencieux, utilisé au démarrage de session) : le coffre
//     démarre, Aria reste en icône dans la zone de notification ;
//   - logiciel : la fenêtre s'ouvre. La fermer la renvoie dans la zone de
//     notification, le coffre continue de tourner.
package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"aria-vaultwarden/internal/journal"
	"aria-vaultwarden/internal/moteur"
	"aria-vaultwarden/internal/paths"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/icon.ico
var appIcon []byte

//go:embed build/tray-on.ico
var trayOn []byte

//go:embed build/tray-off.ico
var trayOff []byte

// version est fixée à la compilation (-X main.version=…).
var version = "dev"

const argSilencieux = "--silencieux"

func main() {
	silencieux := slices.Contains(os.Args[1:], argSilencieux)

	p, err := paths.Default()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(p.Base, 0o700); err != nil {
		log.Fatal(err)
	}
	j := journal.New(filepath.Join(p.Base, "aria.log"))
	if silencieux {
		j.Add(journal.Info, "Aria Vaultwarden %s lancé en mode silencieux", version)
	} else {
		j.Add(journal.Info, "Aria Vaultwarden %s lancé", version)
	}

	var (
		window *application.WebviewWindow
		tray   *application.SystemTray
		toggle *application.MenuItem
	)
	svc := &VaultService{p: p, j: j}
	ctx, arreter := context.WithCancel(context.Background())

	montrer := func() {
		if window == nil {
			return
		}
		window.Show()
		window.Restore()
		window.Focus()
	}

	app := application.New(application.Options{
		Name:        "Aria Vaultwarden",
		Description: "Ton coffre Vaultwarden en HTTPS, depuis la zone de notification",
		Icon:        appIcon,
		Services:    []application.Service{application.NewService(svc)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "io.github.zedjek.ariavaultwarden",
			// Relancer l'exe (double-clic, raccourci) montre la fenêtre déjà ouverte.
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				application.InvokeAsync(montrer)
			},
		},
		Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		OnShutdown: func() {
			arreter()
			svc.moteur.Stop()
		},
	})
	svc.app = app

	// Icône, info-bulle et menu suivent l'état du coffre.
	majTray := func() {
		if tray == nil {
			return
		}
		s := svc.moteur.Status()
		icon, tip, action := trayOff, "Aria Vaultwarden · serveur arrêté", "Démarrer le serveur"
		switch s.Etat {
		case moteur.Actif:
			icon, tip, action = trayOn, "Aria Vaultwarden · en ligne : "+s.Domaine, "Arrêter le serveur"
		case moteur.Demarrage:
			tip, action = "Aria Vaultwarden · démarrage…", "Arrêter le serveur"
		case moteur.Erreur:
			tip = "Aria Vaultwarden · erreur : " + s.Detail
		}
		if r := []rune(tip); len(r) > 120 { // limite de l'info-bulle Windows
			tip = string(r[:119]) + "…"
		}
		tray.SetIcon(icon)
		tray.SetTooltip(tip)
		toggle.SetLabel(action)
	}
	svc.moteur = moteur.New(p, j, func() { application.InvokeAsync(majTray) })
	configure := svc.moteur.Configure()

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Aria Vaultwarden",
		Width:     980,
		Height:    640,
		MinWidth:  420,
		MinHeight: 520,
		// Sans configuration, rien à servir : la fenêtre s'ouvre même en mode silencieux.
		Hidden:           silencieux && configure,
		BackgroundColour: application.NewRGB(3, 8, 18),
		URL:              "/",
		Windows:          application.WindowsWindow{Theme: application.Dark},
	})
	// Fermer la fenêtre la cache : le coffre continue de servir le téléphone.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	menu := app.NewMenu()
	menu.Add("Ouvrir Aria Vaultwarden").OnClick(func(*application.Context) { montrer() })
	menu.Add("Ouvrir le coffre dans le navigateur").OnClick(func(*application.Context) {
		if err := svc.OpenVault(); err != nil {
			j.Add(journal.Erreur, "Ouverture du coffre : %v", err)
		}
	})
	menu.AddSeparator()
	toggle = menu.Add("Démarrer le serveur").OnClick(func(*application.Context) {
		go func() {
			switch svc.moteur.Status().Etat {
			case moteur.Actif, moteur.Demarrage:
				svc.moteur.Stop()
			default:
				if err := svc.moteur.Start(); err != nil {
					j.Add(journal.Erreur, "%v", err)
					application.InvokeAsync(montrer)
				}
			}
		}()
	})
	menu.Add("Sauvegarder maintenant").OnClick(func(*application.Context) {
		go func() { _, _ = svc.Backup() }() // le résultat arrive dans le journal
	})
	menu.AddSeparator()
	menu.Add("Quitter (arrête le serveur)").OnClick(func(*application.Context) { app.Quit() })

	tray = app.SystemTray.New()
	tray.SetMenu(menu)
	tray.OnClick(montrer)
	tray.SetIcon(trayOff)
	tray.SetTooltip("Aria Vaultwarden")

	// Le coffre démarre une fois l'appli lancée (l'icône doit exister pour suivre son état).
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		majTray()
		go svc.boucleQuotidienne(ctx)
		if configure {
			go func() {
				if err := svc.moteur.Start(); err != nil {
					j.Add(journal.Erreur, "%v", err)
				}
			}()
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
