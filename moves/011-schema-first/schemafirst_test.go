package schemafirst

import (
	"bytes"
	"strings"
	"testing"
)

func TestSchemaComesBeforeRules(t *testing.T) {
	r := invoiceRequest("1. total_cents is cents.", "INVOICE 4471")
	si := strings.Index(r.System, `"total_cents"`)
	ri := strings.Index(r.System, "Rules:")
	if si < 0 || ri < 0 || si > ri {
		t.Fatalf("schema must precede rules:\n%s", r.System)
	}
	// The same bytes go in the prompt and the schema field.
	if !bytes.Equal(r.Schema, invoiceSchema) ||
		!strings.Contains(r.System, string(invoiceSchema)) {
		t.Fatal("prompt and request must share one schema copy")
	}
	if r.Messages[0].Role != "user" || r.Messages[0].Content != "INVOICE 4471" {
		t.Fatalf("document must be the user turn: %+v", r.Messages)
	}
}

// Every field a rule names must exist in the schema
// ("remit-to address" named nothing).
func TestRulesNameRealFields(t *testing.T) {
	rules := []string{"total_cents", "invoice_number"}
	for _, f := range rules {
		if !bytes.Contains(invoiceSchema, []byte(`"`+f+`"`)) {
			t.Errorf("rule names %s, which is not in the schema", f)
		}
	}
}
