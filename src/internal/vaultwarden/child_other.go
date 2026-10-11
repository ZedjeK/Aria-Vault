//go:build !windows

package vaultwarden

import "os/exec"

func prepareChild(*exec.Cmd)       {}
func bindToParent(*exec.Cmd) error { return nil }
