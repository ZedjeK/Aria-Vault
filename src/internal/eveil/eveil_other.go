//go:build !windows

package eveil

// Hors Windows (tests) : rien à faire.
func garder() (func(), error) { return func() {}, nil }
