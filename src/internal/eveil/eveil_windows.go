//go:build windows

package eveil

import (
	"errors"
	"runtime"

	"golang.org/x/sys/windows"
)

const (
	esSystemRequired = 0x00000001
	esContinuous     = 0x80000000
)

var procSetThreadExecutionState = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetThreadExecutionState")

// garder : l'état d'exécution est propre à un thread système. Une goroutine
// verrouillée sur son thread le tient jusqu'à la libération.
func garder() (func(), error) {
	pret := make(chan error)
	fin := make(chan struct{})
	termine := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(termine)
		if r, _, err := procSetThreadExecutionState.Call(esContinuous | esSystemRequired); r == 0 {
			pret <- errors.Join(errors.New("SetThreadExecutionState refusé"), err)
			return
		}
		pret <- nil
		<-fin
		procSetThreadExecutionState.Call(esContinuous)
	}()
	if err := <-pret; err != nil {
		return nil, err
	}
	return func() {
		close(fin)
		<-termine
	}, nil
}
