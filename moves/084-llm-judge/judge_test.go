package judge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// Halcyon's first judge: 131 of 150 agree, 12 false
// passes, so 7 false fails.
func TestCompareCounts(t *testing.T) {
	var judge, human []bool
	add := func(n int, j, h bool) {
		for range n {
			judge, human = append(judge, j), append(human, h)
		}
	}
	add(100, true, true)
	add(31, false, false)
	add(12, true, false)
	add(7, false, true)
	s := Compare(judge, human)
	if s != (Score{Agree: 131, FalsePass: 12, FalseFail: 7, N: 150}) {
		t.Fatalf("score = %+v", s)
	}
}

func TestParseRejectsMissingCriterion(t *testing.T) {
	if _, err := Parse(`{"grounded":true,"tone_ok":true,"reason":""}`); !errors.Is(err, ErrShape) {
		t.Fatalf("err = %v", err)
	}
	g, err := Parse(`{"grounded":false,"answers_question":true,"tone_ok":true,"reason":"Friday ETA not in record"}`)
	if err != nil || g.Pass() {
		t.Fatalf("g=%+v err=%v", g, err)
	}
}

// The judge gets the record, so a fluent draft with an
// unsupported date can be failed on "grounded". With a
// fake model this checks the wiring, not judge quality.
func TestJudgeSeesRecord(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{{Text: `{"grounded":false,` +
		`"answers_question":true,"tone_ok":true,"reason":"ETA Friday not in record"}`}}}
	g, err := Judge(context.Background(), f, "HF-1 status: in transit, ETA none",
		"When will HF-1 arrive?", "HF-1 arrives Friday by noon.")
	if err != nil || g.Pass() {
		t.Fatalf("g=%+v err=%v", g, err)
	}
	r := f.Calls[0]
	if !strings.Contains(r.Messages[0].Content, "<record>\nHF-1 status") ||
		r.System != Rubric || len(r.Schema) == 0 {
		t.Fatalf("request = %+v", r)
	}
}
