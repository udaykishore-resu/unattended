package secrets

import (
	"context"
	"errors"
	"regexp"

	"github.com/udaykishore-resu/unattended/llm"
)

// keyShapes is the guard's list: your vendors'
// prefixes, cloud access key IDs and private key
// headers. Described in the book's prose; extend it
// per vendor and version it like a prompt.
var keyShapes = regexp.MustCompile(
	`\b(?:sk|rk|pk)_(?:live|test)_[A-Za-z0-9]{8,}` +
		`|\b(?:AKIA|ASIA)[0-9A-Z]{16}\b` +
		`|-----BEGIN [A-Z ]*PRIVATE KEY-----` +
		`|\bgh[pousr]_[A-Za-z0-9]{36}\b`)

var ErrKeyInPrompt = errors.New(
	"secrets: key-shaped text in prompt; call refused")

// Guard wraps the shared llm.Model. It checks every
// field the provider would see and refuses the call on
// a match, so a refused prompt reaches neither the
// provider nor any wrapper (trace, cache) inside it.
type Guard struct{ Next llm.Model }

func (g Guard) Complete(ctx context.Context,
	r llm.Request) (llm.Response, error) {
	fields := []string{r.System}
	for _, t := range r.Tools {
		fields = append(fields, t.Name, t.Description,
			string(t.InputSchema))
	}
	for _, m := range r.Messages {
		fields = append(fields, m.Content)
		for _, c := range m.ToolCalls {
			fields = append(fields, string(c.Input))
		}
	}
	for _, f := range fields {
		if keyShapes.MatchString(f) {
			return llm.Response{}, ErrKeyInPrompt
		}
	}
	return g.Next.Complete(ctx, r)
}
