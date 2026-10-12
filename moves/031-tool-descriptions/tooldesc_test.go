package tooldesc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

var findByRef = llm.Tool{Name: "find_load_by_ref",
	Description: "Find a load ID from a PO, BOL or pro number."}

func call(name string) llm.ToolCall {
	return llm.ToolCall{ID: "c", Name: name, Input: json.RawMessage(`{}`)}
}

// The description answers the five questions and is
// about 70 words, as the book says.
func TestDescriptionShape(t *testing.T) {
	d := GetLoad.Description
	if n := len(strings.Fields(d)); n < 60 || n > 80 {
		t.Fatalf("description is %d words", n)
	}
	for _, must := range []string{"find_load_by_ref", `"not_found"`, "Read-only", "Does not contact"} {
		if !strings.Contains(d, must) {
			t.Errorf("description lacks %q", must)
		}
	}
}

// A fake model stands in for a real one: this tests
// the harness, not any model's tool choice.
func TestEvalScoresFirstCallAndForbiddenCalls(t *testing.T) {
	cases := []Case{
		{"Where is HF-12345?", "get_load"},
		{"Where is PO 88-1192?", "find_load_by_ref"},
		{"PO 88-1192 status please", "find_load_by_ref"},
		{"Cancel my pickup", ""}, // no tool for this agent
		{"Is my thing ok?", ""},  // ambiguous: ask first
		{"Where is HF-77777?", "get_load"},
	}
	f := &llm.FakeModel{Responses: []llm.Response{
		{ToolCalls: []llm.ToolCall{call("get_load")}},
		{ToolCalls: []llm.ToolCall{call("find_load_by_ref")}},
		{ToolCalls: []llm.ToolCall{call("get_load")}}, // wrong first tool
		{Text: "I can't cancel loads."},
		// Hedges in text, then calls a tool anyway.
		{Text: "Do you mean a load?", ToolCalls: []llm.ToolCall{call("get_load")}},
		// Right first call, then a tool it doesn't have.
		{ToolCalls: []llm.ToolCall{call("get_load"), call("cancel_load")}},
	}}
	rs, err := Eval(context.Background(), f,
		[]llm.Tool{GetLoad, findByRef}, cases)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, true, false, true, false, false}
	for i, r := range rs {
		if r.OK != want[i] {
			t.Errorf("case %d (%q): ok=%v why=%q", i, r.Request, r.OK, r.Why)
		}
	}
	if !strings.Contains(rs[4].Why, "should call no tool") ||
		!strings.Contains(rs[5].Why, "cancel_load") {
		t.Fatalf("whys: %q / %q", rs[4].Why, rs[5].Why)
	}
	// Each request was sent with only the tool list.
	for _, c := range f.Calls {
		if c.System != "" || len(c.Tools) != 2 {
			t.Fatalf("request carried more than the tools: %+v", c)
		}
	}
	acc := FirstCallAccuracy(rs)
	if acc["find_load_by_ref"] != [2]int{1, 2} || acc["(no tool)"] != [2]int{1, 2} {
		t.Fatalf("accuracy = %v", acc)
	}
}
