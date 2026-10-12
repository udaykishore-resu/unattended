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

// The book's code starts here.

type Packet struct {
	LoadID string `json:"load_id"`
	Status string `json:"status"`
	NewETA string `json:"new_eta"`
	Reply  string `json:"draft_reply"`
	Unsure string `json:"unsure_about"`
	Seeded bool   `json:"-"` // set by us, never the model
}

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
	if err != nil {
		return p, err
	}
	return p, p.Check()
}

// The book's code ends here. schema, Check and the
// send path below are named in the book's prose.

// schema is the packet's JSON Schema. The book says
// "Part III builds it"; this is a minimal version
// so the move runs on its own. A provider may or may
// not enforce it strictly, so Check runs regardless.
var schema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["load_id", "status", "new_eta",
    "draft_reply", "unsure_about"],
  "properties": {
    "load_id":      {"type": "string"},
    "status":       {"type": "string",
      "enum": ["at_shipper", "in_transit",
               "delayed", "delivered"]},
    "new_eta":      {"type": "string"},
    "draft_reply":  {"type": "string"},
    "unsure_about": {"type": "string"}
  }
}`)
