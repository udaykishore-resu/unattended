package allowlist

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// A tool that arrives from an MCP server or a plugin is
// registered with the executor but not on the list, so
// it never runs.
func TestRegisteredToolIsNotAllowedTool(t *testing.T) {
	ran := map[string]int{}
	handler := func(name string) Tool {
		return func(context.Context, json.RawMessage) (string, error) {
			ran[name]++
			return "ok", nil
		}
	}
	e := &Executor{Tools: map[string]Tool{
		"get_shipment": handler("get_shipment"),
		"set_status":   handler("set_status"),
	}}
	// Six weeks later, an MCP server adds cancel_load.
	e.Tools["cancel_load"] = handler("cancel_load")

	ctx := context.Background()
	if _, err := e.Do(ctx, llm.ToolCall{Name: "cancel_load",
		Input: json.RawMessage(`{"id":"HF-1"}`)}); err == nil {
		t.Fatal("cancel_load ran")
	}
	if _, err := e.Do(ctx, llm.ToolCall{Name: "set_status",
		Input: json.RawMessage(`{"status":"cancelled"}`)}); err == nil {
		t.Fatal("set_status cancelled ran")
	}
	if out, err := e.Do(ctx, llm.ToolCall{Name: "set_status",
		Input: json.RawMessage(`{"status":"delivered"}`)}); err != nil || out != "ok" {
		t.Fatalf("allowed call: %q %v", out, err)
	}
	if ran["cancel_load"] != 0 || ran["set_status"] != 1 {
		t.Fatalf("ran = %v", ran)
	}
}

// An ownership check that times out is a denial.
func TestCheckErrorFailsClosed(t *testing.T) {
	ran := 0
	Allowed["refund_note"] = func(json.RawMessage) error {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		<-ctx.Done()
		return ctx.Err() // the Postgres lookup timed out
	}
	defer delete(Allowed, "refund_note")
	e := &Executor{Tools: map[string]Tool{
		"refund_note": func(context.Context, json.RawMessage) (string, error) {
			ran++
			return "ok", nil
		}}}
	_, err := e.Do(context.Background(), llm.ToolCall{Name: "refund_note",
		Input: json.RawMessage(`{}`)})
	if !errors.Is(err, context.DeadlineExceeded) || ran != 0 {
		t.Fatalf("err = %v, ran = %d", err, ran)
	}
	if !strings.HasPrefix(err.Error(), "denied:") {
		t.Fatalf("err = %v", err)
	}
}
