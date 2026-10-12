// Package tickets turns a closed ticket into a short
// problem-and-fix document, or skips it.
//
// Move 48: index runbooks and closed tickets first.
package tickets

import (
	"fmt"
	"strings"
	"time"
)

// Ticket is the queue's record; its fields mean
// what they say.
type Ticket struct {
	ID, Subject, FirstMsg, Resolution string
	Reopened                          bool
	Closed                            time.Time
}

// TicketDoc turns a closed ticket into a short
// problem/fix document, or skips it.
func TicketDoc(t Ticket) (string, bool) {
	res := strings.TrimSpace(t.Resolution)
	if t.Reopened || len(strings.Fields(res)) < 12 {
		return "", false
	}
	doc := fmt.Sprintf(
		"# %s\n\n## Problem\n%s\n\n## Fix\n%s\n\n"+
			"Source: ticket %s, closed %s\n",
		t.Subject, t.FirstMsg, res, t.ID,
		t.Closed.Format("2006-01-02"))
	return doc, true
}
