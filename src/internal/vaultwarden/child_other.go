//go:build !windows

package vaultwarden

import "os/exec"

// prepareChild : hors Windows, rien à préparer.
func prepareChild(*exec.Cmd) {}

// bindToParent : hors Windows, pas de job.
func bindToParent(*exec.Cmd) error { return nil }
