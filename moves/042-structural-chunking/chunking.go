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

// SplitMarkdown cuts at headings, keeping trails.
func SplitMarkdown(doc string) []Section {
	out := []Section{{}} // text before any heading
	var trail []string
	var lvls []int // level of each trail entry
	fence := false
	for _, line := range strings.Split(doc, "\n") {
		if isFence(line) {
			fence = !fence
		}
		lvl := headingLevel(line)
		if lvl == 0 || fence {
			out[len(out)-1].Body += line + "\n"
			continue
		}
		n := len(lvls) // pop same or deeper levels
		for n > 0 && lvls[n-1] >= lvl {
			n--
		}
		lvls = append(lvls[:n], lvl)
		trail = append(trail[:n],
			strings.TrimSpace(line[lvl:]))
		out = append(out,
			Section{Path: slices.Clone(trail)})
	}
	return slices.DeleteFunc(out, func(s Section) bool {
		return strings.TrimSpace(s.Body) == ""
	})
}

// headingLevel returns 1 to 6 for a heading line
// with text, else 0. Seven or more #, a missing
// space, or an empty title is not a heading.
func headingLevel(line string) int {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n == len(line) || line[n] != ' ' {
		return 0
	}
	if strings.TrimSpace(line[n:]) == "" {
		return 0
	}
	return n
}

// isFence reports whether line opens or closes a
// code fence: ``` or ~~~ after at most three spaces.
func isFence(line string) bool {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return false
	}
	return strings.HasPrefix(t, "```") ||
		strings.HasPrefix(t, "~~~")
}
