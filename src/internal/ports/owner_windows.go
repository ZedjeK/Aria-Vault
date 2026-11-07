//go:build windows

// Retrouver le programme qui occupe un port, pour un message d'erreur utile
// (« le port 443 est occupé par caddy (PID 1234) »), via PowerShell.

package ports

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// owner tente de nommer le programme qui écoute sur ce port (ex. « caddy (PID 1234) »).
func owner(port int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	script := fmt.Sprintf(
		`$p = Get-NetTCPConnection -LocalPort %d -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty OwningProcess; `+
			`if ($p) { $proc = Get-Process -Id $p -ErrorAction SilentlyContinue; "$($proc.ProcessName) (PID $p)" }`, port)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if s == "" || strings.HasPrefix(s, "(PID") || strings.HasPrefix(s, " (PID") {
		return ""
	}
	return s
}
