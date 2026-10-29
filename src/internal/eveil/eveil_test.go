package eveil

import "testing"

func TestRegler(t *testing.T) {
	for _, v := range []bool{true, true, false, false, true, false} {
		if err := Regler(v); err != nil {
			t.Fatal(err)
		}
		if Actif() != v {
			t.Fatalf("Actif() = %v, attendu %v", Actif(), v)
		}
	}
}
