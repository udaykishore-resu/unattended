// Package caching builds a response-cache key that
// covers everything that changes the answer.
//
// Move 57: response caching and prompt caching for
// shared prefixes.
package caching

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

func CacheKey(tenant, model, ver string,
	r llm.Request) (string, error) {
	b, err := json.Marshal(struct {
		Tenant, Model, Ver string
		Req                llm.Request
	}{tenant, model, ver, canon(r)})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// canon returns a copy of r whose raw JSON fields
// (the output schema, each tool's input schema and
// each tool call's input) are re-encoded with sorted
// keys and no spaces, so two schemas built in a
// different key order hash the same. Struct fields
// already marshal in a fixed order. Raw JSON that
// does not parse is left as is; json.Marshal then
// reports it. Tool order is kept: it changes what
// the model sees, so sort tools where you build
// them (SortTools).
func canon(r llm.Request) llm.Request {
	r.Schema = canonJSON(r.Schema)
	r.Tools = slices.Clone(r.Tools)
	for i := range r.Tools {
		r.Tools[i].InputSchema =
			canonJSON(r.Tools[i].InputSchema)
	}
	r.Messages = slices.Clone(r.Messages)
	for i, m := range r.Messages {
		m.ToolCalls = slices.Clone(m.ToolCalls)
		for j := range m.ToolCalls {
			m.ToolCalls[j].Input =
				canonJSON(m.ToolCalls[j].Input)
		}
		r.Messages[i] = m
	}
	return r
}

func canonJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber() // keep 64-bit IDs exact
	var v any
	if d.Decode(&v) != nil || d.More() {
		return raw
	}
	b, err := json.Marshal(v) // map keys sorted
	if err != nil {
		return raw
	}
	return b
}

// SortTools keeps the cacheable prefix stable: tools
// built from a Go map come out in random order.
// Named in the "Breaks when" list; minimal.
func SortTools(ts []llm.Tool) {
	slices.SortFunc(ts, func(a, b llm.Tool) int {
		return strings.Compare(a.Name, b.Name)
	})
}
