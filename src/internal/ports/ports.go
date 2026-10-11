// Package ports vérifie qu'un port TCP est libre et, sous Windows,
// nomme le programme qui l'occupe.
package ports

import (
	"fmt"
	"net"
	"strconv"
)

// Check renvoie nil si le port est libre, sinon une erreur qui nomme son occupant.
func Check(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return Describe(addr)
	}
	l.Close()
	return nil
}

// Describe renvoie une erreur lisible pour un port impossible à ouvrir.
func Describe(addr string) error {
	_, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)
	if o := owner(port); o != "" {
		return fmt.Errorf("le port %d est déjà utilisé par %s", port, o)
	}
	return fmt.Errorf("le port %d est déjà utilisé par un autre programme", port)
}
