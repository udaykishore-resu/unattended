package batch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestToSendRefusesDuplicateIDs(t *testing.T) {
	ps := []Pending{{Key: "a"}, {Key: "b"}, {Key: "a"}}
	if _, err := ToSend(ps); err == nil {
		t.Fatal("duplicate custom ID accepted")
	}
	sent, err := ToSend(ps[:2])
	if err != nil || len(sent) != 2 {
		t.Fatalf("sent=%d err=%v", len(sent), err)
	}
}

// Every submitted ID ends in exactly one documented
// state, whatever came back.
func TestEveryIDEndsDoneOrMissed(t *testing.T) {
	cases := map[string]Item{
		"ok":        {Resp: llm.Response{StopReason: llm.StopEnd}},
		"truncated": {Resp: llm.Response{StopReason: llm.StopMaxTokens}},
		"refused":   {Resp: llm.Response{StopReason: llm.StopRefusal}},
		"other":     {Resp: llm.Response{StopReason: llm.StopOther}},
		"tool":      {Resp: llm.Response{StopReason: llm.StopToolUse}},
		"errored":   {Err: errors.New("invalid_request")},
		"empty":     {}, // no stop reason at all
	}
	var ps []Pending
	for k := range cases {
		ps = append(ps, Pending{Key: k})
	}
	ps = append(ps, Pending{Key: "lost"}) // never comes back
	sent, err := ToSend(ps)
	if err != nil {
		t.Fatal(err)
	}
	res := map[string]Item{"stranger": {Resp: llm.Response{StopReason: llm.StopEnd}}}
	for k, v := range cases {
		res[k] = v
	}
	for want, done := range map[llm.StopReason]string{
		llm.StopEnd:     "ok",
		llm.StopToolUse: "tool", // a forced tool call
	} {
		out, err := Collect(context.Background(), fakeBatcher{res}, "b", want, sent)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Done) != 1 || out.Done[done].StopReason != want {
			t.Fatalf("want %s: done=%v", want, out.Done)
		}
		if len(out.Done)+len(out.Missed) != len(sent) {
			t.Fatalf("want %s: %d done + %d missed != %d sent", want,
				len(out.Done), len(out.Missed), len(sent))
		}
		for _, k := range out.Missed {
			if _, ok := out.Done[k]; ok {
				t.Fatalf("%s both done and missed", k)
			}
		}
		if slices.Contains(out.Missed, "stranger") {
			t.Fatal("an ID we never sent was counted")
		}
	}
}

func TestResultsErrorIsReturned(t *testing.T) {
	_, err := Collect(context.Background(), errBatcher{}, "b", llm.StopEnd,
		map[string]llm.Request{"a": {}})
	if err == nil {
		t.Fatal("want the Results error")
	}
}

type errBatcher struct{ fakeBatcher }

func (errBatcher) Results(context.Context, string) (map[string]Item, error) {
	return nil, fmt.Errorf("batch expired")
}
