//go:build !windows

package config

// Hors Windows (tests uniquement) : pas de DPAPI, les secrets restent en clair.
func protect(plain []byte) ([]byte, error)  { return append([]byte(nil), plain...), nil }
func unprotect(data []byte) ([]byte, error) { return append([]byte(nil), data...), nil }
