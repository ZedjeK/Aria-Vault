// Package journal garde les derniers événements d'Aria pour l'interface et
// les recopie dans aria.log, pour comprendre après coup ce qui s'est passé
// pendant un démarrage silencieux.
package journal

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	maxEntries = 300
	maxFile    = 1 << 20 // au-delà, aria.log devient aria.log.1
)

// Niveaux d'une entrée.
const (
	Info   = "info"
	OK     = "ok"
	Erreur = "erreur"
)

// Entry est une ligne du journal.
type Entry struct {
	Heure  string `json:"heure"`
	Niveau string `json:"niveau"`
	Texte  string `json:"texte"`
}

// Journal est sûr pour un usage concurrent.
type Journal struct {
	mu      sync.Mutex
	entries []Entry
	path    string
}

// New crée un journal ; path vide = pas de fichier.
func New(path string) *Journal {
	j := &Journal{path: path}
	if path != "" {
		if st, err := os.Stat(path); err == nil && st.Size() > maxFile {
			_ = os.Rename(path, path+".1")
		}
	}
	return j
}

// Add ajoute une entrée.
func (j *Journal) Add(niveau, format string, a ...any) {
	now := time.Now()
	e := Entry{Heure: now.Format("15:04:05"), Niveau: niveau, Texte: fmt.Sprintf(format, a...)}

	j.mu.Lock()
	defer j.mu.Unlock()
	j.entries = append(j.entries, e)
	if len(j.entries) > maxEntries {
		j.entries = append([]Entry(nil), j.entries[len(j.entries)-maxEntries:]...)
	}
	if j.path != "" {
		if f, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			fmt.Fprintf(f, "%s [%s] %s\n", now.Format("2006-01-02 15:04:05"), niveau, e.Texte)
			f.Close()
		}
	}
}

// Logf est la forme attendue par vaultwarden.Server.Start.
func (j *Journal) Logf(format string, a ...any) { j.Add(Info, format, a...) }

// Entries renvoie une copie des entrées, de la plus ancienne à la plus récente.
func (j *Journal) Entries() []Entry {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]Entry(nil), j.entries...)
}

// Core renvoie un cœur zap qui recopie les messages de certmagic dans le journal.
func (j *Journal) Core() zapcore.Core { return &core{j: j} }

type core struct {
	j      *Journal
	fields []zapcore.Field
}

func (c *core) Enabled(l zapcore.Level) bool { return l >= zapcore.InfoLevel }

func (c *core) With(fields []zapcore.Field) zapcore.Core {
	return &core{j: c.j, fields: append(append([]zapcore.Field(nil), c.fields...), fields...)}
}

func (c *core) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

// Champs utiles à l'utilisateur ; le reste (identifiants internes…) est omis.
var kept = []string{"identifier", "identifiers", "error", "remaining", "challenge_type"}

func (c *core) Write(e zapcore.Entry, fields []zapcore.Field) error {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range append(append([]zapcore.Field(nil), c.fields...), fields...) {
		f.AddTo(enc)
	}
	var b strings.Builder
	b.WriteString(e.Message)
	for _, k := range kept {
		if v, ok := enc.Fields[k]; ok {
			fmt.Fprintf(&b, " · %s=%v", k, v)
		}
	}
	niveau := Info
	if e.Level >= zapcore.ErrorLevel {
		niveau = Erreur
	}
	c.j.Add(niveau, "certificat : %s", b.String())
	return nil
}

func (c *core) Sync() error { return nil }
