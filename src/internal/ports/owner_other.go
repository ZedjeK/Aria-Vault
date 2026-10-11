//go:build !windows

package ports

func owner(int) string { return "" }
