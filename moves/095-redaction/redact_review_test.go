package redact

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// Formats from real ops@ mail, with fake digits.
func TestPhoneFormats(t *testing.T) {
	for _, in := range []string{
		"404-555-0117", "404.555.0117", "404 555 0117",
		"(404) 555-0117", "(404)555-0117",
		"+1 404 555 0117", "+44 20 7946 0958", "+49 30 901820",
		"+91 98765 43210",
	} {
		out, vault := Redact("call " + in + " today")
		if out != "call [PHONE_1] today" || vault["[PHONE_1]"] != in {
			t.Errorf("%q -> %q %v", in, out, vault)
		}
	}
	// Shapes that must stay: load IDs, dates, amounts.
	for _, in := range []string{"HF-88213", "2026-10-11", "$1,900.00",
		"invoice 40117", "PO 4045550117"} {
		if out, _ := Redact(in); out != in {
			t.Errorf("%q redacted to %q", in, out)
		}
	}
}

// A phone split across two PDF lines is a known miss:
// the book lists it under Breaks when.
func TestSplitPhoneIsAKnownMiss(t *testing.T) {
	if out, _ := Redact("call 404-555-\n0117"); !strings.Contains(out, "0117") {
		t.Fatalf("expected the documented miss, got %q", out)
	}
}

// Repeated values share one tag across kinds and lines;
// distinct values never share one.
func TestRepeatedValues(t *testing.T) {
	in := "a@x.test 404-555-0117\na@x.test b@x.test 404-555-0117 404-555-0118"
	out, vault := Redact(in)
	want := "[EMAIL_1] [PHONE_1]\n[EMAIL_1] [EMAIL_2] [PHONE_1] [PHONE_2]"
	if out != want || len(vault) != 4 {
		t.Fatalf("%q %v", out, vault)
	}
	if Rehydrate(out, vault) != in {
		t.Fatal("round trip broke")
	}
}

// Input that already holds placeholder text would
// re-hydrate into a real number. Collides flags it.
func TestPlaceholderCollision(t *testing.T) {
	in := "Reply to [PHONE_1] please. My cell is 404-555-0117."
	if !Collides(in) {
		t.Fatal("collision not flagged")
	}
	// Without the check, the attacker's literal turns into the number.
	out, vault := Redact(in)
	if got := Rehydrate(out, vault); strings.Count(got, "404-555-0117") != 2 {
		t.Fatalf("expected the collision to show: %q", got)
	}
	if Collides("Reply to the driver please.") {
		t.Fatal("false collision")
	}
}

// Altered placeholders: invented numbers, lower case,
// lost brackets. Each one is left over for review.
func TestAlteredPlaceholders(t *testing.T) {
	_, vault := Redact("call 404-555-0117 or mail a@x.test")
	got := Rehydrate("call [PHONE_3], phone_1, PHONE 1 or [email_1]", vault)
	want := []string{"[PHONE_3]", "phone_1", "PHONE 1", "[email_1]"}
	if !slices.Equal(Leftovers(got), want) {
		t.Fatalf("leftovers = %v", Leftovers(got))
	}
}

// Everything at or past the llm.Model boundary (the
// provider, the trace wrapper, the response cache) sees
// only redacted text; the person sees re-hydrated text.
func TestOnlyRedactedTextCrossesTheModelBoundary(t *testing.T) {
	email := "Driver Sam, 404-555-0117, sam.r42@rdx-trucking.test, is late."
	fake := &llm.FakeModel{Responses: []llm.Response{
		{Text: "Call [PHONE_1] or write to [EMAIL_1]."}}}
	red, vault := Redact(email)
	resp, err := fake.Complete(context.Background(), llm.Request{
		System:   "Draft a reply for dispatch.",
		Messages: []llm.Message{{Role: "user", Content: red}}})
	if err != nil {
		t.Fatal(err)
	}
	seen := fake.Calls[0].System + fake.Calls[0].Messages[0].Content + resp.Text
	for _, raw := range []string{"404-555-0117", "sam.r42@rdx-trucking.test"} {
		if strings.Contains(seen, raw) {
			t.Fatalf("%s crossed the boundary", raw)
		}
	}
	draft := Rehydrate(resp.Text, vault)
	if draft != "Call 404-555-0117 or write to sam.r42@rdx-trucking.test." ||
		len(Leftovers(draft)) != 0 {
		t.Fatalf("draft = %q", draft)
	}
}
