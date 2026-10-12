package emailrecord

import (
	"strings"
	"testing"
)

// The stale $2,150 rate lived in the quoted part.
func TestStripsQuotedHistory(t *testing.T) {
	cases := map[string]string{
		"gmail": "Can you do Thursday instead?\n\n" +
			"On Mon, Mar 2, 2026 at 9:01 AM Acme wrote:\n" +
			"> Rate is $2,150\n",
		"quote marker": "Thursday works.\n> Rate is $2,150\n",
		"outlook": "Thursday works.\n-----Original Message-----\n" +
			"Rate is $2,150\n",
	}
	for name, body := range cases {
		got := newText(body)
		if strings.Contains(got, "2,150") {
			t.Errorf("%s: stale rate leaked: %q", name, got)
		}
		if got == "" {
			t.Errorf("%s: new text lost", name)
		}
	}
}

func TestKeepsLinesThatOnlyLookLikeHeaders(t *testing.T) {
	body := "On Thursday we need two trucks.\nThanks\n"
	if got := newText(body); got != body {
		t.Fatalf("got %q", got)
	}
}

// newText is a starting heuristic, not a quote parser.
// This documents its known false positive: a new line
// that happens to read "On ... wrote:" cuts real text.
// Shadow mode and the logged first line of each
// stripped block (Move 25, Breaks when) are how you
// find these.
func TestKnownFalsePositive(t *testing.T) {
	body := "Rate is fine.\nOn Friday the driver wrote:\nDock 4 only.\n"
	if got := newText(body); got != "Rate is fine.\n" {
		t.Fatalf("heuristic changed; update the book: %q", got)
	}
}
