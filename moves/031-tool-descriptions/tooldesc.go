// Package tooldesc holds a tool description written
// for a new hire, and the harness that tests tool
// descriptions like prompts: which tool does the
// model call first, and does it call none when no
// tool should be used?
//
// Move 31: tool descriptions are prompts: write them
// for a new hire.
package tooldesc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

// GetLoad answers the five questions: what it does,
// when to use it, what inputs mean, what comes back
// (and what empty means), and what it does not do.
var GetLoad = llm.Tool{
	Name: "get_load",
	Description: `Look up one load by Halcyon load ID (format
HF-12345). Use this when you already have a load
ID. If the customer gave a PO, BOL or pro number,
call find_load_by_ref first to get the load ID.

Returns status, pickup and delivery windows,
carrier name and last check-call time. Times are
UTC. "not_found" means the ID does not exist; it
never means the load is late or lost.

Read-only. Does not contact the carrier.`,
	InputSchema: json.RawMessage(`{"type":"object",
  "additionalProperties":false,"required":["load_id"],
  "properties":{"load_id":{"type":"string",
    "description":"Halcyon load ID, like HF-12345"}}}`),
}

// Case is one real request. Want is the tool the
// model should call first; "" means no tool at all
// (the request is out of scope or ambiguous, and
// the right move is to ask or answer in text).
type Case struct {
	Request string
	Want    string
}

// Result is one case's outcome.
type Result struct {
	Case
	First string // first tool called, "" if none
	OK    bool
	Why   string // set when !OK
}

// Eval sends each case with only the tool list and
// scores the model's first response. A case passes
// only if the first call matches Want AND no call
// in the response names a tool outside the list.
// When Want is "", any call at all fails the case,
// so a model that hedges in text and then calls a
// tool anyway is caught.
func Eval(ctx context.Context, m llm.Model,
	tools []llm.Tool, cases []Case) ([]Result, error) {
	known := map[string]bool{}
	for _, t := range tools {
		known[t.Name] = true
	}
	var out []Result
	for _, c := range cases {
		resp, err := m.Complete(ctx, llm.Request{
			Tools:     tools,
			Messages:  []llm.Message{{Role: "user", Content: c.Request}},
			MaxTokens: 512,
		})
		if err != nil {
			return out, err
		}
		r := Result{Case: c}
		if len(resp.ToolCalls) > 0 {
			r.First = resp.ToolCalls[0].Name
		}
		r.OK, r.Why = score(c, resp.ToolCalls, known)
		out = append(out, r)
	}
	return out, nil
}

func score(c Case, calls []llm.ToolCall,
	known map[string]bool) (bool, string) {
	for _, tc := range calls {
		if !known[tc.Name] {
			return false, fmt.Sprintf("called unknown tool %q", tc.Name)
		}
	}
	switch {
	case c.Want == "" && len(calls) > 0:
		return false, fmt.Sprintf("should call no tool, called %q", calls[0].Name)
	case c.Want != "" && len(calls) == 0:
		return false, "called no tool"
	case c.Want != "" && calls[0].Name != c.Want:
		return false, fmt.Sprintf("first call %q, want %q", calls[0].Name, c.Want)
	}
	return true, ""
}

// FirstCallAccuracy is passes per wanted tool, the
// number to track per tool in production too.
func FirstCallAccuracy(rs []Result) map[string][2]int {
	acc := map[string][2]int{}
	for _, r := range rs {
		k := r.Want
		if k == "" {
			k = "(no tool)"
		}
		v := acc[k]
		v[1]++
		if r.OK {
			v[0]++
		}
		acc[k] = v
	}
	return acc
}
