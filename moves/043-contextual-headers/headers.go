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
