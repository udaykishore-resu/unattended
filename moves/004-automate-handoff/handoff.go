// Package handoff drafts a dispatcher packet. It
// has no database handle, so it cannot act.
//
// Move 4: automate the handoff, not the whole job.
package handoff

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

type Packet struct {
	LoadID string `json:"load_id"`
	Status string `json:"status"`
	NewETA string `json:"new_eta"`
	Reply  string `json:"draft_reply"`
	Unsure string `json:"unsure_about"`
}

// schema is the packet's JSON Schema. The book says
// "Part III builds it"; this is a minimal version
// so the move runs on its own.
var schema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["load_id", "status", "new_eta",
    "draft_reply", "unsure_about"],
  "properties": {
    "load_id":      {"type": "string"},
    "status":       {"type": "string"},
    "new_eta":      {"type": "string"},
    "draft_reply":  {"type": "string"},
    "unsure_about": {"type": "string"}
  }
}`)

func Draft(ctx context.Context, m llm.Model,
	sys, email string) (p Packet, err error) {
	msg := llm.Message{Role: "user", Content: email}
	resp, err := m.Complete(ctx, llm.Request{
		System: sys, Messages: []llm.Message{msg},
		MaxTokens: 600, Schema: schema,
	})
	if err != nil {
		return p, err
	}
	if resp.StopReason != llm.StopEnd {
		return p, fmt.Errorf("draft stopped: %s",
			resp.StopReason)
	}
	err = json.Unmarshal([]byte(resp.Text), &p)
	return p, err
}
