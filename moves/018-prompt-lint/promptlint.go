// Package promptlint flags vague verbs, unfenced
// template slots and always/never conflicts.
//
// Move 18: lint your prompts.
package promptlint

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var slot = regexp.MustCompile(`\{\{.*?\}\}`)
var vague = regexp.MustCompile(`(?i)\b(handle|` +
	`process|deal with|appropriately)\b`)

func Lint(p string) (out []string) {
	inTag := false
	for i, ln := range strings.Split(p, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "<") {
			inTag = !strings.HasPrefix(t, "</")
		}
		at := strconv.Itoa(i+1) + ": "
		if m := vague.FindString(t); m != "" {
			out = append(out, at+"vague verb "+m)
		}
		if slot.MatchString(t) && !inTag {
			out = append(out, at+"unfenced variable")
		}
	}
	return append(out, conflicts(p)...)
}

var rule = regexp.MustCompile(
	`(?i)\b(always|never)\s+(\w+)\s+(\w+)`)

// conflicts is described in prose: it matches
// "always" or "never" plus the next two words, and
// reports when the opposite pair appeared earlier.
// Minimal implementation.
func conflicts(p string) []string {
	first := map[string]int{} // "always reply in" -> line
	var out []string
	for i, ln := range strings.Split(p, "\n") {
		for _, m := range rule.FindAllStringSubmatch(ln, -1) {
			word := strings.ToLower(m[1])
			rest := strings.ToLower(m[2] + " " + m[3])
			opp := "never"
			if word == "never" {
				opp = "always"
			}
			if j, ok := first[opp+" "+rest]; ok {
				out = append(out, fmt.Sprintf(
					"%d: %s %s conflicts with line %d",
					i+1, word, rest, j))
			}
			if _, ok := first[word+" "+rest]; !ok {
				first[word+" "+rest] = i + 1
			}
		}
	}
	return out
}
