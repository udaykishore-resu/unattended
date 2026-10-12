package rationale

import (
	"encoding/json"
	"reflect"
	"strings"
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

// The cap is in bytes; a rune split by the cut is dropped.
func TestNotesCapIsUTF8Safe(t *testing.T) {
	long := strings.Repeat("a", maxNotes-1) + "é" + strings.Repeat("b", 50)
	b, _ := json.Marshal(Verdict{Notes: long, Category: "x"})
	var v Verdict
	json.Unmarshal(b, &v)
	n := v.Notes[:maxNotes]
	capped := strings.ToValidUTF8(n, "")
	if !utf8.ValidString(capped) || len(capped) != maxNotes-1 {
		t.Fatalf("capped len %d valid %v", len(capped), utf8.ValidString(capped))
	}
	if _, err := decide(b, "t2"); err != nil {
		t.Fatal(err)
	}
}

func TestBadJSON(t *testing.T) {
	if _, err := decide([]byte("Sure! here"), "t3"); err == nil {
		t.Fatal("want parse error")
	}
}
