// Package tickets turns a closed ticket into a short
// problem-and-fix document, or skips it.
//
// Move 48: index runbooks and closed tickets first.
package tickets

import (
	"fmt"
	"strings"
	"time"

	pii "github.com/udaykishore-resu/unattended/moves/095-redaction"
)

// Ticket is the queue's record; its fields mean
// what they say.
type Ticket struct {
	ID, Subject, FirstMsg, Resolution string
	CloseCode                         string
	Reopened                          bool
	Closed                            time.Time
}

// fixCodes lists the close codes that mean a fix.
// "customer_quiet" and "duplicate" are not in it.
var fixCodes = map[string]bool{
	"resolved":   true,
	"workaround": true,
	"config_fix": true,
}

// TicketDoc turns a closed ticket into a short
// problem/fix doc, or skips it. Free text is
// redacted before it can reach the index.
func TicketDoc(t Ticket) (string, bool) {
	res := strings.TrimSpace(t.Resolution)
	if t.Reopened || !fixCodes[t.CloseCode] ||
		len(strings.Fields(res)) < 12 {
		return "", false
	}
	doc := fmt.Sprintf(
		"# %s\n\n## Problem\n%s\n\n## Fix\n%s\n\n"+
			"Source: ticket %s, closed %s\n",
		redact(t.Subject), redact(t.FirstMsg),
		redact(res), t.ID,
		t.Closed.Format("2006-01-02"))
	return doc, true
}

// redact is Move 95's pattern pass with the vault
// dropped: the index never re-hydrates. It catches
// emails and phone numbers, not names.
func redact(s string) string {
	out, _ := pii.Redact(s)
	return out
}
