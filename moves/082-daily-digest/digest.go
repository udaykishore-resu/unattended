// Package digest picks fresh notes and resurfaces
// old ones at 7, 30 and 90 days.
//
// Move 82: personal knowledge base with a daily
// digest.
package digest

import (
	"slices"
	"time"
)

// Note is one markdown file. Created comes from front
// matter, since syncing resets file dates.
type Note struct {
	Path              string
	Created, Modified time.Time
}

// Resurface old notes at 7, 30 and 90 days.
var ages = []int{7, 30, 90}

func Pick(notes []Note, now time.Time) (
	fresh, old []Note) {
	day := 24 * time.Hour
	for _, n := range notes {
		if now.Sub(n.Modified) < day {
			fresh = append(fresh, n)
			continue
		}
		d := int(now.Sub(n.Created) / day)
		if slices.Contains(ages, d) {
			old = append(old, n)
		}
	}
	return fresh, old
}
