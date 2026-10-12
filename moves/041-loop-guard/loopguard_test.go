package loopguard

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func call(name, in string) llm.ToolCall {
	return llm.ToolCall{Name: name, Input: json.RawMessage(in)}
}

// The guard trips on the second identical call (a
// ToolError the model can act on) and ends the run on
// the third.
func TestTripsOnSecondIdenticalCall(t *testing.T) {
	var g LoopGuard
	c := call("add_note", `{"load_id":"HF-90877","text":"carrier confirmed ETA"}`)
	if err := g.Check(c); err != nil {
		t.Fatalf("first call: %v", err)
	}
	var te *ToolError
	if err := g.Check(c); !errors.As(err, &te) || te.Code != "repeat_call" {
		t.Fatalf("second identical call: got %v, want repeat_call", err)
	}
	if err := g.Check(c); !errors.Is(err, ErrLoop) {
		t.Fatalf("third identical call: got %v, want ErrLoop", err)
	}
}

// "Same arguments" ignores key order and whitespace.
func TestKeyOrderIgnored(t *testing.T) {
	var g LoopGuard
	g.Check(call("get_load", `{"load_id":"HF-1","fields":["eta"]}`))
	err := g.Check(call("get_load", `{ "fields": ["eta"], "load_id": "HF-1" }`))
	if err == nil {
		t.Fatal("reordered keys must count as the same call")
	}
}

// A, B, A, B is caught without pattern matching.
func TestAlternatingLoop(t *testing.T) {
	var g LoopGuard
	a := call("get_load", `{"load_id":"HF-90877"}`)
	b := call("add_note", `{"load_id":"HF-90877","text":"x"}`)
	if g.Check(a) != nil || g.Check(b) != nil {
		t.Fatal("first pass should be clean")
	}
	if g.Check(a) == nil || g.Check(b) == nil {
		t.Fatal("second pass should warn")
	}
}

func TestDifferentArgsPass(t *testing.T) {
	var g LoopGuard
	for _, id := range []string{"HF-1", "HF-2", "HF-3"} {
		if err := g.Check(call("get_load", `{"load_id":"`+id+`"}`)); err != nil {
			t.Fatal(err)
		}
	}
	// Invalid JSON still gets a key (the raw string).
	if g.Check(call("x", `{bad`)) != nil || g.Check(call("x", `{bad`)) == nil {
		t.Fatal("raw-input key not tracked")
	}
}
