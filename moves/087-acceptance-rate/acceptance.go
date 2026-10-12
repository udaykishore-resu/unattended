// Package acceptance records the fate of every draft
// a person handled: accepted, edited, rewrote or
// ignored.
//
// Move 87: track human acceptance rate, not only
// accuracy.
package acceptance

import (
	"log/slog"
	"strings"
)

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

// OnSend is what the send tool calls with the body it
// is about to send: the final text, not a saved draft.
// It classifies and logs the fate as Move 83's outcome
// event on the draft's call ID.
func OnSend(log *slog.Logger, callID, draft, body string) Fate {
	f := Classify(draft, body, true)
	log.Info("llm.outcome", "call_id", callID, "fate", string(f))
	return f
}
