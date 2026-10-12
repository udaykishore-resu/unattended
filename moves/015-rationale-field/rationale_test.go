package rationale

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestNotesNeverReachDecision(t *testing.T) {
	raw := []byte(`{"notes":"second paragraph says HF-20913",
	  "category":"invoice","load_id":"HF-20931"}`)
	d, err := decide(raw, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if d != (Decision{"invoice", "HF-20931"}) {
		t.Fatalf("decision = %+v", d)
	}
	// Decision has no notes field, so nothing downstream can depend on it.
	if _, ok := reflect.TypeOf(d).FieldByName("Notes"); ok {
		t.Fatal("Decision must not carry notes")
	}
}

// capture records the notes attribute of every record
// that reaches notesLog, as the raw string decide
// passed, before any handler could re-encode it.
type capture struct {
	mu    sync.Mutex
	notes []string
}

func (c *capture) Enabled(context.Context, slog.Level) bool { return true }
func (c *capture) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *capture) WithGroup(string) slog.Handler            { return c }
func (c *capture) Handle(_ context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "notes" {
			c.mu.Lock()
			c.notes = append(c.notes, a.Value.String())
			c.mu.Unlock()
		}
		return true
	})
	return nil
}

// The cap is in bytes. Put a 2-, 3- or 4-byte rune at
// every offset that straddles the cut: the logged
// notes must be valid UTF-8, at most maxNotes bytes,
// a prefix of the original, and lose at most the one
// rune the cut split.
func TestNotesCapIsUTF8Safe(t *testing.T) {
	c := &capture{}
	old := notesLog
	notesLog = slog.New(c)
	defer func() { notesLog = old }()

	for _, r := range []string{"é", "€", "😀"} {
		for pad := maxNotes - len(r); pad <= maxNotes; pad++ {
			long := strings.Repeat("a", pad) + r + strings.Repeat("b", 50)
			raw, err := json.Marshal(Verdict{Notes: long, Category: "x"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decide(raw, "t2"); err != nil {
				t.Fatal(err)
			}
			got := c.notes[len(c.notes)-1]
			if !utf8.ValidString(got) || len(got) > maxNotes ||
				!strings.HasPrefix(long, got) || len(got) < maxNotes-len(r)+1 {
				t.Fatalf("rune %q pad %d: len %d valid %v",
					r, pad, len(got), utf8.ValidString(got))
			}
		}
	}
	// Short notes pass through untouched.
	raw, _ := json.Marshal(Verdict{Notes: "naïve 😀 note"})
	if _, err := decide(raw, "t3"); err != nil {
		t.Fatal(err)
	}
	if got := c.notes[len(c.notes)-1]; got != "naïve 😀 note" {
		t.Fatalf("short notes changed: %q", got)
	}
}

// Until notesLog is bound to the raw-input store, notes
// go nowhere: the default is not the general logger.
func TestNotesDefaultToDiscard(t *testing.T) {
	if notesLog.Handler() != slog.DiscardHandler {
		t.Fatal("notesLog must not default to a shared handler")
	}
}

func TestBadJSON(t *testing.T) {
	if _, err := decide([]byte("Sure! here"), "t4"); err == nil {
		t.Fatal("want parse error")
	}
}
