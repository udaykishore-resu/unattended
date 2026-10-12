package caching

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func withSchemas(schema, toolSchema, callInput string) llm.Request {
	r := req("classify this")
	r.Schema = json.RawMessage(schema)
	r.Tools = []llm.Tool{{Name: "get_load", InputSchema: json.RawMessage(toolSchema)}}
	r.Messages = append(r.Messages, llm.Message{Role: "assistant",
		ToolCalls: []llm.ToolCall{{ID: "c1", Name: "get_load", Input: json.RawMessage(callInput)}}})
	return r
}

// Semantically identical JSON in a different key
// order or spacing gives one key.
func TestKeyIgnoresJSONKeyOrderAndSpacing(t *testing.T) {
	a := withSchemas(
		`{"type":"object","properties":{"queue":{"type":"string"},"conf":{"type":"string"}}}`,
		`{"type":"object","required":["id"]}`,
		`{"id":"HF-1","verbose":false}`)
	b := withSchemas(
		`{ "properties": { "conf": {"type":"string"}, "queue": {"type": "string"} }, "type": "object" }`,
		`{"required":["id"], "type":"object"}`,
		`{"verbose":false, "id":"HF-1"}`)
	if key(t, "acme", "v7", a) != key(t, "acme", "v7", b) {
		t.Fatal("same meaning, different key")
	}
	// A real difference still changes the key.
	c := withSchemas(`{"type":"object"}`, `{"type":"object","required":["id"]}`,
		`{"id":"HF-2","verbose":false}`)
	if key(t, "acme", "v7", a) == key(t, "acme", "v7", c) {
		t.Fatal("different input, same key")
	}
}

// Schemas built from Go maps hash the same every run.
func TestMapBuiltSchemasAreStable(t *testing.T) {
	first := ""
	for range 50 {
		props := map[string]any{}
		for i := range 20 {
			props[fmt.Sprint("f", i)] = map[string]any{"type": "string"}
		}
		raw, _ := json.Marshal(map[string]any{"type": "object", "properties": props})
		r := req("x")
		r.Schema = raw
		k := key(t, "acme", "v7", r)
		if first == "" {
			first = k
		} else if k != first {
			t.Fatal("unstable key")
		}
	}
}

// Large integers survive canonicalisation exactly.
func TestBigNumbersKept(t *testing.T) {
	a := withSchemas(`{}`, `{}`, `{"id":12345678901234567890}`)
	b := withSchemas(`{}`, `{}`, `{"id":12345678901234567891}`)
	if key(t, "acme", "v7", a) == key(t, "acme", "v7", b) {
		t.Fatal("numbers rounded together")
	}
}

// canon works on a copy: the caller's request is not
// rewritten.
func TestCallerRequestUntouched(t *testing.T) {
	r := withSchemas(`{ "b":1, "a":2 }`, `{ "b":1, "a":2 }`, `{ "b":1, "a":2 }`)
	key(t, "acme", "v7", r)
	if string(r.Schema) != `{ "b":1, "a":2 }` ||
		string(r.Tools[0].InputSchema) != `{ "b":1, "a":2 }` ||
		string(r.Messages[1].ToolCalls[0].Input) != `{ "b":1, "a":2 }` {
		t.Fatal("caller's request was modified")
	}
}

func TestInvalidRawJSONIsAnError(t *testing.T) {
	r := req("x")
	r.Schema = json.RawMessage(`{"type":`)
	if _, err := CacheKey("acme", "small", "v7", r); err == nil {
		t.Fatal("invalid schema hashed")
	}
}
