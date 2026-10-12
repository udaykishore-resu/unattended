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
}

// The request carries the schema in Request.Schema,
// never as words in the prompt.
func TestSchemaTravelsInRequest(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{}`}}}
	resp, _ := f.Complete(context.Background(), llm.Request{Schema: schema})
	if _, err := output(resp); err != nil {
		t.Fatal(err)
	}
	if string(f.Calls[0].Schema) != string(schema) {
		t.Fatal("schema not sent")
	}
}
