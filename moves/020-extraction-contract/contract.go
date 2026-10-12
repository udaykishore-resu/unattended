// Package contract makes a JSON Schema the
// extraction contract and reads the record from a
// forced tool call.
//
// Move 20: schemas and tool definitions as the
// extraction contract.
package contract

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/udaykishore-resu/unattended/llm"
)

var tenderSchema = json.RawMessage(`{
  "type": "object",
  "additionalProperties": false,
  "required": ["shipper_ref", "origin_zip",
    "dest_zip", "pickup_date", "weight_lb"],
  "properties": {
    "shipper_ref": {"type": "string"},
    "origin_zip":  {"type": "string",
                    "pattern": "^[0-9]{5}$"},
    "dest_zip":    {"type": "string",
                    "pattern": "^[0-9]{5}$"},
    "pickup_date": {"type": "string",
                    "description": "as written"},
    "weight_lb":   {"type": "integer",
                    "minimum": 1, "maximum": 80000}
  }
}`)

// SaveTender is the single tool the schema rides on.
// Halcyon's adapter forces a tool whenever a request
// carries exactly one.
var SaveTender = llm.Tool{
	Name:        "save_tender",
	Description: "Save one tender extracted from the email.",
	InputSchema: tenderSchema,
}

// Tender mirrors the schema.
type Tender struct {
	ShipperRef string `json:"shipper_ref"`
	OriginZip  string `json:"origin_zip"`
	DestZip    string `json:"dest_zip"`
	PickupDate string `json:"pickup_date"`
	WeightLb   int    `json:"weight_lb"`
}

var ErrNoToolCall = errors.New("no tool call")

func oneCall(r llm.Response, name string) (
	llm.ToolCall, error) {
	if len(r.ToolCalls) != 1 ||
		r.ToolCalls[0].Name != name {
		return llm.ToolCall{}, ErrNoToolCall
	}
	return r.ToolCalls[0], nil
}

func decodeStrict(in json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
