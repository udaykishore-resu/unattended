// Package skeleton gives every prompt one shape:
// role, numbered rules, examples, fenced input.
//
// Move 13: a fixed skeleton.
package skeleton

import (
	"fmt"
	"strings"
)

// Skeleton is the one shape every prompt follows.
type Skeleton struct {
	Role     string   // who the model is, one line
	Rules    []string // numbered, one idea each
	Examples string   // pre-rendered, may be empty
}

func (s Skeleton) System() string {
	var b strings.Builder
	b.WriteString(s.Role + "\n\nRules:\n")
	for i, r := range s.Rules {
		fmt.Fprintf(&b, "%d. %s\n", i+1, r)
	}
	if s.Examples != "" {
		b.WriteString("\nExamples:\n" + s.Examples)
	}
	return b.String()
}

// User is the "sibling method" the book mentions:
// it wraps the input in <input> tags for the user
// turn. Minimal implementation.
func (s Skeleton) User(input string) string {
	return "<input>\n" + input + "\n</input>"
}
