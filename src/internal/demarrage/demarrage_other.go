//go:build !windows

package demarrage

// Aligner est sans effet hors Windows.
func Aligner(string, ...string) (string, error) { return "", nil }
