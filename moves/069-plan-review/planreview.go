// Package planreview trims a Terraform plan to its
// real changes and masks sensitive values on both
// sides before a model reads it.
//
// Move 69: review Terraform and Kubernetes manifests
// against policy.
package planreview

import "encoding/json"

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
