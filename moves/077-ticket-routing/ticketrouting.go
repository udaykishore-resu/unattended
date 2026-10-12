// Package ticketrouting shows a draft reply only if
// every help article it cites was in the prompt, and
// never for a claims ticket.
//
// Move 77: support ticket routing plus a draft reply.
package ticketrouting

import (
	"regexp"
	"strings"
)

// Ticket is what code knows about the ticket before
// the model runs. Claim comes from the ticket form,
// not from the model.
type Ticket struct {
	ID    string
	Claim bool
}

// Result is the model's output: a queue from an
// enum (including "unsure"), an urgent flag, the
// draft, and the article IDs it used.
type Result struct {
	Queue  string   `json:"queue"`
	Urgent bool     `json:"urgent"`
	Draft  string   `json:"draft"`
	KBIDs  []string `json:"kb_ids"`
}

// Keep a draft only if every article it cites
// was in the prompt. This checks membership,
// not that the article supports the reply.
func Usable(t Ticket, r Result,
	shown map[string]bool) bool {
	if t.Claim || r.Queue == "claims" ||
		r.Queue == "unsure" || len(r.KBIDs) == 0 {
		return false
	}
	for _, id := range r.KBIDs {
		if !shown[id] {
			return false
		}
	}
	return true
}

var promise = regexp.MustCompile(
	`(?i)\b(refund\w*|credit\w*|waiv\w*|guarantee\w*|compensat\w*)\b`)

// Promises returns promise words in a draft. A hit
// puts a flag on the draft for the agent. It is a
// word list: "we'll take care of that charge" slips
// past it, so it is a prompt for the reviewer, not
// the control. The control is that a person sends.
func Promises(draft string) []string {
	var out []string
	for _, m := range promise.FindAllString(draft, -1) {
		out = append(out, strings.ToLower(m))
	}
	return out
}

// Shown is the send path's view: the draft the agent
// sees (empty when not usable) and its flags.
func Shown(t Ticket, r Result,
	shown map[string]bool) (draft string, flags []string) {
	if !Usable(t, r, shown) {
		return "", nil
	}
	return r.Draft, Promises(r.Draft)
}
