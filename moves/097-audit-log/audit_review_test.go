package audit

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// table stands in for the insert-only Postgres table:
// each Write is one atomic row insert.
type table struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	fail int // fail the next n writes
	torn bool
}

func (t *table) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fail > 0 {
		t.fail--
		if t.torn { // half a line reaches the file
			t.buf.Write(p[:len(p)/2])
		}
		return 0, errors.New("disk full")
	}
	return t.buf.Write(p)
}

func (t *table) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

func entry(i int) Entry {
	return Entry{Class: "invoice-7y", RunID: fmt.Sprint("r", i),
		Agent: "billing-agent:1.4.2", Tool: "set_amount",
		Input: fmt.Sprint("invoice:", 40000+i), Result: "ok"}
}

// Verify recomputes the hash with the same sum Write
// used, from the parsed row. A store that reorders keys
// or whitespace (jsonb does both) still verifies.
func TestVerifyRecomputesCanonicalHash(t *testing.T) {
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	for i := range 3 {
		if err := l.Write(entry(i)); err != nil {
			t.Fatal(err)
		}
	}
	var reordered strings.Builder
	for _, ln := range strings.Split(strings.TrimSpace(tb.String()), "\n") {
		var m map[string]any
		json.Unmarshal([]byte(ln), &m)
		b, _ := json.MarshalIndent(m, "", " ") // sorted keys, new spacing
		reordered.Write(bytes.ReplaceAll(b, []byte("\n"), nil))
		reordered.WriteByte('\n')
	}
	if err := Verify(strings.NewReader(reordered.String()), l.last); err != nil {
		t.Fatal(err)
	}
}

// Many goroutines in one pod: mu keeps one chain per
// segment, and file order matches chain order.
func TestConcurrentWriters(t *testing.T) {
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	var wg sync.WaitGroup
	for g := range 32 {
		wg.Go(func() {
			for i := range 25 {
				if err := l.Write(entry(g*100 + i)); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	if n := strings.Count(tb.String(), "\n"); n != 800 {
		t.Fatalf("%d rows", n)
	}
	if err := Verify(strings.NewReader(tb.String()), l.last); err != nil {
		t.Fatal(err)
	}
}

// Two replicas write to one table at once. The pod name
// in the segment key gives each its own chain.
func TestReplicasDoNotFork(t *testing.T) {
	var tb table
	a, b := NewLog(&tb, "pod-a", nil), NewLog(&tb, "pod-b", nil)
	var wg sync.WaitGroup
	for _, l := range []*Log{a, b} {
		wg.Go(func() {
			for i := range 50 {
				l.Write(entry(i))
			}
		})
	}
	wg.Wait()
	cps := map[string]string{}
	for _, l := range []*Log{a, b} {
		for s, h := range l.last {
			cps[s] = h
		}
	}
	if len(cps) != 2 {
		t.Fatalf("segments = %v", cps)
	}
	if err := Verify(strings.NewReader(tb.String()), cps); err != nil {
		t.Fatal(err)
	}
}

// A restarted pod loads each open segment's tail; one
// that starts empty begins a second chain and fails.
func TestRestartLoadsTails(t *testing.T) {
	for _, load := range []bool{true, false} {
		var tb table
		l := NewLog(&tb, "pod-a", nil)
		l.Write(entry(1))
		l.Write(entry(2))
		var tails map[string]string
		if load {
			var err error
			if tails, err = Tails(strings.NewReader(tb.String())); err != nil {
				t.Fatal(err)
			}
		}
		l2 := NewLog(&tb, "pod-a", tails) // the pod restarted
		l2.Write(entry(3))
		err := Verify(strings.NewReader(tb.String()), l2.last)
		if load != (err == nil) {
			t.Fatalf("load=%v: err = %v", load, err)
		}
	}
}

// A failed write returns an error and does not advance
// the chain, so the next row links to the last stored one.
func TestFailedWriteDoesNotAdvanceChain(t *testing.T) {
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	l.Write(entry(1))
	tb.fail = 1
	if err := l.Write(entry(2)); err == nil {
		t.Fatal("want the write error")
	}
	l.Write(entry(3))
	if err := Verify(strings.NewReader(tb.String()), l.last); err != nil {
		t.Fatal(err)
	}
}

// A torn write (half a row, then an error) is not
// silent: the verifier stops at that line.
func TestTornWriteIsDetected(t *testing.T) {
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	l.Write(entry(1))
	tb.fail, tb.torn = 1, true
	l.Write(entry(2))
	l.Write(entry(3))
	if err := Verify(strings.NewReader(tb.String()), l.last); err == nil ||
		!strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err = %v", err)
	}
}

// Someone with write access edits a row and rewrites
// every hash after it. The chain alone verifies; the
// signed checkpoint does not, and the forger cannot
// sign a new one.
func TestRewrittenChainNeedsSignedCheckpoint(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	for i := range 4 {
		l.Write(entry(i))
	}
	var cps []Checkpoint
	for seg, h := range l.last { // day close
		cps = append(cps, Sign(priv, seg, h))
	}

	// The forger zeroes invoice 40001 and re-chains.
	var forged strings.Builder
	prev := ""
	for _, ln := range strings.Split(strings.TrimSpace(tb.String()), "\n") {
		var e Entry
		json.Unmarshal([]byte(ln), &e)
		if e.Input == "invoice:40001" {
			e.Result = "set to 0"
		}
		e.Prev = prev
		e.Hash, _ = e.sum()
		prev = e.Hash
		b, _ := json.Marshal(e)
		forged.Write(append(b, '\n'))
	}
	seg := cps[0].Seg
	self := map[string]string{seg: prev}
	if err := Verify(strings.NewReader(forged.String()), self); err != nil {
		t.Fatalf("a re-chained log verifies against its own tail: %v", err)
	}
	good, err := Trusted(pub, cps)
	if err != nil {
		t.Fatal(err)
	}
	if Verify(strings.NewReader(forged.String()), good) == nil {
		t.Fatal("forgery passed the signed checkpoint")
	}
	_, otherKey, _ := ed25519.GenerateKey(nil)
	if _, err := Trusted(pub, []Checkpoint{Sign(otherKey, seg, prev)}); err == nil {
		t.Fatal("forged checkpoint accepted")
	}
}

// A whole live segment deleted is reported, not skipped.
func TestDeletedSegmentIsReported(t *testing.T) {
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	l.Write(entry(1))
	if err := Verify(strings.NewReader(""), l.last); err == nil ||
		!strings.Contains(err.Error(), "rows missing") {
		t.Fatalf("err = %v", err)
	}
}

// Breaks when: a failed audit write must block the action.
func TestAuditFailureBlocksAction(t *testing.T) {
	var tb table
	l := NewLog(&tb, "pod-a", nil)
	tb.fail = 1
	ran := 0
	err := l.Do(entry(1), func() error { ran++; return nil })
	if err == nil || ran != 0 {
		t.Fatalf("err = %v, ran = %d", err, ran)
	}
	// Healthy log: pending row, action, outcome row.
	if err := l.Do(entry(2), func() error { ran++; return nil }); err != nil || ran != 1 {
		t.Fatalf("err = %v, ran = %d", err, ran)
	}
	rows := strings.Split(strings.TrimSpace(tb.String()), "\n")
	if len(rows) != 2 || !strings.Contains(rows[0], `"result":"pending"`) ||
		!strings.Contains(rows[1], `"result":"ok"`) {
		t.Fatalf("rows = %v", rows)
	}
}
