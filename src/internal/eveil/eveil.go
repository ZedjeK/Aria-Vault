// Package eveil empêche Windows de se mettre en veille tout seul pendant que
// le coffre est en ligne (ES_SYSTEM_REQUIRED). L'écran peut toujours s'éteindre,
// et une mise en veille demandée (menu Démarrer, capot fermé) reste possible.
package eveil

import "sync"

var (
	mu      sync.Mutex
	liberer func() // non nil tant que la veille est empêchée
)

// Regler empêche (true) ou autorise de nouveau (false) la mise en veille.
// Les appels répétés avec la même valeur ne font rien.
func Regler(empecher bool) error {
	mu.Lock()
	defer mu.Unlock()
	switch {
	case empecher && liberer == nil:
		l, err := garder()
		if err != nil {
			return err
		}
		liberer = l
	case !empecher && liberer != nil:
		liberer()
		liberer = nil
	}
	return nil
}

// Actif indique si la veille est actuellement empêchée.
func Actif() bool {
	mu.Lock()
	defer mu.Unlock()
	return liberer != nil
}
