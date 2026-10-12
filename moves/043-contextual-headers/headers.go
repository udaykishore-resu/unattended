// Package headers puts a short header on every chunk
// before it is indexed.
//
// Move 43: contextual headers on every chunk.
package headers

import (
	"fmt"
	"strings"
	"time"
)

type Chunk struct {
	ID, DocTitle, Text string
	Path               []string
	Updated            time.Time // reviewed date
}

func Header(c Chunk) string {
	return fmt.Sprintf("[%s > %s | updated %s]",
		c.DocTitle, strings.Join(c.Path, " > "),
		c.Updated.Format("2006-01-02"))
}

const situatePrompt = `You will see a full document
and one chunk from it. Write one or two sentences
that place the chunk: who or what it is about, and
which part of the document it sits in. Do not
repeat the chunk. Do not add facts that are not in
the document. Output only the sentences.`

// IndexText is what goes to the embedder and the
// keyword index: header, model sentences, chunk.
// Users see only c.Text, stored apart. Minimal
// helper for the prose.
func IndexText(c Chunk, situated string) string {
	return Header(c) + "\n" + situated + "\n" + c.Text
}

// CheckMeta rejects chunks whose fixed header would
// carry missing or impossible metadata: no title, no
// heading trail, or a reviewed date that is zero or
// in the future. The fixed layer adds no invented
// claims, but it is only as right as this metadata.
func CheckMeta(c Chunk, now time.Time) error {
	switch {
	case strings.TrimSpace(c.DocTitle) == "":
		return fmt.Errorf("chunk %s: no doc title", c.ID)
	case len(c.Path) == 0:
		return fmt.Errorf("chunk %s: no heading trail", c.ID)
	case c.Updated.IsZero() || c.Updated.After(now):
		return fmt.Errorf("chunk %s: bad reviewed date %s",
			c.ID, c.Updated.Format("2006-01-02"))
	}
	return nil
}
