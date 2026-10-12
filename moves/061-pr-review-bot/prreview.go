// Package prreview keeps only review findings that
// are on an added line, of an allowed kind, rated
// high, and quote the exact code.
//
// Move 61: PR review bot that only comments when
// it's sure.
package prreview

import "strings"

type Finding struct {
	File, Kind, Evidence string
	Line                 int
	Confidence           string // high|medium|low
}

type Added map[string]map[int]string

func Keep(fs []Finding, diff Added,
	kinds map[string]bool) []Finding {
	var out []Finding
	for _, f := range fs {
		text, ok := diff[f.File][f.Line]
		if ok && kinds[f.Kind] &&
			f.Confidence == "high" &&
			f.Evidence != "" &&
			strings.Contains(text, f.Evidence) {
			out = append(out, f)
		}
	}
	return out
}
