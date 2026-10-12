// Package digest picks fresh notes and resurfaces
// old ones at 7, 30 and 90 days.
//
// Move 82: personal knowledge base with a daily
// digest.
package digest

import (
	"fmt"
	"slices"
	"strings"
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

// Link is the model's answer for one old note: one
// shared idea and a phrase quoted from each side, or
// NoLink.
type Link struct {
	Idea                 string
	OldPhrase, NewPhrase string
	NoLink               bool
}

// Keep checks both phrases with a plain string search
// (whitespace and case folded) and drops a link whose
// phrases are missing or empty.
func Keep(l Link, oldText, newText string) bool {
	if l.NoLink {
		return false
	}
	return found(l.OldPhrase, oldText) &&
		found(l.NewPhrase, newText)
}

func found(phrase, text string) bool {
	p := fold(phrase)
	return p != "" && strings.Contains(fold(text), p)
}

func fold(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// Line renders one digest item with its path and
// created date, so the reader can tell an old note
// from a current one at a glance.
func Line(n Note, gist string) string {
	return fmt.Sprintf("%s (created %s): %s", n.Path,
		n.Created.Format(time.DateOnly), gist)
}
