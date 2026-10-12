// Package briefs checks that every [Sn] tag in a
// brief points at a real source.
//
// Move 76: one-page research briefs.
package briefs

import (
	"regexp"
	"strconv"
)

var cite = regexp.MustCompile(`\[S(\d+)\]`)

// BadCites returns citations that point nowhere.
func BadCites(brief string, n int) []string {
	var bad []string
	for _, m := range cite.FindAllStringSubmatch(
		brief, -1) {
		k, _ := strconv.Atoi(m[1])
		if k < 1 || k > n {
			bad = append(bad, m[0])
		}
	}
	return bad
}
