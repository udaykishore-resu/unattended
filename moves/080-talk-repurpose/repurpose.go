// Package repurpose flags quotes in a piece that the
// speaker never said, word for word.
//
// Move 80: one talk, five pieces of content.
package repurpose

import (
	"regexp"
	"strings"
)

var quoted = regexp.MustCompile(`"([^"]{12,})"`)

// Invented returns quotes that the speaker
// never said, word for word.
func Invented(piece, transcript string) []string {
	flat := strings.Join(
		strings.Fields(transcript), " ")
	var bad []string
	for _, m := range quoted.FindAllStringSubmatch(
		piece, -1) {
		q := strings.Join(strings.Fields(m[1]), " ")
		if !strings.Contains(flat, q) {
			bad = append(bad, m[1])
		}
	}
	return bad
}
