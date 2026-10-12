package logtriage

import (
	"context"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
	redact "github.com/udaykishore-resu/unattended/moves/095-redaction"
)

// Overlaps and order: each case is checked against
// the exact template, so a greedy or misordered rule
// shows up here, not in an incident.
func TestMaskOrderAndOverlaps(t *testing.T) {
	cases := map[string]string{
		// lower and upper case UUIDs both collapse
		"req 123e4567-e89b-12d3-a456-426614174000 done": "req <uuid> done",
		"req 123E4567-E89B-12D3-A456-426614174000 done": "req <uuid> done",
		// an IP with a port: the IP rule runs before numbers
		"dial 10.2.3.4:5432 refused": "dial <ip>:<n> refused",
		// a three-part version is not an IP
		"client v1.27.2 connected": "client v<n>.<n> connected",
		// RFC 3339 timestamps keep their shape
		"at 2026-03-09T14:12:05.123Z": "at <n>-<n>-<n>T<n>:<n>:<n>Z",
		// ticket-style IDs run before plain numbers
		"quote LD-88213 failed after 412ms": "quote <id> failed after <n>ms",
		// a quoted value goes first, digits and all
		`carrier "ACME 42" returned 503`: `carrier "<s>" returned <n>`,
	}
	for in, want := range cases {
		if got := Template(in); got != want {
			t.Errorf("Template(%q)\n got %q\nwant %q", in, got, want)
		}
	}
	// Same template for lines that differ only in values.
	a := Template("quote LD-1 failed after 9ms from 10.0.0.1")
	b := Template("quote LD-88213 failed after 412ms from 172.16.4.20")
	if a != b {
		t.Errorf("%q != %q", a, b)
	}
}

// A known limit, pinned so nobody mistakes the masks
// for redaction: an unquoted email survives Template.
func TestMasksAreNotRedaction(t *testing.T) {
	got := Template("login failed for dana@halcyon.test")
	if !strings.Contains(got, "dana@halcyon.test") {
		t.Fatalf("expected the email to survive masking: %q", got)
	}
}

// Samples are redacted before the request leaves.
func TestSamplesRedactedBeforeTheModel(t *testing.T) {
	lines := []string{
		"login failed for dana@halcyon.test from 10.0.0.7",
		"login failed for ines@halcyon.test from 10.0.0.8",
		"callback 555-201-3344 timed out",
	}
	m := &llm.FakeModel{Responses: []llm.Response{{Text: "C1 is noise"}}}
	scrub := func(s string) string { r, _ := redact.Redact(s); return r }
	if _, err := Ask(context.Background(), m, Clusters(lines), 30, scrub); err != nil {
		t.Fatal(err)
	}
	sent := m.Calls[0].Messages[0].Content
	for _, pii := range []string{"dana@", "ines@", "555-201-3344"} {
		if strings.Contains(sent, pii) {
			t.Fatalf("%s reached the model:\n%s", pii, sent)
		}
	}
	if !strings.Contains(sent, "[EMAIL_1]") {
		t.Fatalf("no placeholder:\n%s", sent)
	}
}
