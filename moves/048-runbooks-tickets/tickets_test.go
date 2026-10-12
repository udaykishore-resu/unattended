package tickets

import (
	"strings"
	"testing"
	"time"
)

func TestTicketDoc(t *testing.T) {
	tk := Ticket{ID: "T-881", Subject: "EDI feed stuck",
		FirstMsg:   "Our 214s stopped arriving this morning.",
		Resolution: "Reset the EDI connection in the partner portal, then resend the last batch from the queue.",
		Closed:     time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)}
	doc, ok := TicketDoc(tk)
	if !ok {
		t.Fatal("good ticket skipped")
	}
	for _, want := range []string{"# EDI feed stuck", "## Problem", "## Fix",
		"Source: ticket T-881, closed 2026-09-03"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in\n%s", want, doc)
		}
	}
}

func TestSkips(t *testing.T) {
	reopened := Ticket{Resolution: strings.Repeat("word ", 20), Reopened: true}
	if _, ok := TicketDoc(reopened); ok {
		t.Error("reopened tickets didn't hold")
	}
	if _, ok := TicketDoc(Ticket{Resolution: "Fixed."}); ok {
		t.Error(`"Fixed." teaches nothing`)
	}
}
