// Package status builds the weekly facts in code and
// blocks a report draft that holds any number the
// facts don't.
//
// Move 79: weekly status report straight from your
// tools' APIs.
package status

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var num = regexp.MustCompile(`\b\d+\b`)

// Stray returns numbers in the draft that are
// not in the facts. Anything here is suspect.
func Stray(draft string, allowed []int) []int {
	ok := map[int]bool{}
	for _, n := range allowed {
		ok[n] = true
	}
	var bad []int
	for _, s := range num.FindAllString(draft, -1) {
		n, _ := strconv.Atoi(s)
		if !ok[n] {
			bad = append(bad, n)
		}
	}
	return bad
}

// Stray is a narrow check. It catches a number the
// facts don't hold (34 when code counted 31). It does
// not catch a real number pinned to the wrong metric
// ("31 incidents" when 31 is the PR count). The
// author still reads every figure before sending.

// Facts is what code counted. A nil pointer means
// the source was down: "not available", never zero.
type Facts struct {
	Week       int       `json:"week"`
	From       time.Time `json:"from,omitzero"`
	To         time.Time `json:"to,omitzero"`
	MergedPRs  *int      `json:"merged_prs"`
	Incidents  *int      `json:"incidents"`
	Deploys    *int      `json:"deploys"`
	SlippedIDs []string  `json:"slipped_ids"`
}

// Allowed builds the allowed list from the struct
// plus the week number and the dates of the range.
func (f Facts) Allowed() []int {
	out := []int{f.Week}
	for _, p := range []*int{f.MergedPRs, f.Incidents, f.Deploys} {
		if p != nil {
			out = append(out, *p)
		}
	}
	if f.From.IsZero() || f.To.IsZero() {
		return out
	}
	last := f.To.AddDate(0, 0, -1) // To is exclusive
	for _, t := range []time.Time{f.From, last} {
		out = append(out, t.Year(), int(t.Month()), t.Day())
	}
	return out
}

// ErrStray blocks a draft. The caller does not send
// a blocked draft to its author; it regenerates once
// or hands over the facts alone.
type ErrStray struct{ Numbers []int }

func (e ErrStray) Error() string {
	return fmt.Sprintf("draft has numbers not in facts: %v", e.Numbers)
}

// Release is the gate between the model and the
// author: any stray number blocks the draft.
func Release(draft string, f Facts) error {
	if bad := Stray(draft, f.Allowed()); len(bad) > 0 {
		return ErrStray{bad}
	}
	return nil
}

// Week is the one helper that pins the report range:
// Monday 00:00 to the next Monday 00:00 in the team's
// zone, for the week holding now. Every fetcher gets
// this range, converted once.
func Week(now time.Time, loc *time.Location) (from, to time.Time) {
	l := now.In(loc)
	back := (int(l.Weekday()) + 6) % 7 // days since Monday
	from = time.Date(l.Year(), l.Month(), l.Day()-back, 0, 0, 0, 0, loc)
	return from, from.AddDate(0, 0, 7)
}

// Page is one page of an API listing.
type Page struct {
	IDs  []string
	Next string // empty on the last page
}

// Count follows next-page links and counts distinct
// IDs. Any failed page returns nil: the source is
// "not available", never a partial count or a zero.
func Count(fetch func(cursor string) (Page, error)) (n *int, pages int) {
	seen := map[string]bool{}
	cursor := ""
	for {
		p, err := fetch(cursor)
		if err != nil {
			return nil, pages
		}
		pages++
		for _, id := range p.IDs {
			seen[id] = true
		}
		if p.Next == "" {
			c := len(seen)
			return &c, pages
		}
		cursor = p.Next
	}
}

// Show renders one fact for the prompt.
func Show(name string, v *int) string {
	if v == nil {
		return name + ": not available"
	}
	return fmt.Sprintf("%s: %d", name, *v)
}
