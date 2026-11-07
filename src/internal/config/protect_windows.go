//go:build windows

// Chiffrement des secrets avec DPAPI (CryptProtectData), lié au compte
// Windows : un autre compte, ou un autre PC, ne peut pas les déchiffrer.
// C'est pour ça que l'export .ariavault existe.

package config

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Chiffrement DPAPI lié au compte Windows courant.
const dpapiName = "Aria Vault"

// protect chiffre plain pour le compte Windows courant.
func protect(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, nil
	}
	name, err := windows.UTF16PtrFromString(dpapiName)
	if err != nil {
		return nil, err
	}
	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, name, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

// unprotect déchiffre un secret produit par protect.
func unprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
