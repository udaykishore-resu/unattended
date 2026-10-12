package structured

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestTextAndToolCallBothRead(t *testing.T) {
	native := llm.Response{Text: `{"category":"wismo"}`, StopReason: llm.StopEnd}
	forced := llm.Response{StopReason: llm.StopToolUse, ToolCalls: []llm.ToolCall{
		{ID: "c1", Name: "classify", Input: json.RawMessage(`{"category":"claim"}`)}}}
	for _, r := range []llm.Response{native, forced} {
		b, err := output(r)
		if err != nil || !json.Valid(b) {
			t.Fatalf("output(%+v) = %s, %v", r, b, err)
		}
	}
}

func TestStopReasonsAreTypedErrors(t *testing.T) {
	cut := llm.Response{Text: `{"categ`, StopReason: llm.StopMaxTokens}
	if _, err := output(cut); !errors.Is(err, ErrCutOff) {
		t.Fatalf("cut off: %v", err)
	}
	ref := llm.Response{Text: "I can't help with that.", StopReason: llm.StopRefusal}
	if _, err := output(ref); !errors.Is(err, ErrRefused) {
		t.Fatalf("refusal: %v", err)
	}
	// Fail closed: a stop value the adapter could not
	// name (StopOther) or left empty is not trusted,
	// even when the text happens to parse.
	for _, s := range []llm.StopReason{llm.StopOther, ""} {
		odd := llm.Response{Text: `{"category":"wismo"}`, StopReason: s}
		if _, err := output(odd); !errors.Is(err, ErrStopped) {
			t.Fatalf("stop %q: %v", s, err)
		}
	}
}

// The request carries the schema in Request.Schema,
// never as words in the prompt. This checks the
// caller's side of the contract only: whether a real
// adapter turns Schema into the vendor's native mode
// needs that adapter's own contract test against a
// recorded request (adapters/README.md, test 7).
func TestSchemaTravelsInRequest(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{}`}}}
	resp, err := f.Complete(context.Background(), llm.Request{Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output(resp); err != nil {
		t.Fatal(err)
	}
	if string(f.Calls[0].Schema) != string(schema) {
		t.Fatal("schema not sent")
	}
}

// Shape is all the provider promises. A reply that
// passes output and decodes cleanly can still break a
// business rule; the caller checks those after decode.
func TestValidShapeStillNeedsBusinessChecks(t *testing.T) {
	type verdict struct {
		Category string `json:"category"`
		LoadID   string `json:"load_id"`
	}
	r := llm.Response{Text: `{"category":"refund_now","load_id":""}`, StopReason: llm.StopEnd}
	b, err := output(r)
	if err != nil {
		t.Fatal(err)
	}
	var v verdict
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{"wismo": true, "claim": true, "invoice": true, "other": true}
	if known[v.Category] && v.LoadID != "" {
		t.Fatal("this reply was meant to fail the business checks")
	}
}
