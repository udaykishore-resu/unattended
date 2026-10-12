// Package postmortem checks a model-written
// postmortem draft. Unsourced flags numbers that do
// not appear in the facts it was given; Unconfirmed
// flags summary sentences that claim a cause or a fix
// the review has not confirmed. Both are candidate
// detectors for a reviewer, not proof of sourcing.
//
// Move 68: postmortem first draft from chat history
// and metrics.
package postmortem

import (
	"errors"
	"regexp"
	"strings"
)

var num = regexp.MustCompile(`\d[\d,.]*%?`)

// Unsourced compares whole number tokens; norm trims
// trailing punctuation and drops thousands
// separators, so "1,340." matches "1340".
func Unsourced(draft, facts string) []string {
	known := map[string]bool{}
	for _, n := range num.FindAllString(facts, -1) {
		known[norm(n)] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, n := range num.FindAllString(draft, -1) {
		k := norm(n)
		if seen[k] || len(k) < 2 {
			continue
		}
		seen[k] = true
		if !known[k] {
			out = append(out, k)
		}
	}
	return out
}

func norm(n string) string {
	n = strings.TrimRight(n, ".,")
	return strings.ReplaceAll(n, ",", "")
}

// causal matches wording that claims a cause or a fix.
var causal = regexp.MustCompile(`(?i)\b(caus(e|ed|es|ing)|root|` +
	`fix(ed|es)?|due to|because|led to|triggered|resolved by)\b`)

var sentence = regexp.MustCompile(`[^.!?]+[.!?]*`)

// Unconfirmed returns each summary sentence that
// claims a cause or a fix without naming a factor the
// review confirmed. "Possibly related" passes; "caused
// by" and "fixed it" need a confirmed factor.
func Unconfirmed(summary string, confirmed []string) []string {
	var out []string
	for _, s := range sentence.FindAllString(summary, -1) {
		s = strings.TrimSpace(s)
		if s == "" || !causal.MatchString(s) {
			continue
		}
		named := false
		for _, f := range confirmed {
			if f != "" && strings.Contains(strings.ToLower(s), strings.ToLower(f)) {
				named = true
				break
			}
		}
		if !named {
			out = append(out, s)
		}
	}
	return out
}

// Inputs are what the draft is built from.
type Inputs struct {
	TimelineSignedBy string   // incident lead's sign-off (Move 67)
	Facts            string   // metrics table computed in code
	Confirmed        []string // factors confirmed in review; empty at first
}

// ErrUnsigned: the draft starts only from a timeline
// the incident lead signed.
var ErrUnsigned = errors.New("postmortem: timeline not signed off")

// Ready refuses to start a draft without the sign-off.
func (in Inputs) Ready() error {
	if strings.TrimSpace(in.TimelineSignedBy) == "" {
		return ErrUnsigned
	}
	return nil
}

// Flags lists everything the reviewer must look at
// before the draft becomes the record.
type Flags struct {
	Numbers, Causes []string
}

// Check runs both detectors over a draft.
func Check(in Inputs, summary, body string) (Flags, error) {
	if err := in.Ready(); err != nil {
		return Flags{}, err
	}
	return Flags{
		Numbers: Unsourced(summary+"\n"+body, in.Facts),
		Causes:  Unconfirmed(summary, in.Confirmed),
	}, nil
}
