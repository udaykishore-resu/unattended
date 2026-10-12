package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// The guard refuses a key shape in any prompt field:
// System, a user turn, a tool result carrying an
// exception message, a tool call's input.
func TestGuardRefusesKeyShapes(t *testing.T) {
	leaks := map[string]llm.Request{
		"system": {System: "Use bearer token sk_live_4dQx9mAbCdEf for the portal."},
		"user": {Messages: []llm.Message{{Role: "user",
			Content: "my key is AKIAIOSFODNN7EXAMPLE"}}},
		"tool result with exception": {Messages: []llm.Message{{Role: "tool",
			ToolCallID: "1", Content: "error: auth failed for sk_test_ZZZZ1111aaaa"}}},
		"tool call input": {Messages: []llm.Message{{Role: "assistant",
			ToolCalls: []llm.ToolCall{{ID: "1", Name: "get_carrier_status",
				Input: json.RawMessage(`{"path":"/v1/status/1","token":"sk_live_4dQx9mAbCdEf"}`)}}}}},
		"private key": {Messages: []llm.Message{{Role: "user",
			Content: "-----BEGIN RSA PRIVATE KEY-----\nMIIE..."}}},
		"tool description": {Tools: []llm.Tool{{Name: "x",
			Description: "auth with ghp_" + strings.Repeat("a", 36)}}},
	}
	for name, req := range leaks {
		fake := &llm.FakeModel{Responses: []llm.Response{{Text: "ok"}}}
		_, err := Guard{Next: fake}.Complete(context.Background(), req)
		if !errors.Is(err, ErrKeyInPrompt) || len(fake.Calls) != 0 {
			t.Errorf("%s: err = %v, provider saw %d calls", name, err, len(fake.Calls))
		}
	}
	// Clean prompts pass, including load IDs that look a bit like keys.
	fake := &llm.FakeModel{Responses: []llm.Response{{Text: "ok"}}}
	_, err := Guard{Next: fake}.Complete(context.Background(), llm.Request{
		System:   "You are the carrier portal agent.",
		Messages: []llm.Message{{Role: "user", Content: "Status of HF-88213, skip_live_check please"}}})
	if err != nil || len(fake.Calls) != 1 {
		t.Fatalf("clean prompt refused: %v", err)
	}
}

// End to end: the model asks for a status, the tool reads
// the token from the environment at call time, the portal
// echoes it, and nothing the model sees holds it. Errors
// returned to the model do not hold it either.
func TestTokenNeverReachesTheModel(t *testing.T) {
	var hits atomic.Int32
	srv := portal(t, &hits)
	defer srv.Close()
	in := Integration{Base: srv.URL, Env: "PORTAL_TOKEN", Paths: []string{"/v1/status/"}}

	// Fetched just in time: set after the Integration exists.
	t.Setenv("PORTAL_TOKEN", tok)

	fake := &llm.FakeModel{Responses: []llm.Response{
		{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "get_carrier_status",
			Input: json.RawMessage(`{"path":"/v1/status/HF-1"}`)}}},
		{Text: "The portal rejected the request."},
	}}
	m := Guard{Next: fake}
	msgs := []llm.Message{{Role: "user", Content: "Check HF-1 with the carrier."}}
	req := llm.Request{System: "You check carrier status.", Messages: msgs}
	resp, err := m.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	var in2 struct{ Path string }
	json.Unmarshal(resp.ToolCalls[0].Input, &in2)
	body, terr := in.Get(context.Background(), in2.Path)
	result := body
	if terr != nil {
		result = "error: " + terr.Error() + "\n" + body
	}
	req.Messages = append(req.Messages,
		llm.Message{Role: "assistant", ToolCalls: resp.ToolCalls},
		llm.Message{Role: "tool", ToolCallID: "c1", Content: result})
	if _, err := m.Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	for _, c := range fake.Calls {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "4dQ") {
			t.Fatalf("token reached the model: %s", b)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("portal hits = %d", hits.Load())
	}
}

// A transport failure's message names the URL, never the
// token: the token travels only in the header.
func TestTransportErrorHoldsNoToken(t *testing.T) {
	t.Setenv("PORTAL_TOKEN", tok)
	in := Integration{Base: "http://127.0.0.1:1", Env: "PORTAL_TOKEN",
		Paths: []string{"/v1/status/"}}
	_, err := in.Get(context.Background(), "/v1/status/HF-1")
	if err == nil || strings.Contains(err.Error(), "4dQ") {
		t.Fatalf("err = %v", err)
	}
}

// Encoded dot-dot is decoded by url.Parse and refused.
func TestEncodedTraversalRefused(t *testing.T) {
	t.Setenv("PORTAL_TOKEN", tok)
	in := Integration{Base: "http://127.0.0.1:1", Env: "PORTAL_TOKEN",
		Paths: []string{"/v1/status/"}}
	for _, rel := range []string{"/v1/status/%2e%2e/admin", "/v1/status/.%2E/admin",
		"/v1/status/..%2fadmin"} {
		if _, err := in.Get(context.Background(), rel); err == nil ||
			err.Error() != "path not allowed" {
			t.Errorf("%q: %v", rel, err)
		}
	}
}
