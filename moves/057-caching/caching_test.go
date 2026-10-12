package caching

import (
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func req(ticket string) llm.Request {
	return llm.Request{
		System:      "Support policy... (6,000 tokens)",
		CacheSystem: true, // provider prompt caching
		Messages:    []llm.Message{{Role: "user", Content: ticket}},
		MaxTokens:   100,
	}
}

func key(t *testing.T, tenant, ver string, r llm.Request) string {
	t.Helper()
	k, err := CacheKey(tenant, "small", ver, r)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestKeyCoversWhatChangesTheAnswer(t *testing.T) {
	base := key(t, "acme", "v7", req("where is HF-1?"))
	if key(t, "acme", "v7", req("where is HF-1?")) != base {
		t.Fatal("same request, different key")
	}
	if key(t, "zeta", "v7", req("where is HF-1?")) == base {
		t.Fatal("two tenants must never share an answer")
	}
	// A prompt fix bumps the version, so stale answers stop matching.
	if key(t, "acme", "v8", req("where is HF-1?")) == base {
		t.Fatal("prompt version must be in the key")
	}
	if key(t, "acme", "v7", req("where is HF-2?")) == base {
		t.Fatal("messages must be in the key")
	}
}

func TestToolOrderIsStable(t *testing.T) {
	a := []llm.Tool{{Name: "get_load"}, {Name: "add_note"}}
	b := []llm.Tool{{Name: "add_note"}, {Name: "get_load"}}
	SortTools(a)
	SortTools(b)
	ra, rb := req("x"), req("x")
	ra.Tools, rb.Tools = a, b
	if key(t, "acme", "v7", ra) != key(t, "acme", "v7", rb) {
		t.Fatal("sorted tools should give one prefix")
	}
}
