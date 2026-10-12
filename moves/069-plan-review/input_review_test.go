package planreview

import (
	"errors"
	"strings"
	"testing"
)

var planWithVersion = `{"format_version": "1.2",` + plan[1:]

// The model reads the plan's real changes and the
// engine's results, not the source manifests.
func TestInputHasPlanAndEngineResults(t *testing.T) {
	in, err := Input("Prod deployments keep at least 2 replicas.",
		[]byte(planWithVersion), []Finding{
			{Rule: "min-replicas", Address: "kubernetes_deployment.dispatch", Pass: true},
			{Rule: "memory-limit", Address: "kubernetes_deployment.dispatch", Pass: false},
		})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pass min-replicas", "FAIL memory-limit",
		`"Address":"kubernetes_deployment.dispatch"`, `"replicas":1`,
		"at least 2 replicas"} {
		if !strings.Contains(in, want) {
			t.Errorf("missing %s in\n%s", want, in)
		}
	}
	if strings.Contains(in, "hunter2") {
		t.Fatalf("secret leaked:\n%s", in)
	}
}

// No plan, or something that isn't plan JSON: the
// check fails instead of reviewing what it was given.
func TestMissingPlanFailsTheCheck(t *testing.T) {
	for _, p := range []string{"", "apiVersion: apps/v1\nkind: Deployment\n", `{"resource_changes":[]}`} {
		if _, err := Input("policy", []byte(p), nil); !errors.Is(err, ErrNoPlan) {
			t.Errorf("%q: err = %v", p, err)
		}
	}
}
