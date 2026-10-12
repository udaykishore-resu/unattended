package evidence

import (
	"encoding/json"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

const email = `Hi team,
Please book pickup on
Thursday 3/12 at the Dayton dock.
Weight is 42,000 lbs.`

func TestGrounded(t *testing.T) {
	cases := map[string]bool{
		"pickup on thursday 3/12":        true, // wraps a line, other case
		"Weight is 42,000 lbs.":          true,
		"Weight is 42,000 lbs":           true,  // trailing period dropped
		"pickup on Friday 3/13":          false, // invented quote fails
		"":                               false, // no quote, no trust
		"   ":                            false,
		"the shipper confirmed by phone": false,
		"Weight is 42000 lbs.":           false, // punctuation changed: review
	}
	for ev, want := range cases {
		if got := grounded(email, ev); got != want {
			t.Errorf("grounded(%q) = %v, want %v", ev, got, want)
		}
	}
}

// Unicode whitespace (NBSP, thin space, tab, CRLF)
// folds like a space; a zero-width space is not
// whitespace, so it fails closed and goes to review.
func TestUnicodeWhitespace(t *testing.T) {
	src := "Weight is  42,000\tlbs.\r\nThanks"
	if !grounded(src, "weight is 42,000 lbs.") {
		t.Fatal("NBSP/thin space/tab should fold")
	}
	if grounded("Weight is 42,​000 lbs.", "Weight is 42,000 lbs.") {
		t.Fatal("zero-width space must not match")
	}
	// Curly vs straight apostrophe: no match, review.
	if grounded("Acme’s dock", "Acme's dock") {
		t.Fatal("quote style changed: must not match")
	}
}

// grounded is a presence check: a phrase repeated in
// the source matches, whichever copy the model meant.
func TestRepeatedPhraseIsPresenceOnly(t *testing.T) {
	src := "Pickup 3/12. Delivery 3/12."
	if !grounded(src, "3/12") {
		t.Fatal("repeated phrase should be found")
	}
}

// The reviewer's case: quote says 42,000, value says
// 24000. grounded passes; the numeric check catches it.
func TestValueMustMatchItsQuote(t *testing.T) {
	ok := Field[int64]{42000, "high", "Weight is 42,000 lbs."}
	if p := CheckInt("weight_lb", email, ok); len(p) != 0 {
		t.Fatalf("good field: %v", p)
	}
	swapped := Field[int64]{24000, "high", "Weight is 42,000 lbs."}
	p := CheckInt("weight_lb", email, swapped)
	if len(p) != 1 || p[0] != "weight_lb: value 24000 not in its evidence" {
		t.Fatalf("swapped digits: %v", p)
	}
	invented := Field[int64]{41000, "high", "Weight is 41,000 lbs."}
	if p := CheckInt("weight_lb", email, invented); len(p) != 1 {
		t.Fatalf("invented quote: %v", p)
	}
}

func TestNumIn(t *testing.T) {
	cases := []struct {
		q    string
		v    int64
		want bool
	}{
		{"42,000 lbs", 42000, true},
		{"42000", 42000, true},
		{"420,000 lbs", 42000, false}, // digits inside a bigger number
		{"approx 42k", 42000, false},  // not a copy: review
		{"42.000 kg", 42000, false},   // EU separators: convert in Hack 28
		{"weight 42,000; pallets 20", 20, true},
	}
	for _, c := range cases {
		if got := numIn(c.q, c.v); got != c.want {
			t.Errorf("numIn(%q, %d) = %v", c.q, c.v, got)
		}
	}
}

// Check against the exact text sent. After redaction,
// a quote of the raw name fails; the redacted quote
// passes.
func TestCheckAgainstTextSent(t *testing.T) {
	raw := "Contact Jane Roe. Weight is 42,000 lbs."
	req := llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "Contact [NAME]. Weight is 42,000 lbs."},
	}}
	sent := Sent(req)
	if grounded(sent, "Contact Jane Roe") {
		t.Fatal("raw text was never sent")
	}
	if !grounded(sent, "contact [name]") || !grounded(raw, "Contact Jane Roe") {
		t.Fatal("redacted quote should match the text sent")
	}
	f := Field[int64]{42000, "high", "Weight is 42,000 lbs."}
	if p := CheckInt("weight_lb", sent, f); len(p) != 0 {
		t.Fatalf("got %v", p)
	}
}

// Synthesis story (Part III): with the currency parser
// deleted, the model returned a converted USD amount.
// Its quote is not in the email, so the evidence check
// sends it to review instead of to the batch.
func TestConvertedAmountFailsEvidence(t *testing.T) {
	claim := "Detention, 6 hours: CAD 1.240,00 per agreement."
	if grounded(claim, "USD 912.00") {
		t.Fatal("a converted amount is not in the source")
	}
	if !grounded(claim, "CAD 1.240,00") {
		t.Fatal("the raw amount is in the source")
	}
}

func TestFieldDecodes(t *testing.T) {
	var f Field[int]
	in := `{"value":42000,"confidence":"high","evidence":"Weight is 42,000 lbs."}`
	if err := json.Unmarshal([]byte(in), &f); err != nil {
		t.Fatal(err)
	}
	if f.Value != 42000 || !grounded(email, f.Evidence) {
		t.Fatalf("field = %+v", f)
	}
}
