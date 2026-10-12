// Package planreview trims a Terraform plan to its
// real changes and masks sensitive values on both
// sides before a model reads it.
//
// Move 69: review Terraform and Kubernetes manifests
// against policy.
package planreview

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Change struct {
	Address string
	Change  struct {
		Actions       []string
		Before, After any
		BeforeSens    any `json:"before_sensitive"`
		AfterSens     any `json:"after_sensitive"`
	}
}

func Trim(cs []Change) []Change {
	var out []Change
	for _, c := range cs {
		d := &c.Change
		if len(d.Actions) == 1 && d.Actions[0] == "no-op" {
			continue
		}
		d.Before = mask(d.Before, d.BeforeSens)
		d.After = mask(d.After, d.AfterSens)
		d.BeforeSens, d.AfterSens = nil, nil
		out = append(out, c)
	}
	return out
}

func mask(v, s any) any {
	if b, ok := s.(bool); ok && b {
		return "<sensitive>"
	}
	switch v := v.(type) {
	case map[string]any:
		sm, _ := s.(map[string]any)
		for k, x := range v {
			v[k] = mask(x, sm[k])
		}
	case []any:
		sl, _ := s.([]any)
		for i, x := range v {
			if i < len(sl) {
				v[i] = mask(x, sl[i])
			}
		}
	}
	return v
}

// ResourceChanges decodes only resource_changes from
// `terraform show -json`, dropping prior_state,
// planned_values, variables and configuration, which
// hold the same values unmasked. Minimal helper.
func ResourceChanges(plan []byte) ([]Change, error) {
	var p struct {
		ResourceChanges []Change `json:"resource_changes"`
	}
	err := json.Unmarshal(plan, &p)
	return p.ResourceChanges, err
}

// Finding is one policy-engine result, pass or fail.
type Finding struct {
	Rule, Address string
	Pass          bool
}

// ErrNoPlan: the check fails when a stack has no plan.
// A review of source manifests alone can't show what
// will change, so the model never gets one.
var ErrNoPlan = errors.New("planreview: plan missing")

// Input builds what the model reads: the policy in
// plain English, the engine's results, and the
// trimmed, masked changes. Described in prose;
// minimal implementation.
func Input(policy string, plan []byte, fs []Finding) (string, error) {
	var head struct {
		FormatVersion string `json:"format_version"`
	}
	if len(plan) == 0 || json.Unmarshal(plan, &head) != nil || head.FormatVersion == "" {
		return "", ErrNoPlan
	}
	cs, err := ResourceChanges(plan)
	if err != nil {
		return "", err
	}
	changes, err := json.Marshal(Trim(cs))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<policy>\n%s\n</policy>\n<engine>\n", policy)
	for _, f := range fs {
		verdict := "FAIL"
		if f.Pass {
			verdict = "pass"
		}
		fmt.Fprintf(&b, "%s %s %s\n", verdict, f.Rule, f.Address)
	}
	fmt.Fprintf(&b, "</engine>\n<changes>\n%s\n</changes>\n", changes)
	return b.String(), nil
}
