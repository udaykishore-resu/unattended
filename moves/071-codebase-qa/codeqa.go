// Package codeqa checks every [path:line@sha]
// citation in an answer against the chunks that were
// actually retrieved.
//
// Move 71: codebase Q&A over internal repos.
package codeqa

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var cite = regexp.MustCompile(
	`\[([\w./-]+):(\d+)@([0-9a-f]{7,40})\]`)

func Check(answer string, got []Chunk) error {
	ms := cite.FindAllStringSubmatch(answer, -1)
	if len(ms) == 0 {
		return fmt.Errorf("no citations")
	}
	for _, m := range ms {
		line, _ := strconv.Atoi(m[2])
		if !covered(got, m[1], line, m[3]) {
			return fmt.Errorf("bad citation %s", m[0])
		}
	}
	return nil
}

// Chunk is one function, type or config file, with
// the commit it was indexed at.
type Chunk struct {
	Path, Commit string
	Start, End   int // line range
	Text         string
}

// covered checks that some retrieved chunk has that
// path, a commit starting with that SHA, and a line
// range holding the line.
func covered(got []Chunk, path string, line int,
	sha string) bool {
	for _, c := range got {
		if c.Path == path &&
			strings.HasPrefix(c.Commit, sha) &&
			c.Start <= line && line <= c.End {
			return true
		}
	}
	return false
}
