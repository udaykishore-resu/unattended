package contract

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestRecordComesFromTheToolCall(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{
		Text: "Saving it now.",
		ToolCalls: []llm.ToolCall{{ID: "c1", Name: "save_tender",
			Input: json.RawMessage(`{"shipper_ref":"ACME-1",
			  "origin_zip":"02134","dest_zip":"60601",
			  "pickup_date":"3/12","weight_lb":42000}`)}},
	}}}
	resp, _ := f.Complete(context.Background(), llm.Request{
		Tools: []llm.Tool{SaveTender}})
	call, err := oneCall(resp, "save_tender")
	if err != nil {
		t.Fatal(err)
	}
	var tn Tender
	if err := decodeStrict(call.Input, &tn); err != nil {
		t.Fatal(err)
	}
	// ZIPs are strings, so the leading zero survives.
	if tn.OriginZip != "02134" || tn.WeightLb != 42000 {
		t.Fatalf("tender = %+v", tn)
	}
}

func TestProseIsNoToolCall(t *testing.T) {
	r := llm.Response{Text: "Here's a summary of the tender..."}
	if _, err := oneCall(r, "save_tender"); !errors.Is(err, ErrNoToolCall) {
		t.Fatalf("got %v", err)
	}
	wrong := llm.Response{ToolCalls: []llm.ToolCall{{Name: "save_invoice"}}}
	if _, err := oneCall(wrong, "save_tender"); !errors.Is(err, ErrNoToolCall) {
		t.Fatalf("wrong tool: %v", err)
	}
}

func TestStrictDecodeRejectsExtraFields(t *testing.T) {
	in := json.RawMessage(`{"shipper_ref":"A","hazmat":true}`)
	var tn Tender
	if err := decodeStrict(in, &tn); err == nil ||
		!strings.Contains(err.Error(), "hazmat") {
		t.Fatalf("want unknown-field error, got %v", err)
	}
	// "approx 42k" never reaches an integer column.
	bad := json.RawMessage(`{"weight_lb":"approx 42k"}`)
	if err := decodeStrict(bad, &tn); err == nil {
		t.Fatal("a string weight must not decode")
	}
}

// The test the book asks for: every JSON key of the
// struct exists in the schema, so they cannot drift.
func TestSchemaAndStructAgree(t *testing.T) {
	var s struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	}
	if err := json.Unmarshal(tenderSchema, &s); err != nil {
		t.Fatal(err)
	}
	ty := reflect.TypeOf(Tender{})
	for i := range ty.NumField() {
		key := strings.Split(ty.Field(i).Tag.Get("json"), ",")[0]
		if _, ok := s.Properties[key]; !ok {
			t.Errorf("struct key %q missing from schema", key)
		}
	}
	if len(s.Properties) != ty.NumField() {
		t.Errorf("schema has %d properties, struct %d fields",
			len(s.Properties), ty.NumField())
	}
}
