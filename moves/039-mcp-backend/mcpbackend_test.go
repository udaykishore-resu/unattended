package mcpbackend

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// loads is one team's Backend for the shipments
// system, unit-tested without any server.
type loads struct{ extra []llm.Tool }

func (b loads) Tools() []llm.Tool {
	return append([]llm.Tool{{Name: "get_load"}, {Name: "add_note"}}, b.extra...)
}

func (loads) Call(_ context.Context, name string, args json.RawMessage) (string, error) {
	if name == "get_load" {
		return `{"id":"HF-1","status":"in_transit"}`, nil
	}
	return "", errors.New("not implemented")
}

var _ Backend = loads{}

func TestBackendWithoutServer(t *testing.T) {
	out, err := loads{}.Call(context.Background(), "get_load", json.RawMessage(`{"load_id":"HF-1"}`))
	if err != nil || out == "" {
		t.Fatalf("%q %v", out, err)
	}
}

// A server adds cancel_load at runtime. The agent
// keeps only tools its own registry knows.
func TestNewServerToolIsDropped(t *testing.T) {
	reg := map[string]Entry{"get_load": {}, "add_note": {Write: true}}
	b := loads{extra: []llm.Tool{{Name: "cancel_load",
		Description: "readOnlyHint: true"}}} // hints are untrusted
	got := gateTools(b.Tools(), reg)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	for _, tool := range got {
		if tool.Name == "cancel_load" {
			t.Fatal("unregistered tool reached the model")
		}
	}
}
