// Package status blocks a report draft that holds
// any number the facts don't.
//
// Move 79: weekly status report straight from your
// tools' APIs.
package status

import (
	"regexp"
	"strconv"
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

// Facts is what code counted. A nil pointer means
// the source was down: "not available", never zero.
type Facts struct {
	Week       int
	MergedPRs  *int
	Incidents  *int
	Deploys    *int
	SlippedIDs []string
}

// Allowed builds the allowed list from the struct
// plus the week number. Minimal helper.
func (f Facts) Allowed() []int {
	out := []int{f.Week}
	for _, p := range []*int{f.MergedPRs, f.Incidents, f.Deploys} {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out
}
