package demarrage

import "testing"

func TestCommandeEtExe(t *testing.T) {
	exe := `C:\Users\Jérémy Z\Desktop\Aria Vault\Aria-Vault.exe`
	cmd := Commande(exe, "--silencieux")
	if cmd != `"`+exe+`" --silencieux` {
		t.Errorf("commande = %s", cmd)
	}
	if got := Exe(cmd); got != exe {
		t.Errorf("exe relu = %q", got)
	}
	if got := Exe(`C:\Aria\Aria-Vaultwarden.exe --silencieux`); got != `C:\Aria\Aria-Vaultwarden.exe` {
		t.Errorf("sans guillemets = %q", got)
	}
}
