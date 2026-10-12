package rewrite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// A toy index: each query string maps to results.
// Every call records the tenant it was given.
func plug(t *testing.T) *[]string {
	var seen []string
	hybridSearch = func(tenant, q string) []string {
		seen = append(seen, tenant+"/hybrid:"+q)
		if strings.Contains(q, "portal login") {
			return []string{"runbook-portal-login"}
		}
		return []string{"runbook-anything"}
	}
	keywordSearch = func(tenant, kw string) []string {
		seen = append(seen, tenant+"/kw:"+kw)
		if strings.Contains(kw, "portal") {
			return []string{"runbook-portal-login"}
		}
		return nil
	}
	t.Cleanup(func() {
		hybridSearch = func(string, string) []string { return nil }
		keywordSearch = func(string, string) []string { return nil }
	})
	return &seen
}

func TestRewriteFindsWhatTheUserMeant(t *testing.T) {
	seen := plug(t)
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{
	  "query":"Brightline tracking portal login error",
	  "keywords":["Brightline","portal"],"already_clear":false}`}}}
	ids := Search(context.Background(), f, "halcyon",
		[]string{"agent: Brightline can't log into the tracking portal"}, "still broken?")
	if ids[0] != "runbook-portal-login" {
		t.Fatalf("ids = %v", ids)
	}
	// The original query is always one of the fused lists.
	if !strings.Contains(strings.Join(*seen, "|"), "halcyon/hybrid:still broken?") {
		t.Fatalf("original not searched: %v", *seen)
	}
}

func TestFailureSearchesAsTyped(t *testing.T) {
	seen := plug(t)
	f := &llm.FakeModel{Err: errors.New("timeout")}
	Search(context.Background(), f, "halcyon", nil, "detention brightline atlanta")
	want := "halcyon/hybrid:detention brightline atlanta|" +
		"halcyon/hybrid:detention brightline atlanta|halcyon/kw:"
	if got := strings.Join(*seen, "|"); got != want {
		t.Fatalf("got %s", got)
	}
}
