package rewrite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// A toy index: each query string maps to results.
func plug(t *testing.T) *[]string {
	var seen []string
	hybrid = func(q string) []string {
		seen = append(seen, "hybrid:"+q)
		if strings.Contains(q, "portal login") {
			return []string{"runbook-portal-login"}
		}
		return []string{"runbook-anything"}
	}
	keywordOnly = func(kw string) []string {
		seen = append(seen, "kw:"+kw)
		if strings.Contains(kw, "portal") {
			return []string{"runbook-portal-login"}
		}
		return nil
	}
	t.Cleanup(func() {
		hybrid = func(string) []string { return nil }
		keywordOnly = func(string) []string { return nil }
	})
	return &seen
}

func TestRewriteFindsWhatTheUserMeant(t *testing.T) {
	seen := plug(t)
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{
	  "query":"Brightline tracking portal login error",
	  "keywords":["Brightline","portal"],"already_clear":false}`}}}
	ids := Search(context.Background(), f,
		[]string{"agent: Brightline can't log into the tracking portal"}, "still broken?")
	if ids[0] != "runbook-portal-login" {
		t.Fatalf("ids = %v", ids)
	}
	// The original query is always one of the fused lists.
	if !strings.Contains(strings.Join(*seen, "|"), "hybrid:still broken?") {
		t.Fatalf("original not searched: %v", *seen)
	}
}

func TestFailureSearchesAsTyped(t *testing.T) {
	seen := plug(t)
	f := &llm.FakeModel{Err: errors.New("timeout")}
	Search(context.Background(), f, nil, "detention brightline atlanta")
	want := "hybrid:detention brightline atlanta|hybrid:detention brightline atlanta|kw:"
	if got := strings.Join(*seen, "|"); got != want {
		t.Fatalf("got %s", got)
	}
}
