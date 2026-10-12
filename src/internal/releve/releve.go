// Package releve permet à une nouvelle version d'Aria de prendre la place de
// l'ancienne encore lancée, sans passer par « Quitter ».
//
// L'instance en cours décrit qui elle est dans instance.json (PID, date de
// création du processus, version) et attend un signal de fermeture. Une
// version plus récente qui démarre envoie ce signal, attend que l'ancienne
// s'arrête proprement (serveur arrêté, sauvegarde en cours terminée), puis
// démarre à sa place. Une version identique ou plus ancienne ne fait rien :
// l'instance unique de Wails se contente alors de montrer la fenêtre ouverte.
//
// Les versions 0.3 (Aria Vaultwarden) ne connaissent pas ce signal : elles
// sont reconnues à leur verrou d'instance unique et arrêtées directement,
// comme le ferait le Gestionnaire des tâches.
package releve

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	fichier       = "instance.json"
	attentePropre = 20 * time.Second // le serveur peut mettre quelques secondes à s'arrêter
	attenteForce  = 5 * time.Second
)

// Instance décrit l'Aria en cours d'exécution.
type Instance struct {
	PID      uint32 `json:"pid"`
	Creation int64  `json:"creation"` // date de création du processus : protège contre la réutilisation du PID
	Version  string `json:"version"`
	Exe      string `json:"exe"`
}

// Enregistrer décrit l'instance courante dans base\instance.json.
func Enregistrer(base, version string) error {
	creation, err := creationDe(uint32(os.Getpid()))
	if err != nil {
		return err
	}
	exe, _ := os.Executable()
	data, err := json.MarshalIndent(Instance{PID: uint32(os.Getpid()), Creation: creation, Version: version, Exe: exe}, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(base, fichier+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(base, fichier))
}

// Effacer supprime instance.json s'il décrit bien le processus courant.
func Effacer(base string) {
	if inst, err := lire(base); err == nil && inst.PID == uint32(os.Getpid()) {
		os.Remove(filepath.Join(base, fichier))
	}
}

func lire(base string) (Instance, error) {
	data, err := os.ReadFile(filepath.Join(base, fichier))
	if err != nil {
		return Instance{}, err
	}
	var inst Instance
	if err := json.Unmarshal(data, &inst); err != nil {
		return Instance{}, err
	}
	if inst.PID == 0 {
		return Instance{}, errors.New("instance.json sans PID")
	}
	return inst, nil
}

// Reprendre ferme une version plus ancienne d'Aria encore lancée pour lui
// succéder. À appeler avant application.New. Renvoie les messages à
// recopier dans le journal (le journal n'existe pas encore à ce stade).
func Reprendre(base, version string) []string {
	var msgs []string

	// Versions 0.3 : pas de fermeture propre possible, arrêt direct.
	for _, a := range anciennes03() {
		if err := terminer(a.pid, attenteForce); err != nil {
			msgs = append(msgs, fmt.Sprintf("Ancienne version %s (PID %d) impossible à fermer : %v", a.nom, a.pid, err))
			continue
		}
		msgs = append(msgs, fmt.Sprintf("Ancienne version %s (PID %d) fermée pour laisser la place", a.nom, a.pid))
	}

	// Versions 0.4 et suivantes : instance.json et signal de fermeture.
	inst, err := lire(base)
	if err != nil || inst.PID == uint32(os.Getpid()) || !PlusRecente(version, inst.Version) {
		return msgs
	}
	if !vivante(inst.PID, inst.Creation) {
		return msgs // instance.json périmé (plantage, redémarrage)
	}
	if err := demanderFermeture(); err == nil {
		if attendre(inst.PID, attentePropre) {
			return append(msgs, fmt.Sprintf("Version %s fermée proprement : la %s prend le relais", inst.Version, version))
		}
	}
	if err := terminer(inst.PID, attenteForce); err != nil {
		return append(msgs, fmt.Sprintf("Version %s (PID %d) impossible à fermer : %v", inst.Version, inst.PID, err))
	}
	return append(msgs, fmt.Sprintf("Version %s ne répondait pas : arrêtée pour laisser la place à la %s", inst.Version, version))
}

// PlusRecente indique si la version a est strictement plus récente que b
// (« 0.4.10 » > « 0.4.9 »). Une version non numérotée (« dev ») n'est jamais
// plus récente, et toute version numérotée l'emporte sur elle.
func PlusRecente(a, b string) bool {
	va, okA := parse(a)
	vb, okB := parse(b)
	switch {
	case !okA:
		return false
	case !okB:
		return true
	}
	for i := range va {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

type ancienne struct {
	pid uint32
	nom string
}
