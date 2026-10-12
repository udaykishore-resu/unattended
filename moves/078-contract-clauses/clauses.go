// Package clauses runs the contract playbook as
// code over model-extracted clauses.
//
// Move 78: contract clause extraction and red-flag
// notes.
package clauses

import "strings"

// Clause is one extracted clause: kind from an enum,
// exact quote, page, and any day count.
type Clause struct {
	Kind  string `json:"kind"`
	Quote string `json:"quote"`
	Page  int    `json:"page"`
	Days  *int   `json:"days"`
}

// Flag is one note for the named reviewer.
type Flag struct{ Code, Msg string }

func Review(cs []Clause, pages []string) []Flag {
	var fs []Flag
	seen := map[string]bool{}
	for _, c := range cs {
		if c.Page < 1 || c.Page > len(pages) ||
			!strings.Contains(pages[c.Page-1], c.Quote) {
			fs = append(fs, Flag{"quote", "not on page"})
			continue
		}
		seen[c.Kind] = true
		if c.Kind == "payment_terms" && c.Days != nil &&
			*c.Days > 45 {
			fs = append(fs, Flag{"pay", "over 45 days"})
		}
	}
	if !seen["liability_cap"] {
		fs = append(fs, Flag{"cap", "no cap found"})
	}
	return fs
}
