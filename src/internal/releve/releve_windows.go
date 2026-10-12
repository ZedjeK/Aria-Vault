//go:build windows

package releve

import (
	"errors"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// Signal de fermeture, propre à la session Windows de l'utilisateur.
	nomSignal = `Local\AriaVault-fermeture`
	// Verrou d'instance unique posé par Wails pour les versions 0.3
	// (UniqueID « io.github.zedjek.ariavaultwarden »).
	verrou03 = "wails-app-io.github.zedjek.ariavaultwarden-sim"
)

// EcouterFermeture attend le signal d'une version plus récente et appelle
// fermer une seule fois. À appeler par l'instance principale uniquement.
func EcouterFermeture(fermer func()) error {
	nom, err := windows.UTF16PtrFromString(nomSignal)
	if err != nil {
		return err
	}
	h, err := windows.CreateEvent(nil, 0, 0, nom) // réinitialisation automatique
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	go func() {
		if ev, _ := windows.WaitForSingleObject(h, windows.INFINITE); ev == windows.WAIT_OBJECT_0 {
			fermer()
		}
	}()
	return nil
}

func demanderFermeture() error {
	nom, err := windows.UTF16PtrFromString(nomSignal)
	if err != nil {
		return err
	}
	h, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, nom)
	if err != nil {
		return err // l'ancienne instance n'écoute pas (bloquée ?)
	}
	defer windows.CloseHandle(h)
	return windows.SetEvent(h)
}

func creationDe(pid uint32) (int64, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(h)
	var creation, sortie, noyau, utilisateur windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &sortie, &noyau, &utilisateur); err != nil {
		return 0, err
	}
	return creation.Nanoseconds(), nil
}

// vivante vérifie que le PID désigne toujours le même processus (même date de création).
func vivante(pid uint32, creation int64) bool {
	c, err := creationDe(pid)
	return err == nil && c == creation
}

// attendre renvoie true si le processus s'est terminé dans le délai.
func attendre(pid uint32, delai time.Duration) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return true // déjà terminé
	}
	defer windows.CloseHandle(h)
	ev, err := windows.WaitForSingleObject(h, uint32(delai/time.Millisecond))
	return err == nil && ev == windows.WAIT_OBJECT_0
}

// terminer arrête le processus et attend sa fin. Vaultwarden, rattaché à lui
// par un job « kill on close », s'arrête avec lui.
func terminer(pid uint32, delai time.Duration) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return nil // déjà terminé
		}
		return err
	}
	defer windows.CloseHandle(h)
	if err := windows.TerminateProcess(h, 1); err != nil {
		return err
	}
	if ev, err := windows.WaitForSingleObject(h, uint32(delai/time.Millisecond)); err != nil || ev != windows.WAIT_OBJECT_0 {
		return errors.New("le processus ne s'est pas terminé à temps")
	}
	return nil
}

// anciennes03 repère une version 0.3 lancée : son verrou d'instance unique
// existe, et son exe porte le nom d'origine (Aria-Vaultwarden*.exe).
func anciennes03() []ancienne {
	nom, err := windows.UTF16PtrFromString(verrou03)
	if err != nil {
		return nil
	}
	m, err := windows.OpenMutex(windows.SYNCHRONIZE, false, nom)
	if err != nil {
		return nil // aucune version 0.3 lancée
	}
	windows.CloseHandle(m)

	var session uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &session); err != nil {
		return nil
	}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)

	var out []ancienne
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		exe := windows.UTF16ToString(e.ExeFile[:])
		bas := strings.ToLower(exe)
		if e.ProcessID == uint32(os.Getpid()) || !strings.HasPrefix(bas, "aria-vaultwarden") || !strings.HasSuffix(bas, ".exe") {
			continue
		}
		var s uint32
		if windows.ProcessIdToSessionId(e.ProcessID, &s) != nil || s != session {
			continue // jamais la session d'un autre utilisateur
		}
		out = append(out, ancienne{pid: e.ProcessID, nom: exe})
	}
	return out
}
