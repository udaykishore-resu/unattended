// Package taint drops write tools and open reads
// from a run once it has read untrusted text.
//
// Move 93: treat every retrieved document as
// untrusted input.
package taint

import "github.com/udaykishore-resu/unattended/llm"

type Run struct {
	tainted bool
	tools   []llm.Tool
	closed  map[string]bool // reads: IDs, enums only
}

// Add is called per message. Taint never clears.
func (r *Run) Add(untrusted bool) {
	r.tainted = r.tainted || untrusted
}

// Allows runs again in the executor: a model can
// name a tool it was not offered.
func (r *Run) Allows(name string) bool {
	return !r.tainted || r.closed[name]
}

// Tools: once tainted, only closed reads remain.
func (r *Run) Tools() []llm.Tool {
	var out []llm.Tool
	for _, t := range r.tools {
		if r.Allows(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// NewRun starts a clean run with its full tool list
// and the names of its closed reads. Minimal
// constructor; the book shows only the methods.
func NewRun(tools []llm.Tool, closed ...string) *Run {
	r := &Run{tools: tools, closed: map[string]bool{}}
	for _, c := range closed {
		r.closed[c] = true
	}
	return r
}

// Tainted reports whether outside text has entered.
func (r *Run) Tainted() bool { return r.tainted }

// Text is bytes plus the label of where they came
// from. The label travels with chunks, summaries and
// handoff notes, so taint survives each step.
type Text struct {
	S         string
	Untrusted bool
}

// Read adds t to the run and returns its body.
func (r *Run) Read(t Text) string {
	r.Add(t.Untrusted)
	return t.S
}

// Out labels text the run wrote. Anything a tainted
// run writes, such as a summary, is untrusted for
// whoever reads it next.
func (r *Run) Out(s string) Text {
	return Text{S: s, Untrusted: r.tainted}
}
