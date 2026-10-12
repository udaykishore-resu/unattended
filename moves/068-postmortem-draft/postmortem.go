// Package postmortem flags numbers in a draft that
// do not appear in the facts it was given.
//
// Move 68: postmortem first draft from chat history
// and metrics.
package postmortem

import (
	"regexp"
	"strings"
)

var num = regexp.MustCompile(`\d[\d,.]*%?`)

func Unsourced(draft, facts string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range num.FindAllString(draft, -1) {
		n = strings.TrimRight(n, ".,")
		if seen[n] || len(n) < 2 {
			continue
		}
		seen[n] = true
		if !strings.Contains(facts, n) {
			out = append(out, n)
		}
	}
	return out
}
