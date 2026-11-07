//go:build !windows

package ports

// owner : hors Windows, on ne cherche pas le programme qui occupe le port.
func owner(int) string { return "" }
