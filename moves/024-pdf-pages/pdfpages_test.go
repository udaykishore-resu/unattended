package pdfpages

import (
	"context"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestRowsTaggedWithPage(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{
	  "header":"Lanes 401-450","continues_from_prev":false,
	  "rows":[{"origin":"30301","dest":"75201","rate_cents":185000},
	          {"origin":"30301","dest":"77001","rate_cents":172500}]}`}}}
	p, err := onePage(context.Background(), f, 9, llm.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Rows) != 2 || p.Rows[0].Page != 9 || p.Rows[1].Page != 9 {
		t.Fatalf("rows = %+v", p.Rows)
	}
}

// A page that stopped partway (the "388 of 612"
// failure) is an error, never data.
func TestCutOffPageIsAnError(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{
		Text: `{"rows":[{"origin":"30301"`, StopReason: llm.StopMaxTokens}}}
	_, err := onePage(context.Background(), f, 9, llm.Request{})
	if err == nil || !strings.Contains(err.Error(), "page 9: stopped: max_tokens") {
		t.Fatalf("got %v", err)
	}
}

func TestBadJSONNamesThePage(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: "Sure!"}}}
	_, err := onePage(context.Background(), f, 11, llm.Request{})
	if err == nil || !strings.HasPrefix(err.Error(), "page 11:") {
		t.Fatalf("got %v", err)
	}
}
