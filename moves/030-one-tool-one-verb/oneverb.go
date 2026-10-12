// Package oneverb defines narrow tools: one verb,
// one kind of object, one narrow schema.
//
// Move 30: one tool, one verb.
package oneverb

import (
	"encoding/json"

	"github.com/udaykishore-resu/unattended/llm"
)

var setPickupWindow = llm.Tool{
	Name: "set_pickup_window",
	Description: "Change the pickup window on one " +
		"load. Does not notify the carrier.",
	InputSchema: json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["load_id", "start", "end"],
  "properties": {
    "load_id": {"type": "string"},
    "start": {"type": "string",
      "description": "RFC 3339, shipper local"},
    "end": {"type": "string"}
  }
}`),
}

// Tools is this agent's whole list. There is no
// cancel_load here, so the agent cannot cancel.
var Tools = []llm.Tool{
	setPickupWindow,
	{Name: "get_load", Description: "Look up one load by load ID.",
		InputSchema: json.RawMessage(`{"type":"object",
		"properties":{"load_id":{"type":"string"}}}`)},
	{Name: "add_note", Description: "Add a note to one load.",
		InputSchema: json.RawMessage(`{"type":"object",
		"properties":{"load_id":{"type":"string"},
		"text":{"type":"string"}}}`)},
}

// Verbs is the short list the CI test checks tool
// names against. Adding a verb takes a pull request.
var Verbs = map[string]bool{
	"get": true, "find": true, "set": true,
	"add": true, "mark": true, "cancel": true,
}
