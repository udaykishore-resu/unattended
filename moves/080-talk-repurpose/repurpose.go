// Package repurpose drafts one piece per format from
// the approved outline, and flags quotes in a piece
// that the speaker never said, word for word.
//
// Move 80: one talk, five pieces of content.
package repurpose

import (
	"context"
	"regexp"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

var quoted = regexp.MustCompile(`["“]([^"“”]{12,})["”]`)

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

// Format is one output: its own system prompt (reader,
// opening story) and whether it is public.
type Format struct {
	Name, System string
	Public       bool
}

// Request builds the stage-two call. It takes the
// approved outline and the speaker's own samples, and
// has no transcript parameter: stage two cannot see
// anything the speaker did not approve.
func Request(f Format, outline string,
	samples []string) llm.Request {
	var b strings.Builder
	b.WriteString("<outline>\n" + outline + "\n</outline>\n")
	for _, s := range samples {
		b.WriteString("<sample>\n" + s + "\n</sample>\n")
	}
	return llm.Request{
		System:    f.System,
		Messages:  []llm.Message{{Role: "user", Content: b.String()}},
		MaxTokens: 2000,
	}
}

// Draft runs one format and returns the piece with
// any quotes the speaker never said. Every piece,
// flagged or not, goes to the speaker's review queue.
func Draft(ctx context.Context, m llm.Model, f Format,
	outline string, samples []string,
	transcript string) (piece string, invented []string, err error) {
	resp, err := m.Complete(ctx, Request(f, outline, samples))
	if err != nil {
		return "", nil, err
	}
	return resp.Text, Invented(resp.Text, transcript), nil
}
