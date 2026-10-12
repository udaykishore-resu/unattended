package toolerrors

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func tc(id, name, in string) llm.ToolCall {
	return llm.ToolCall{ID: id, Name: name, Input: json.RawMessage(in)}
}

// The stuck agent passed a pro number. A ToolError
// with a fix gets it unstuck in two calls.
func TestModelFixesItsCall(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{
		{ToolCalls: []llm.ToolCall{tc("c1", "get_check_call", `{"load_id":"4418302"}`)}},
		{ToolCalls: []llm.ToolCall{tc("c2", "get_check_call", `{"load_id":"HF-12345"}`)}},
		{Text: "Truck is 40 miles out."},
	}}
	run := func(_ context.Context, c llm.ToolCall) (string, error) {
		if strings.Contains(string(c.Input), "HF-") {
			return `{"miles_out":40}`, nil
		}
		return "", &ToolError{Code: "invalid_load_id",
			Msg: "'4418302' is not a load ID; load IDs look like HF-12345",
			Fix: "call find_load_by_ref with this number"}
	}
	got, err := Loop(context.Background(), f, llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "where is it?"}}}, run)
	if err != nil || got != "Truck is 40 miles out." {
		t.Fatalf("got %q, %v", got, err)
	}
	// Round-trip shape: user, assistant(c1), tool(c1), assistant(c2), tool(c2).
	msgs := f.Calls[2].Messages
	roles := []string{}
	for _, m := range msgs {
		roles = append(roles, m.Role+":"+m.ToolCallID)
	}
	if strings.Join(roles, " ") != "user: assistant: tool:c1 assistant: tool:c2" {
		t.Fatalf("history = %v", roles)
	}
	if !strings.Contains(msgs[2].Content, "find_load_by_ref") {
		t.Fatalf("model never saw the fix: %s", msgs[2].Content)
	}
}

// Raw Go errors carry hostnames and SQL. The model
// gets a fixed message instead.
func TestRawErrorsNeverReachTheModel(t *testing.T) {
	raw := errors.New("dial tcp db-prod-3.internal:5432: SELECT * FROM loads")
	got := toolResult("", raw)
	if strings.Contains(got, "db-prod") || !strings.Contains(got, `"code":"internal"`) {
		t.Fatalf("leaked: %s", got)
	}
	wrapped := toolResult("", errors.Join(errors.New("ctx"), &ToolError{Code: "not_found"}))
	if !strings.Contains(wrapped, `"code":"not_found"`) {
		t.Fatalf("wrapped ToolError lost: %s", wrapped)
	}
}

// Half a JSON argument must never execute.
func TestMaxTokensStopsTheRun(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{
		ToolCalls:  []llm.ToolCall{tc("c1", "add_note", `{"text":"carr`)},
		StopReason: llm.StopMaxTokens}}}
	ran := false
	_, err := Loop(context.Background(), f, llm.Request{},
		func(context.Context, llm.ToolCall) (string, error) { ran = true; return "", nil })
	if err == nil || ran {
		t.Fatalf("err=%v ran=%v", err, ran)
	}
}

func TestStepCap(t *testing.T) {
	f := &llm.FakeModel{}
	for range 20 {
		f.Responses = append(f.Responses, llm.Response{
			ToolCalls: []llm.ToolCall{tc("c", "get_load", `{}`)}})
	}
	_, err := Loop(context.Background(), f, llm.Request{},
		func(context.Context, llm.ToolCall) (string, error) { return "{}", nil })
	if err == nil || err.Error() != "step cap reached" {
		t.Fatalf("got %v", err)
	}
}
