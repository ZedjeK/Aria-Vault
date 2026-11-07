package chiffre

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
)

var magie = []byte("TESTMAGI")

func TestScellerOuvrir(t *testing.T) {
	data, err := Sceller("une bonne phrase", magie, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("secret")) {
		t.Fatal("contenu en clair")
	}
	clair, err := Ouvrir("une bonne phrase", magie, data)
	if err != nil || string(clair) != "secret" {
		t.Fatalf("%q, %v", clair, err)
	}
	if _, err := Ouvrir("mauvaise phrase", magie, data); !errors.Is(err, ErrPhrase) {
		t.Errorf("mauvaise phrase : %v", err)
	}
	data[9] ^= 1 // paramètre r modifié : refusé
	if _, err := Ouvrir("une bonne phrase", magie, data); err == nil {
		t.Error("en-tête modifié accepté")
	}
	if _, err := Sceller("court", magie, nil); err == nil {
		t.Error("phrase trop courte acceptée")
	}
}

func TestFlux(t *testing.T) {
	for _, taille := range []int{0, 1, tailleBloc - 1, tailleBloc, tailleBloc + 1, 3*tailleBloc + 17} {
		clair := make([]byte, taille)
		rand.Read(clair)
		var c bytes.Buffer
		if err := Chiffrer(&c, bytes.NewReader(clair), "une bonne phrase", magie); err != nil {
			t.Fatal(err)
		}
		var d bytes.Buffer
		if err := Dechiffrer(&d, bytes.NewReader(c.Bytes()), "une bonne phrase", magie); err != nil {
			t.Fatalf("taille %d : %v", taille, err)
		}
		if !bytes.Equal(d.Bytes(), clair) {
			t.Fatalf("taille %d : contenu différent", taille)
		}
		if err := Dechiffrer(&bytes.Buffer{}, bytes.NewReader(c.Bytes()), "mauvaise phrase", magie); !errors.Is(err, ErrPhrase) {
			t.Errorf("taille %d, mauvaise phrase : %v", taille, err)
		}
		// Tronqué au bloc près : détecté grâce à l'indicateur de dernier bloc.
		if taille > tailleBloc {
			coupe := c.Bytes()[:tailleBase+7+tailleBloc+tailleTag]
			if err := Dechiffrer(&bytes.Buffer{}, bytes.NewReader(coupe), "une bonne phrase", magie); err == nil {
				t.Errorf("taille %d : fichier tronqué accepté", taille)
			}
		}
		if taille > 0 {
			abime := append([]byte(nil), c.Bytes()...)
			abime[len(abime)-1] ^= 1
			if err := Dechiffrer(&bytes.Buffer{}, bytes.NewReader(abime), "une bonne phrase", magie); err == nil {
				t.Errorf("taille %d : fichier modifié accepté", taille)
			}
		}
	}
}

func TestLongueurPhrase(t *testing.T) {
	const onze, douze = "onze lettre", "douze lettre"
	if err := VerifierPhrase(onze); err == nil {
		t.Errorf("nouvelle phrase de %d caractères acceptée", len(onze))
	}
	if err := VerifierPhrase(douze); err != nil {
		t.Errorf("phrase de %d caractères refusée : %v", len(douze), err)
	}
	// Les caractères comptent, pas les octets : 12 lettres accentuées passent.
	if err := VerifierPhrase("ééééééééééé"); err == nil {
		t.Error("11 caractères accentués (22 octets) acceptés")
	}
	if err := VerifierPhrase("éééééééééééé"); err != nil {
		t.Errorf("12 caractères accentués refusés : %v", err)
	}
	if _, err := Sceller(onze, magie, []byte("x")); err == nil {
		t.Error("export scellé avec une phrase trop courte")
	}
	if err := VerifierPhraseEnregistree("sept ca"); err == nil {
		t.Error("phrase enregistrée sous l'ancien minimum acceptée")
	}
}

// Une phrase de 8 caractères, valable avant le passage à 12 : les fichiers
// déjà faits avec elle doivent rester lisibles, et une phrase des sauvegardes
// déjà enregistrée doit continuer de chiffrer.
func TestAnciennePhrase(t *testing.T) {
	const ancienne = "huit car"

	var c bytes.Buffer
	if err := Chiffrer(&c, bytes.NewReader([]byte("coffre")), ancienne, magie); err != nil {
		t.Fatalf("phrase enregistrée de 8 caractères refusée : %v", err)
	}
	var d bytes.Buffer
	if err := Dechiffrer(&d, bytes.NewReader(c.Bytes()), ancienne, magie); err != nil || d.String() != "coffre" {
		t.Fatalf("ancienne sauvegarde illisible : %q, %v", d.String(), err)
	}

	// Fichier scellé comme le faisait Sceller avec l'ancien minimum.
	h, err := entete(magie, 12)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := aeadDe(ancienne, h)
	if err != nil {
		t.Fatal(err)
	}
	scelle := gcm.Seal(h, h[tailleBase:], []byte("réglages"), h)
	clair, err := Ouvrir(ancienne, magie, scelle)
	if err != nil || string(clair) != "réglages" {
		t.Fatalf("ancien export illisible : %q, %v", clair, err)
	}
}
