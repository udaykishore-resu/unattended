// Package callfollowup gates transcription on
// consent and turns a call summary into the rep's
// checklist.
//
// Move 81: call recording, summary, follow-up email.
package callfollowup

import (
	"errors"
	"slices"
)

func Start(c Call) error {
	if !c.ConsentRecorded {
		return ErrNoConsent // never transcribe
	}
	return enqueue(c.ID)
}

func Checks(s Summary, turns []Turn) []string {
	var list []string
	for _, c := range s.Commitments {
		sides := sidesAt(c.Start, turns)
		switch {
		case len(sides) == 0:
			list = append(list,
				"no turn at timestamp: "+c.What)
		case !slices.Contains(sides, c.Who):
			list = append(list,
				"check who said: "+c.What)
		}
	}
	for _, n := range s.Numbers {
		list = append(list, "confirm: "+n)
	}
	return list
}

// The types and helpers below are described in the
// book's prose; minimal implementations.

// ErrNoConsent stops a call with no consent marker.
var ErrNoConsent = errors.New("no recorded consent")

type Call struct {
	ID              string
	ConsentRecorded bool
}

// enqueue hands the call to the transcription
// queue. A variable so tests can see it.
var enqueue = func(id string) error { return nil }

// Turn is one numbered transcript turn, like
// [0412 them], spanning [Start, End) seconds. Side
// ("us" or "them") comes from the phone system.
type Turn struct {
	Start, End int
	Side, Text string
}

// Summary holds decisions, open questions,
// commitments, and every number said aloud.
type Summary struct {
	Decisions, Questions []string
	Commitments          []Commitment
	Numbers              []string
}

// Commitment is who ("us" or "them"), what, when and
// a start second.
type Commitment struct {
	Who, What, When string
	Start           int
}

// sidesAt returns the side of every turn that spans
// the claimed second. Overlapping speech gives two.
func sidesAt(sec int, turns []Turn) []string {
	var out []string
	for _, t := range turns {
		if t.Start <= sec && sec < t.End {
			out = append(out, t.Side)
		}
	}
	return out
}
