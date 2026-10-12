// Package judge grades free-text drafts with a yes/no
// rubric and grades the judge against hand labels.
//
// Move 84: LLM-as-judge with a rubric, and an eval
// for the judge.
package judge

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/udaykishore-resu/unattended/llm"
)

//go:embed rubric.txt
var Rubric string

// Schema constrains the judge's output shape; Parse
// still checks every field in Go.
var Schema = json.RawMessage(`{"type":"object",
"required":["grounded","answers_question","tone_ok","reason"],
"properties":{"grounded":{"type":"boolean"},
"answers_question":{"type":"boolean"},
"tone_ok":{"type":"boolean"},"reason":{"type":"string"}}}`)

// Grade is one judge verdict.
type Grade struct {
	Grounded        *bool  `json:"grounded"`
	AnswersQuestion *bool  `json:"answers_question"`
	ToneOK          *bool  `json:"tone_ok"`
	Reason          string `json:"reason"`
}

// Pass is true only when every criterion is true.
func (g Grade) Pass() bool {
	return *g.Grounded && *g.AnswersQuestion && *g.ToneOK
}

var ErrShape = errors.New("judge output missing a criterion")

// Parse rejects output with a missing criterion
// instead of reading it as false or true.
func Parse(text string) (Grade, error) {
	var g Grade
	if err := json.Unmarshal([]byte(text), &g); err != nil {
		return g, err
	}
	if g.Grounded == nil || g.AnswersQuestion == nil || g.ToneOK == nil {
		return g, ErrShape
	}
	return g, nil
}

// Judge asks the model to grade one draft. The judge
// sees the shipment record: without it, "grounded"
// can't be checked.
func Judge(ctx context.Context, m llm.Model,
	record, ticket, draft string) (Grade, error) {
	resp, err := m.Complete(ctx, llm.Request{
		System:    Rubric,
		Schema:    Schema,
		MaxTokens: 300,
		Messages: []llm.Message{{Role: "user", Content: fmt.Sprintf(
			"<record>\n%s\n</record>\n<ticket>\n%s\n</ticket>\n<draft>\n%s\n</draft>",
			record, ticket, draft)}},
	})
	if err != nil {
		return Grade{}, err
	}
	return Parse(resp.Text)
}

// Score compares judge verdicts with hand labels
// (true = pass). A false pass is the dangerous kind:
// the judge passed a draft a person failed.
type Score struct{ Agree, FalsePass, FalseFail, N int }

func Compare(judge, human []bool) Score {
	s := Score{N: len(human)}
	for i := range human {
		switch {
		case judge[i] == human[i]:
			s.Agree++
		case judge[i]:
			s.FalsePass++
		default:
			s.FalseFail++
		}
	}
	return s
}
