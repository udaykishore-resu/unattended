// Package actionitems checks each model-proposed
// action item: owner on the roster, quote really in
// the notes.
//
// Move 74: meeting notes, action items, tracker.
package actionitems

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Item struct {
	Task  string `json:"task"`
	Owner string `json:"owner"` // must be on roster
	Due   string `json:"due"`   // phrase as said
	Quote string `json:"quote"` // copied from notes
}

func Check(it Item, roster map[string]bool,
	notes string) string {
	switch {
	case flat(it.Quote) == "":
		return "no quote"
	case !roster[it.Owner]:
		return "owner not on roster"
	case !strings.Contains(flat(notes), flat(it.Quote)):
		return "quote not found in notes"
	}
	return ""
}

var straight = strings.NewReplacer(
	"“", `"`, "”", `"`,
	"‘", "'", "’", "'")

// flat lowercases, straightens curly quotes and
// collapses whitespace.
func flat(s string) string {
	s = straight.Replace(strings.ToLower(s))
	return strings.Join(strings.Fields(s), " ")
}

// Key hashes meeting id, owner and flattened quote,
// never the task text, which the model rewords every
// run (Move 52). Described in prose; minimal.
func Key(meetingID string, it Item) string {
	sum := sha256.Sum256([]byte(meetingID + "\x00" +
		it.Owner + "\x00" + flat(it.Quote)))
	return hex.EncodeToString(sum[:16])
}
