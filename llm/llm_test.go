package llm_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// FakeModel must satisfy Model.
var _ llm.Model = (*llm.FakeModel)(nil)

func TestFakeReplaysInOrder(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{
		{Text: "one"}, {Text: "two"},
	}}
	ctx := context.Background()
	for _, want := range []string{"one", "two"} {
		got, err := f.Complete(ctx, llm.Request{})
		if err != nil || got.Text != want {
			t.Fatalf("got %q, %v; want %q", got.Text, err, want)
		}
	}
	if _, err := f.Complete(ctx, llm.Request{}); !errors.Is(err, llm.ErrNoResponses) {
		t.Fatalf("third call: got %v, want ErrNoResponses", err)
	}
	if len(f.Calls) != 3 {
		t.Fatalf("recorded %d calls, want 3", len(f.Calls))
	}
}

func TestFakeFillsStopReason(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{
		{Text: "done"},
		{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "get_load"}}},
		{Text: "cut", StopReason: llm.StopMaxTokens},
	}}
	want := []llm.StopReason{llm.StopEnd, llm.StopToolUse, llm.StopMaxTokens}
	for i, w := range want {
		got, _ := f.Complete(context.Background(), llm.Request{})
		if got.StopReason != w {
			t.Errorf("response %d: stop %q, want %q", i, got.StopReason, w)
		}
	}
}

func TestFakeCopiesHistory(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{}, {}}}
	req := llm.Request{Messages: make([]llm.Message, 1, 4)}
	req.Messages[0] = llm.Message{Role: "user", Content: "hi"}
	f.Complete(context.Background(), req)
	// An agent loop appends to (and may overwrite) the same backing array.
	req.Messages = append(req.Messages, llm.Message{Role: "assistant"})
	req.Messages[0].Content = "changed"
	f.Complete(context.Background(), req)
	if got := f.Calls[0].Messages; len(got) != 1 || got[0].Content != "hi" {
		t.Fatalf("first recorded call changed: %+v", got)
	}
}

func TestFakeErr(t *testing.T) {
	boom := errors.New("boom")
	f := &llm.FakeModel{Err: boom}
	if _, err := f.Complete(context.Background(), llm.Request{}); !errors.Is(err, boom) {
		t.Fatalf("got %v, want boom", err)
	}
}

func TestFakeConcurrent(t *testing.T) {
	f := &llm.FakeModel{}
	for range 50 {
		f.Responses = append(f.Responses, llm.Response{Text: "x"})
	}
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if _, err := f.Complete(context.Background(), llm.Request{}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if len(f.Calls) != 50 {
		t.Fatalf("calls = %d", len(f.Calls))
	}
}

// The recorded history is a deep copy: editing a sent
// tool call's input in place does not change it.
func TestFakeCopiesToolCallsDeep(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{}}}
	in := []byte(`{"load_id":"HF-90412"}`)
	msgs := []llm.Message{{Role: "assistant",
		ToolCalls: []llm.ToolCall{{ID: "c1", Name: "get_load", Input: in}}}}
	f.Complete(context.Background(), llm.Request{Messages: msgs})
	msgs[0].ToolCalls[0].Name = "set_eta"
	in[2] = 'X'
	got := f.Calls[0].Messages[0].ToolCalls[0]
	if got.Name != "get_load" || string(got.Input) != `{"load_id":"HF-90412"}` {
		t.Fatalf("recorded call changed: %s %s", got.Name, got.Input)
	}
}

// A nil history stays nil, so tests comparing requests
// with reflect.DeepEqual see what they sent.
func TestFakeKeepsNilHistory(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{}}}
	f.Complete(context.Background(), llm.Request{})
	if f.Calls[0].Messages != nil {
		t.Fatalf("messages = %#v, want nil", f.Calls[0].Messages)
	}
}
