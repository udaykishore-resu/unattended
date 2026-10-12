// Package clauses runs the contract playbook as
// code over model-extracted clauses.
//
// Move 78: contract clause extraction and red-flag
// notes.
package clauses

import (
	"regexp"
	"strconv"
	"strings"
)

// Clause is one extracted clause: kind from an enum,
// exact quote, page, and any day count.
type Clause struct {
	Kind  string `json:"kind"`
	Quote string `json:"quote"`
	Page  int    `json:"page"`
	Days  *int   `json:"days"`
}

// Flag is one note for the named reviewer. There is
// no field that marks a contract approved or cleared:
// the output can only add work for the reviewer.
type Flag struct{ Code, Msg string }

func Review(cs []Clause, pages []string) []Flag {
	var fs []Flag
	seen := map[string]bool{}
	for _, c := range cs {
		if c.Page < 1 || c.Page > len(pages) ||
			strings.TrimSpace(c.Quote) == "" ||
			!strings.Contains(pages[c.Page-1], c.Quote) {
			fs = append(fs, Flag{"quote", "not on page"})
			continue
		}
		seen[c.Kind] = true
		if c.Kind != "payment_terms" {
			continue
		}
		d, ok := quoteDays(c.Quote) // Go reads it
		if ok && d > 45 {
			fs = append(fs, Flag{"pay", "over 45 days"})
		}
		if !ok || c.Days == nil || *c.Days != d {
			fs = append(fs, Flag{"pay", "check days"})
		}
	}
	if !seen["liability_cap"] {
		miss := Flag{"cap", "no cap found"}
		fs = append([]Flag{miss}, fs...)
	}
	return fs
}

var dayCount = regexp.MustCompile(
	`(?i)\bnet\s*(\d+)\b|\b(\d+)\)?\s*` +
		`(?:calendar\s+|business\s+)?days?\b`)

// quoteDays reads the day count from the quote
// itself: "sixty (60) days", "60 calendar days",
// "Net 30". With several counts it returns the
// largest, the conservative pick for a maximum-days
// rule. Spelled-out numbers alone ("sixty days")
// return false, which Review turns into a flag.
func quoteDays(q string) (int, bool) {
	best, found := 0, false
	for _, m := range dayCount.FindAllStringSubmatch(q, -1) {
		s := m[1] + m[2]
		n, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		if !found || n > best {
			best, found = n, true
		}
	}
	return best, found
}

// Thin returns the 1-based pages with fewer than min
// non-space characters. A scan with no text layer
// shows up here; zero clauses on thin pages is a
// failure, never a clean bill.
func Thin(pages []string, min int) []int {
	var out []int
	for i, p := range pages {
		n := len(strings.Join(strings.Fields(p), ""))
		if n < min {
			out = append(out, i+1)
		}
	}
	return out
}
