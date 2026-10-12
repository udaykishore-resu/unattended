package tickets

import (
	"strings"
	"testing"
	"time"
)

var long = "Reset the EDI connection in the partner " +
	"portal, then resend the last batch from the queue."

// A long resolution under a close code that is not a
// fix ("customer went quiet") is skipped.
func TestOnlyFixCodesAreIndexed(t *testing.T) {
	for code, want := range map[string]bool{
		"resolved": true, "workaround": true,
		"customer_quiet": false, "duplicate": false, "": false,
	} {
		_, ok := TicketDoc(Ticket{ID: "T-1", Subject: "s",
			Resolution: long, CloseCode: code})
		if ok != want {
			t.Errorf("close code %q: indexed=%v, want %v", code, ok, want)
		}
	}
}

// Phones and emails are redacted in the subject, the
// first message and the resolution before indexing.
func TestRedactsEveryFreeTextField(t *testing.T) {
	tk := Ticket{ID: "T-882", CloseCode: "resolved",
		Subject:  "Call back 404-555-0137 re: EDI",
		FirstMsg: "Dana here, reach me at dana@brightline.example or 404-555-0137.",
		Resolution: long + " Confirmed with the shipper at 678.555.0199 " +
			"and ops@brightline.example.",
		Closed: time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)}
	doc, ok := TicketDoc(tk)
	if !ok {
		t.Fatal("skipped")
	}
	for _, leak := range []string{"404-555-0137", "678.555.0199",
		"dana@brightline.example", "ops@brightline.example"} {
		if strings.Contains(doc, leak) {
			t.Errorf("%q reached the index:\n%s", leak, doc)
		}
	}
	if !strings.Contains(doc, "[PHONE_1]") || !strings.Contains(doc, "[EMAIL_1]") {
		t.Errorf("placeholders missing:\n%s", doc)
	}
	// Metadata in the doc is the ticket ID and the
	// close date only; neither carries personal data.
	if !strings.Contains(doc, "Source: ticket T-882, closed 2026-09-03") {
		t.Errorf("source line:\n%s", doc)
	}
}

// Names are not caught by the pattern pass. This test
// pins that limit so the book never claims otherwise.
func TestNamesAreNotRedactedByPatterns(t *testing.T) {
	tk := Ticket{ID: "T-883", CloseCode: "resolved",
		Subject: "EDI", FirstMsg: "Dana from Brightline here.",
		Resolution: long}
	doc, _ := TicketDoc(tk)
	if !strings.Contains(doc, "Dana") {
		t.Fatal("names now redacted: update the book's wording")
	}
}
