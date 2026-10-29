// Package chiffre protège des données par une phrase de passe :
// scrypt pour dériver la clé, AES-256-GCM pour chiffrer et authentifier.
//
// Deux formes partagent le même en-tête :
//
//		magie (8 o) | log2(N) | r | p | sel (16 o) | …
//
//	  - Sceller / Ouvrir : un petit contenu d'un seul bloc (fichier .ariavault),
//	    suivi d'un nonce de 12 o puis du chiffré.
//	  - Chiffrer / Dechiffrer : un flux de taille quelconque (sauvegardes), suivi
//	    d'un préfixe de nonce de 7 o puis de blocs de 64 Kio chiffrés chacun avec
//	    le nonce préfixe | compteur (4 o) | dernier (1 o). Un bloc retiré, déplacé
//	    ou un fichier tronqué fait échouer le déchiffrement.
//
// L'en-tête est authentifié avec chaque bloc : le moindre octet modifié est détecté.
package chiffre

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	PhraseMin = 8

	// Paramètres scrypt recommandés pour un usage interactif (~0,3 s, 128 Mo).
	logN, paramR, paramP = 17, 8, 1

	tailleBase = 8 + 3 + 16
	tailleBloc = 64 << 10
	tailleTag  = 16
)

// ErrPhrase signale une phrase de passe incorrecte (ou des données altérées).
var ErrPhrase = errors.New("phrase de passe incorrecte (ou fichier modifié)")

// ErrFormat signale des données qui ne sont pas au format attendu.
var ErrFormat = errors.New("format de fichier inconnu")

// VerifierPhrase refuse une phrase trop courte.
func VerifierPhrase(phrase string) error {
	if len([]rune(phrase)) < PhraseMin {
		return fmt.Errorf("la phrase de passe doit faire au moins %d caractères", PhraseMin)
	}
	return nil
}

// entete fabrique un en-tête neuf : magie, paramètres, sel, puis n octets aléatoires.
func entete(magie []byte, n int) ([]byte, error) {
	if len(magie) != 8 {
		return nil, errors.New("magie de 8 octets attendue")
	}
	h := make([]byte, 0, tailleBase+n)
	h = append(h, magie...)
	h = append(h, logN, paramR, paramP)
	alea := make([]byte, 16+n)
	if _, err := rand.Read(alea); err != nil {
		return nil, err
	}
	return append(h, alea...), nil
}

// aeadDe dérive la clé de l'en-tête. Les paramètres lus sont bornés : un
// fichier piégé ne doit pas faire exploser la mémoire.
func aeadDe(phrase string, h []byte) (cipher.AEAD, error) {
	n, r, p := int(h[8]), int(h[9]), int(h[10])
	if n < 14 || n > 20 || r < 1 || r > 16 || p < 1 || p > 4 {
		return nil, errors.New("paramètres de chiffrement invalides dans le fichier")
	}
	cle, err := scrypt.Key([]byte(phrase), h[11:27], 1<<n, r, p, 32)
	if err != nil {
		return nil, err
	}
	bloc, err := aes.NewCipher(cle)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(bloc)
}

// EstChiffre indique si les données commencent par la magie donnée.
func EstChiffre(debut, magie []byte) bool {
	return len(debut) >= len(magie) && bytes.Equal(debut[:len(magie)], magie)
}

// Sceller chiffre un petit contenu en un seul bloc.
func Sceller(phrase string, magie, clair []byte) ([]byte, error) {
	if err := VerifierPhrase(phrase); err != nil {
		return nil, err
	}
	h, err := entete(magie, 12)
	if err != nil {
		return nil, err
	}
	gcm, err := aeadDe(phrase, h)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(h, h[tailleBase:], clair, h), nil
}

// Ouvrir déchiffre un contenu produit par Sceller.
func Ouvrir(phrase string, magie, data []byte) ([]byte, error) {
	const taille = tailleBase + 12
	if len(data) <= taille || !EstChiffre(data, magie) {
		return nil, ErrFormat
	}
	if phrase == "" {
		return nil, errors.New("la phrase de passe est obligatoire")
	}
	h := data[:taille]
	gcm, err := aeadDe(phrase, h)
	if err != nil {
		return nil, err
	}
	clair, err := gcm.Open(nil, h[tailleBase:], data[taille:], h)
	if err != nil {
		return nil, ErrPhrase
	}
	return clair, nil
}

func nonceBloc(prefixe []byte, compteur uint32, dernier bool) []byte {
	n := make([]byte, 12)
	copy(n, prefixe)
	binary.BigEndian.PutUint32(n[7:11], compteur)
	if dernier {
		n[11] = 1
	}
	return n
}

// Chiffrer chiffre tout src dans dst.
func Chiffrer(dst io.Writer, src io.Reader, phrase string, magie []byte) error {
	if err := VerifierPhrase(phrase); err != nil {
		return err
	}
	h, err := entete(magie, 7)
	if err != nil {
		return err
	}
	gcm, err := aeadDe(phrase, h)
	if err != nil {
		return err
	}
	if _, err := dst.Write(h); err != nil {
		return err
	}
	prefixe := h[tailleBase:]
	br := bufio.NewReaderSize(src, tailleBloc+1)
	buf := make([]byte, tailleBloc)
	sortie := make([]byte, 0, tailleBloc+tailleTag)
	for compteur := uint32(0); ; compteur++ {
		if compteur == ^uint32(0) {
			return errors.New("fichier trop volumineux")
		}
		n, err := io.ReadFull(br, buf)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			return err
		}
		dernier := n < tailleBloc
		if !dernier {
			if _, err := br.Peek(1); errors.Is(err, io.EOF) {
				dernier = true
			} else if err != nil {
				return err
			}
		}
		sortie = gcm.Seal(sortie[:0], nonceBloc(prefixe, compteur, dernier), buf[:n], h)
		if _, err := dst.Write(sortie); err != nil {
			return err
		}
		if dernier {
			return nil
		}
	}
}

// Dechiffrer déchiffre src dans dst. En cas d'erreur, ce qui a déjà été écrit
// dans dst ne doit pas être utilisé.
func Dechiffrer(dst io.Writer, src io.Reader, phrase string, magie []byte) error {
	const taille = tailleBase + 7
	h := make([]byte, taille)
	if _, err := io.ReadFull(src, h); err != nil || !EstChiffre(h, magie) {
		return ErrFormat
	}
	if phrase == "" {
		return errors.New("la phrase de passe est obligatoire")
	}
	gcm, err := aeadDe(phrase, h)
	if err != nil {
		return err
	}
	prefixe := h[tailleBase:]
	br := bufio.NewReaderSize(src, tailleBloc+tailleTag+1)
	buf := make([]byte, tailleBloc+tailleTag)
	clair := make([]byte, 0, tailleBloc)
	for compteur := uint32(0); ; compteur++ {
		n, err := io.ReadFull(br, buf)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			return err
		}
		if n < tailleTag {
			return errors.New("fichier tronqué")
		}
		dernier := n < len(buf)
		if !dernier {
			if _, err := br.Peek(1); errors.Is(err, io.EOF) {
				dernier = true
			} else if err != nil {
				return err
			}
		}
		clair, err = gcm.Open(clair[:0], nonceBloc(prefixe, compteur, dernier), buf[:n], h)
		if err != nil {
			if compteur == 0 {
				return ErrPhrase
			}
			return errors.New("fichier altéré ou tronqué")
		}
		if _, err := dst.Write(clair); err != nil {
			return err
		}
		if dernier {
			return nil
		}
	}
}
