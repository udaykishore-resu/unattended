// Package redact swaps personal data for stable
// placeholders before the model, and swaps them back
// just before a person sees the text.
//
// Move 95: redact PII before the model, re-hydrate
// after.
package redact

import (
	"fmt"
	"regexp"
	"strings"
)

// Redact swaps each match for a stable placeholder.
// The vault maps placeholders back to real values.
func Redact(s string) (string, map[string]string) {
	vault := map[string]string{}
	seen := map[string]string{}
	for _, p := range patterns { // email before phone
		n := 0
		s = p.re.ReplaceAllStringFunc(s,
			func(m string) string {
				if ph, ok := seen[m]; ok {
					return ph
				}
				n++
				ph := fmt.Sprintf("[%s_%d]", p.kind, n)
				seen[m], vault[ph] = ph, m
				return ph
			})
	}
	return s, vault
}

// patterns is a list of kinds and regexes. Email
// runs before phone, because an address can hold a
// run of digits. Version this list like a prompt.
var patterns = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"EMAIL", regexp.MustCompile(
		`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)},
	// One regex per kind, so numbering stays unique:
	// +44 20 7946 0958, (404) 555-0117, 404-555-0117.
	{"PHONE", regexp.MustCompile(
		`\+\d{1,3}(?:[ .-]?\d{1,4}){2,5}\b|` +
			`\(\d{3}\) ?\d{3}[-. ]\d{4}\b|` +
			`\b\d{3}[-. ]\d{3}[-. ]\d{4}\b`)},
}

// Rehydrate swaps placeholders back with a
// strings.ReplaceAll loop. Described in prose.
func Rehydrate(s string, vault map[string]string) string {
	for ph, real := range vault {
		s = strings.ReplaceAll(s, ph, real)
	}
	return s
}

// leftover matches placeholders and the ways a model
// rewrites them: [PHONE_3], phone_1, PHONE 2.
var leftover = regexp.MustCompile(
	`(?i)\[?\b(?:EMAIL|PHONE)[_ ]\d+\b\]?`)

// Leftovers finds placeholders the model invented or
// rewrote, still present after re-hydration; send
// those drafts to review.
func Leftovers(s string) []string {
	return leftover.FindAllString(s, -1)
}

// Collides reports input that already holds
// placeholder-shaped text. Re-hydration would turn it
// into someone's real value, so route it to review
// before redacting.
func Collides(s string) bool {
	return leftover.MatchString(s)
}
