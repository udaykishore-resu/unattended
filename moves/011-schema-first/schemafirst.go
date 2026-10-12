// Package schemafirst puts the output schema at
// the top of the system prompt and in the request.
//
// Move 11: put the output schema before the
// instructions.
package schemafirst

import (
	"encoding/json"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

// invoiceSchema is one JSON Schema value with
// total_cents as an integer. One copy serves the
// prompt, the request and the validator.
var invoiceSchema = json.RawMessage(`{
  "type": "object",
  "required": ["invoice_number", "total_cents"],
  "properties": {
    "invoice_number": {"type": "string"},
    "total_cents": {"type": "integer",
      "description": "integer number of cents"}
  }
}`)

func invoiceRequest(rules, doc string) llm.Request {
	var sys strings.Builder
	sys.WriteString("Return one JSON object that ")
	sys.WriteString("matches this schema:\n")
	sys.Write(invoiceSchema)
	sys.WriteString("\n\nRules:\n" + rules)
	return llm.Request{
		System: sys.String(),
		Messages: []llm.Message{
			{Role: "user", Content: doc},
		},
		MaxTokens: 400,
		Schema:    invoiceSchema,
	}
}
