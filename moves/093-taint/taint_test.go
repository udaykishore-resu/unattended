package taint

import (
	"slices"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func names(ts []llm.Tool) []string {
	var out []string
	for _, t := range ts {
		out = append(out, t.Name)
	}
	return out
}

func TestTaintIsOneWay(t *testing.T) {
	r := NewRun([]llm.Tool{{Name: "get_load"}, {Name: "add_note"},
		{Name: "set_status"}, {Name: "fetch_url"}, {Name: "search_text"}},
		"get_load")
	if len(r.Tools()) != 5 {
		t.Fatal("a clean run keeps every tool")
	}
	// A carrier PDF enters the run.
	r.Add(true)
	if got := names(r.Tools()); !slices.Equal(got, []string{"get_load"}) {
		t.Fatalf("tainted tools = %v", got)
	}
	// Trusted messages later never clear it.
	r.Add(false)
	r.Add(false)
	if !r.Tainted() || len(r.Tools()) != 1 {
		t.Fatal("taint cleared")
	}
}
