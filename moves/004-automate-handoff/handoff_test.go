package handoff

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestDraftReturnsPacket(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{
	  "load_id": "HF-90412", "status": "delayed",
	  "new_eta": "Sat 14:00", "draft_reply": "Thanks, noted.",
	  "unsure_about": "carrier gave a range; used the late end"}`}}}
	p, err := Draft(context.Background(), f, "sys", "truck late")
	if err != nil {
		t.Fatal(err)
	}
	if p.LoadID != "HF-90412" || p.Unsure == "" {
		t.Fatalf("packet = %+v", p)
	}
	// The request carried the schema; the output is a draft, not an action.
	if !json.Valid(f.Calls[0].Schema) {
		t.Fatal("request schema is not valid JSON")
	}
}

func TestTruncatedOrRefusedIsError(t *testing.T) {
	for _, stop := range []llm.StopReason{
		llm.StopMaxTokens, llm.StopRefusal} {
		f := &llm.FakeModel{Responses: []llm.Response{
			{Text: `{"load_id": "HF-1"`, StopReason: stop}}}
		if _, err := Draft(context.Background(), f, "", "x"); err == nil {
			t.Errorf("%s: want an error before parsing", stop)
		}
	}
}
