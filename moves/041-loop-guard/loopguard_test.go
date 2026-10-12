package loopguard

import (
	"encoding/json"
	"errors"
	"sync"
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

// Numbers keep their exact text: two IDs that differ
// past float64 precision are different calls.
func TestBigNumbersStayDistinct(t *testing.T) {
	var g LoopGuard
	if g.Check(call("get_load", `{"pro":12345678901234567}`)) != nil ||
		g.Check(call("get_load", `{"pro":12345678901234568}`)) != nil {
		t.Fatal("distinct big IDs were treated as a repeat")
	}
}

// The key never fails for any input the model can
// send: valid JSON, invalid JSON, invalid UTF-8,
// empty input. If it ever did, Check would return
// that error and the run would end (fail closed).
func TestKeyNeverFailsOnModelInput(t *testing.T) {
	for _, in := range []string{`{"a":1}`, `{bad`, "\xff\xfe", ``, `[1,2]`, `"s"`, `1e400`} {
		if _, err := key(call("x", in)); err != nil {
			t.Errorf("key(%q): %v", in, err)
		}
	}
}

// One guard per run may be shared by goroutines that
// run a turn's tool calls in parallel: Check is
// locked, and exactly one of N identical concurrent
// calls passes. Run with -race.
func TestConcurrentChecks(t *testing.T) {
	var g LoopGuard
	c := call("get_load", `{"load_id":"HF-1"}`)
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := map[string]int{}
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := g.Check(c)
			var te *ToolError
			k := "ok"
			switch {
			case errors.Is(err, ErrLoop):
				k = "loop"
			case errors.As(err, &te):
				k = "repeat"
			}
			mu.Lock()
			results[k]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if results["ok"] != 1 || results["repeat"] != 1 || results["loop"] != 1 {
		t.Fatalf("results = %v", results)
	}
}
