package batch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

type fakeBatcher struct{ res map[string]Item }

func (fakeBatcher) Submit(context.Context, map[string]llm.Request) (string, error) {
	return "batch-1", nil
}
func (fakeBatcher) Status(context.Context, string) (string, error) { return "ended", nil }
func (f fakeBatcher) Results(context.Context, string) (map[string]Item, error) {
	return f.res, nil
}

// Loop over what you sent, not what came back: the
// only way to see the missing items.
func TestMissingItemsAreFound(t *testing.T) {
	sent := map[string]llm.Request{}
	res := map[string]Item{}
	for i := range 20 {
		key := fmt.Sprintf("lane-%02d", i)
		sent[key] = llm.Request{}
		switch {
		case i < 14:
			res[key] = Item{Resp: llm.Response{Text: "ok", StopReason: llm.StopEnd}}
		case i == 14:
			res[key] = Item{Err: errors.New("invalid_request")}
		case i == 15:
			res[key] = Item{Resp: llm.Response{Text: "cut", StopReason: llm.StopMaxTokens}}
		}
		// 16..19 never came back.
	}
	out, err := Collect(context.Background(), fakeBatcher{res}, "batch-1", llm.StopEnd, sent)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out.Missed)
	want := []string{"lane-14", "lane-15", "lane-16", "lane-17", "lane-18", "lane-19"}
	if len(out.Done) != 14 || !slices.Equal(out.Missed, want) {
		t.Fatalf("done=%d missed=%v", len(out.Done), out.Missed)
	}
}
