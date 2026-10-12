package rerank

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func cands(n int) []Chunk {
	var out []Chunk
	for i := range n {
		out = append(out, Chunk{ID: fmt.Sprintf("c%d", i)})
	}
	return out
}

func TestReranksAndDropsZeros(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{"scores":[
	  {"id":"c0","score":1},{"id":"c1","score":0},{"id":"c2","score":3},
	  {"id":"c3","score":2},{"id":"c4","score":3}]}`}}}
	got, err := Rerank(context.Background(), f, "detention rate", cands(5), 3)
	if err != nil {
		t.Fatal(err)
	}
	ids := fmt.Sprint(got)
	// Ties keep retrieval order: c2 before c4.
	if ids != "[{c2 } {c4 } {c3 }]" {
		t.Fatalf("got %s", ids)
	}
}

// Every chunk scored 0: an empty list, the signal
// Move 46 needs to abstain.
func TestAllZeroIsEmpty(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{"scores":[
	  {"id":"c0","score":0},{"id":"c1","score":0}]}`}}}
	got, err := Rerank(context.Background(), f, "q", cands(2), 5)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

// An outage costs quality; search keeps working.
func TestOutageFallsBackToRetrievalOrder(t *testing.T) {
	f := &llm.FakeModel{Err: errors.New("503")}
	got, err := Rerank(context.Background(), f, "q", cands(50), 5)
	if err == nil || len(got) != 5 || got[0].ID != "c0" {
		t.Fatalf("got %v, %v", got, err)
	}
}
