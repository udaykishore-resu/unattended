package repair

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func call(id, input string) llm.Response {
	return llm.Response{ToolCalls: []llm.ToolCall{
		{ID: id, Name: "save_tender", Input: json.RawMessage(input)}}}
}

const good = `{"shipper_ref":"A","origin_zip":"45402",
  "dest_zip":"60601","pickup_date":"3/12","weight_lb":42000}`

func TestCodeFixesZipPlus4WithoutACall(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{call("c1",
		strings.Replace(good, `"45402"`, `"45402-1123"`, 1))}}
	tn, err := Extract(context.Background(), f, llm.Request{})
	if err != nil || *tn.OriginZip != "45402" || len(f.Calls) != 1 {
		t.Fatalf("tender=%+v err=%v calls=%d", tn, err, len(f.Calls))
	}
}

func TestOneRepairThenPass(t *testing.T) {
	bad := strings.Replace(good, `"45402"`, `"4540"`, 1)
	f := &llm.FakeModel{Responses: []llm.Response{
		call("c1", bad), call("c2", good)}}
	tn, err := Extract(context.Background(), f, llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "email"}}})
	if err != nil {
		t.Fatal(err)
	}
	if *tn.OriginZip != "45402" {
		t.Fatalf("tender = %+v", tn)
	}
	// The repair went back as the call's result: the
	// assistant turn with its ToolCalls, then one tool
	// message carrying that call's ID.
	msgs := f.Calls[1].Messages
	if len(msgs) != 3 || msgs[1].Role != "assistant" ||
		len(msgs[1].ToolCalls) != 1 || msgs[2].Role != "tool" ||
		msgs[2].ToolCallID != "c1" {
		t.Fatalf("repair turn shape wrong: %+v", msgs)
	}
	if !strings.Contains(msgs[2].Content, `"4540"`) ||
		!strings.Contains(msgs[2].Content, "use null") {
		t.Fatalf("repair note = %q", msgs[2].Content)
	}
}

// Wrong twice means reject. The fake has no third
// response, so a third call would fail the test.
func TestRejectAfterOneRepair(t *testing.T) {
	bad := strings.Replace(good, `"45402"`, `"4540"`, 1)
	f := &llm.FakeModel{Responses: []llm.Response{
		call("c1", bad), call("c2", bad)}}
	_, err := Extract(context.Background(), f, llm.Request{})
	if !errors.Is(err, ErrRejected) || len(f.Calls) != 2 {
		t.Fatalf("err=%v calls=%d", err, len(f.Calls))
	}
}

func TestNullIsAnAnswer(t *testing.T) {
	nulls := strings.Replace(good, `"45402"`, `null`, 1)
	f := &llm.FakeModel{Responses: []llm.Response{call("c1", nulls)}}
	tn, err := Extract(context.Background(), f, llm.Request{})
	if err != nil || tn.OriginZip != nil {
		t.Fatalf("tender=%+v err=%v", tn, err)
	}
}

func TestProseReplyGoesToReview(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: "see attached"}}}
	if _, err := Extract(context.Background(), f, llm.Request{}); !errors.Is(err, ErrNoToolCall) {
		t.Fatalf("got %v", err)
	}
}
