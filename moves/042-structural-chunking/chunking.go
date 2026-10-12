// Package chunking splits Markdown at headings and
// keeps each section's trail of parent headings.
//
// Move 42: chunk by document structure, not
// character count.
package chunking

import (
	"slices"
	"strings"
)

// A Section holds a heading Path and a Body.
type Section struct {
	Path []string
	Body string
}

// SplitMarkdown cuts at headings. Each section
// keeps its trail of parent headings.
func SplitMarkdown(doc string) []Section {
	out := []Section{{}} // text before any heading
	var trail []string
	for _, line := range strings.Split(doc, "\n") {
		lvl := headingLevel(line)
		if lvl > 0 {
			trail = append(trail[:min(lvl-1, len(trail))],
				strings.TrimSpace(line[lvl:]))
			out = append(out,
				Section{Path: slices.Clone(trail)})
			continue
		}
		out[len(out)-1].Body += line + "\n"
	}
	return slices.DeleteFunc(out, func(s Section) bool {
		return strings.TrimSpace(s.Body) == ""
	})
}

// headingLevel returns 1 to 6 for a heading line,
// else 0. Described in prose; minimal.
func headingLevel(line string) int {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n == len(line) || line[n] != ' ' {
		return 0
	}
	return n
}
