package audit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// write logs a day of actions in two retention
// classes and returns the log plus each segment's
// checkpoint (the last hash, signed at day close).
func write(t *testing.T) (string, map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	l := NewLog(&buf)
	for _, e := range []Entry{
		{Class: "invoice-7y", RunID: "r1", Agent: "billing-agent:1.4.2",
			Tool: "set_amount", Input: "invoice:40117", Result: "pending"},
		{Class: "dispatch-2y", RunID: "r2", Agent: "dispatch-agent:2.0.1",
			Tool: "add_note", Input: "load:HF-1", Result: "ok"},
		{Class: "invoice-7y", RunID: "r1", Agent: "billing-agent:1.4.2",
			Tool: "set_amount", Input: "invoice:40117", Decision: "approved",
			Approver: "marcus", Result: "ok"},
		{Class: "dispatch-2y", RunID: "r3", Agent: "dispatch-agent:2.0.1",
			Tool: "cancel_load", Decision: "denied", Result: "denied"},
	} {
		if err := l.Write(e); err != nil {
			t.Fatal(err)
		}
	}
	cps := map[string]string{}
	for seg, h := range l.last {
		cps[seg] = h
	}
	return buf.String(), cps
}

func TestCleanLogVerifies(t *testing.T) {
	log, cps := write(t)
	if err := Verify(strings.NewReader(log), cps); err != nil {
		t.Fatal(err)
	}
}

// Retention drops the whole dispatch segment but
// keeps its tiny checkpoint. The invoice chain still
// verifies.
func TestChainVerifiesAfterSegmentExpires(t *testing.T) {
	log, cps := write(t)
	var kept []string
	for _, ln := range strings.SplitAfter(log, "\n") {
		if ln != "" && !strings.Contains(ln, `"class":"dispatch-2y"`) {
			kept = append(kept, ln)
		}
	}
	live := map[string]string{}
	for seg, h := range cps {
		if strings.HasPrefix(seg, "invoice-7y/") {
			live[seg] = h
		}
	}
	if err := Verify(strings.NewReader(strings.Join(kept, "")), live); err != nil {
		t.Fatalf("expired segment broke the rest: %v", err)
	}
}

// "Who set invoice 40117 to zero?" An edited row
// breaks its hash; a deleted row breaks the chain.
func TestTamperingIsCaught(t *testing.T) {
	log, cps := write(t)
	edited := strings.Replace(log, `"approver":"marcus"`, `"approver":"theo"`, 1)
	if err := Verify(strings.NewReader(edited), cps); err == nil {
		t.Fatal("edited row verified")
	}
	lines := strings.SplitAfter(log, "\n")
	var dropped []string
	for i, ln := range lines {
		if i != 0 { // delete the first invoice row
			dropped = append(dropped, ln)
		}
	}
	if err := Verify(strings.NewReader(strings.Join(dropped, "")), cps); err == nil {
		t.Fatal("deleted row verified")
	}
	// Deleting the last row of a segment is caught by its checkpoint.
	noTail := strings.Join(lines[:2], "") + lines[3]
	if err := Verify(strings.NewReader(noTail), cps); err == nil {
		t.Fatal("truncated segment verified")
	}
}

func TestEntryShape(t *testing.T) {
	log, _ := write(t)
	var e Entry
	json.Unmarshal([]byte(strings.SplitN(log, "\n", 2)[0]), &e)
	if e.Prev != "" || len(e.Hash) != 64 || !strings.HasPrefix(e.Seg, "invoice-7y/") {
		t.Fatalf("first entry = %+v", e)
	}
}
