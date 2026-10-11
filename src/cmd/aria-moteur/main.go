// aria-moteur : prototype en console du moteur d'Aria Vaultwarden (étape 1).
// Il lance Vaultwarden embarqué et le sert en HTTPS, sans interface graphique,
// pour valider la connexion depuis l'appli Bitwarden du téléphone.
//
//	aria-moteur configurer               enregistre le domaine et le jeton DuckDNS
//	aria-moteur [--inscriptions] [--test] démarre le coffre (Ctrl+C pour arrêter)
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"

	"aria-vaultwarden/internal/config"
	"aria-vaultwarden/internal/paths"
	"aria-vaultwarden/internal/tlsproxy"
	"aria-vaultwarden/internal/vaultwarden"
)

const portLocal = 8000

func main() {
	consoleUTF8()
	p, err := paths.Default()
	if err != nil {
		fail(err)
	}

	if len(os.Args) > 1 && os.Args[1] == "configurer" {
		if err := configurer(p); err != nil {
			fail(err)
		}
		return
	}

	fs := flag.NewFlagSet("aria-moteur", flag.ExitOnError)
	inscriptions := fs.Bool("inscriptions", false, "autorise la création de comptes le temps de cette session")
	test := fs.Bool("test", false, "certificat de test Let's Encrypt (refusé par le téléphone, sans limite de fréquence)")
	fs.Parse(os.Args[1:])

	if err := demarrer(p, *inscriptions, *test); err != nil {
		fail(err)
	}
}

func demarrer(p paths.Paths, inscriptions, test bool) error {
	cfg, err := config.Load(p.Config)
	if err != nil {
		return err
	}
	if err := cfg.Valider(); err != nil {
		return fmt.Errorf("%w\nLance d'abord : aria-moteur configurer", err)
	}
	jeton, _ := cfg.Jeton()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	step("Coffre : %s", p.Donnees)
	vw := vaultwarden.New(p.Moteur, p.Donnees)
	if err := vw.Start(vaultwarden.Options{Domaine: cfg.Domaine, Port: portLocal, Inscriptions: inscriptions}, info); err != nil {
		return err
	}
	defer vw.Stop()
	if inscriptions {
		info("Inscriptions OUVERTES pour cette session : relance sans --inscriptions une fois ton compte créé")
	}

	step("Certificat HTTPS pour %s", cfg.Domaine)
	if test {
		info("Mode test : certificat de l'autorité de test, le téléphone le refusera")
	}
	info("Première demande : jusqu'à quelques minutes, le temps que DuckDNS publie la preuve")
	proxy, err := tlsproxy.Start(ctx, tlsproxy.Options{
		Domaine:     cfg.Domaine,
		Jeton:       jeton,
		Email:       cfg.Email,
		Staging:     test,
		Certificats: p.Certificats,
		Ecoute:      ":443",
		Cible:       fmt.Sprintf("http://127.0.0.1:%d", portLocal),
		Logger:      consoleLogger(),
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil // Ctrl+C pendant la demande
		}
		return fmt.Errorf("HTTPS : %w", err)
	}
	defer proxy.Stop()

	fmt.Println()
	fmt.Printf("  Coffre prêt : https://%s\n", cfg.Domaine)
	fmt.Println("  Ctrl+C pour arrêter.")
	fmt.Println()

	select {
	case <-ctx.Done():
		step("Arrêt demandé")
	case <-vw.Done():
		return fmt.Errorf("Vaultwarden s'est arrêté de lui-même : voir %s\\vaultwarden.log", p.Donnees)
	case err := <-proxy.Done():
		return fmt.Errorf("le serveur HTTPS s'est arrêté : %v", err)
	}
	return nil
}

func configurer(p paths.Paths) error {
	cfg, err := config.Load(p.Config)
	if err != nil {
		return err
	}
	in := bufio.NewReader(os.Stdin)
	ask := func(question, current string) string {
		if current != "" {
			fmt.Printf("%s [%s] : ", question, current)
		} else {
			fmt.Printf("%s : ", question)
		}
		line, _ := in.ReadString('\n')
		if line = strings.TrimSpace(line); line == "" {
			return current
		}
		return line
	}

	d, err := config.NormaliserDomaine(ask("Sous-domaine DuckDNS (ex. vault.exemple.duckdns.org)", cfg.Domaine))
	if err != nil {
		return err
	}
	cfg.Domaine = d

	if len(cfg.JetonChiffre) > 0 {
		fmt.Print("Jeton DuckDNS [Entrée = garder l'actuel] : ")
	} else {
		fmt.Print("Jeton DuckDNS (affiché en haut de duckdns.org) : ")
	}
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return err
	}
	if j := strings.TrimSpace(string(raw)); j != "" {
		if err := cfg.SetJeton(j); err != nil {
			return err
		}
	}

	cfg.Email = ask("E-mail pour les avis Let's Encrypt (facultatif)", cfg.Email)

	if err := cfg.Valider(); err != nil {
		return err
	}
	if err := cfg.Save(p.Config); err != nil {
		return err
	}
	step("Configuration enregistrée (jeton chiffré pour ton compte Windows) : %s", p.Config)
	return nil
}

func consoleLogger() *zap.Logger {
	enc := zapcore.EncoderConfig{
		TimeKey:        "t",
		LevelKey:       "",
		MessageKey:     "msg",
		NameKey:        "",
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(enc), zapcore.Lock(os.Stderr), zapcore.InfoLevel)
	return zap.New(core)
}

func step(format string, a ...any) { fmt.Printf("\n==> "+format+"\n", a...) }
func info(format string, a ...any) { fmt.Printf("    "+format+"\n", a...) }

func fail(err error) {
	fmt.Fprintf(os.Stderr, "\n[X] %v\n", err)
	os.Exit(1)
}
