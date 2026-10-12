// Package acceptance records the fate of every draft
// a person handled: accepted, edited, rewrote or
// ignored.
//
// Move 87: track human acceptance rate, not only
// accuracy.
package acceptance

import "strings"

type Fate string

const (
	Accepted Fate = "accepted" // sent as is
	Edited   Fate = "edited"
	Rewrote  Fate = "rewrote"
	Ignored  Fate = "ignored" // never used
)

func Classify(draft, sent string, used bool) Fate {
	switch {
	case !used:
		return Ignored
	case norm(draft) == norm(sent):
		return Accepted
	case overlap(draft, sent) >= 0.6:
		return Edited
	}
	return Rewrote
}

// norm lowercases and collapses whitespace.
func norm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// overlap is the share of sent words that came from
// the draft.
func overlap(draft, sent string) float64 {
	have := map[string]int{}
	for _, w := range strings.Fields(norm(draft)) {
		have[w]++
	}
	words := strings.Fields(norm(sent))
	if len(words) == 0 {
		return 0
	}
	from := 0
	for _, w := range words {
		if have[w] > 0 {
			have[w]--
			from++
		}
	}
	return float64(from) / float64(len(words))
}
