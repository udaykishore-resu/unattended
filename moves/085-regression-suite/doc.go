// Package regression turns the eval set into a test
// with a floor per label. The move is a test: see
// eval_test.go. It skips unless RUN_EVALS is set:
//
//	RUN_EVALS=1 go test ./moves/085-regression-suite
//
// Move 85: prompt regression suite in CI.
package regression

import (
	"context"
	"encoding/json"

	"github.com/udaykishore-resu/unattended/llm"
)

// Case is one labeled email from real traffic.
// Critical marks a case added for a past incident.
type Case struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Label    string `json:"label"`
	Critical bool   `json:"critical"`
	// Recorded is the model's answer, replayed by
	// the fake model so the suite runs offline. In
	// CI, swap in the production adapter and drop it.
	Recorded string `json:"recorded"`
}

// Classify is the production call under test.
func Classify(ctx context.Context, m llm.Model,
	email string) (string, error) {
	resp, err := m.Complete(ctx, llm.Request{
		System:    "Label this ops@ email.",
		Messages:  []llm.Message{{Role: "user", Content: "<email>\n" + email + "\n</email>"}},
		MaxTokens: 50,
		Schema: json.RawMessage(`{"type":"object","properties":{"label":{"enum":` +
			`["damage_claim","detention","rate_request","other"]}}}`),
	})
	if err != nil {
		return "", err
	}
	var v struct {
		Label string `json:"label"`
	}
	err = json.Unmarshal([]byte(resp.Text), &v)
	return v.Label, err
}
