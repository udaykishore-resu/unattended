// Package routing sends work to a small model first
// and escalates when the answer isn't high and
// grounded.
//
// Move 58: route by difficulty.
package routing

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

func (rt Router) Classify(ctx context.Context,
	req llm.Request, src string) (Label, string, error) {
	if !rt.Escalate(src) {
		l, err := ask(ctx, rt.Small, req)
		if err == nil && rt.accept(l, src) {
			return l, "small", nil
		}
	}
	l, err := ask(ctx, rt.Large, req)
	if err == nil && rt.accept(l, src) {
		return l, "large", nil
	}
	return l, "human", rt.Review.Add(ctx, req, src)
}

func (rt Router) accept(l Label, src string) bool {
	return rt.Allowed[l.Queue] &&
		l.Confidence == "high" && grounded(src, l.Evidence)
}

// The rest is described in the prose; minimal.

type Label struct {
	Queue      string `json:"queue"`
	Confidence string `json:"confidence"` // high|medium|low
	Evidence   string `json:"evidence"`
}

// Review files a ticket in the review queue (Move 59).
type Review interface {
	Add(ctx context.Context, req llm.Request,
		src string) error
}

type Router struct {
	Small, Large llm.Model
	Allowed      map[string]bool
	Escalate     func(src string) bool // cheap rules
	Review       Review
}

// ask calls the model with a forced schema and
// decodes the Label.
func ask(ctx context.Context, m llm.Model,
	req llm.Request) (Label, error) {
	resp, err := m.Complete(ctx, req)
	if err != nil {
		return Label{}, err
	}
	var l Label
	err = json.Unmarshal([]byte(resp.Text), &l)
	return l, err
}

// grounded is the evidence check from Move 23.
func grounded(src, evidence string) bool {
	norm := func(s string) string {
		return strings.Join(strings.Fields(strings.ToLower(s)), " ")
	}
	e := norm(evidence)
	return e != "" && strings.Contains(norm(src), e)
}
