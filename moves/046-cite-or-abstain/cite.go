// Package cite rejects answers whose quotes are not
// in the chunks sent with this request.
//
// Move 46: cite or abstain.
package cite

import (
	"errors"
	"fmt"
	"strings"
)

// An Answer holds the Abstain flag and its
// Citations, each a ChunkID and a Quote.
type Answer struct {
	Abstain   bool       `json:"abstain"`
	Text      string     `json:"answer"`
	Citations []Citation `json:"citations"`
}

type Citation struct {
	ChunkID string `json:"chunk_id"`
	Quote   string `json:"quote"`
}

// Check rejects answers whose citations don't
// match a chunk we sent, word for word.
func Check(a Answer, sent map[string]string) error {
	if a.Abstain {
		return nil
	}
	if len(a.Citations) == 0 {
		return errors.New("answer has no citation")
	}
	for _, c := range a.Citations {
		text, ok := sent[c.ChunkID]
		if !ok {
			return fmt.Errorf("unknown chunk %q", c.ChunkID)
		}
		q := norm(c.Quote)
		if len(q) < 20 || !strings.Contains(norm(text), q) {
			return fmt.Errorf("bad quote for %s", c.ChunkID)
		}
	}
	return nil
}

// norm lowercases and folds whitespace.
func norm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// AbstainInCode is the code exit before any model
// call: an empty rerank, or a best score of 1 (on
// Move 45's 0-3 scale). Described in prose; minimal.
func AbstainInCode(bestScore, kept int) bool {
	return kept == 0 || bestScore <= 1
}
