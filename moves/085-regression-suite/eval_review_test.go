package regression

import (
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

// recording wraps the fake and remembers which model
// name it was built for.
type recording struct {
	*llm.FakeModel
	built string
}

// CI grades the model and settings production uses:
// every request carries the config's system prompt
// and token cap, and the model is built from the
// config's model name.
func TestSuiteUsesProductionSettings(t *testing.T) {
	prod, err := LoadSettings(SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	var rec *recording
	old := newModel
	t.Cleanup(func() { newModel = old })
	newModel = func(s Settings, cs []Case) llm.Model {
		rec = &recording{FakeModel: old(s, cs).(*llm.FakeModel), built: s.Model}
		return rec
	}
	tally(&recorder{TB: t}, loadCases(t))
	if rec.built != prod.Model {
		t.Fatalf("suite built %q, production runs %q", rec.built, prod.Model)
	}
	if len(rec.Calls) == 0 {
		t.Fatal("no calls")
	}
	for _, r := range rec.Calls {
		if r.System != prod.System || r.MaxTokens != prod.MaxTokens {
			t.Fatalf("request drifted from production settings: %+v", r)
		}
	}
}

// A critical miss fails the build whatever the floors
// say, so lowering floors in the same pull request
// can't wave it through.
func TestCriticalMissIgnoresFloors(t *testing.T) {
	saved := map[string]float64{}
	for k, v := range floors {
		saved[k] = v
		floors[k] = 0
	}
	t.Cleanup(func() {
		for k, v := range saved {
			floors[k] = v
		}
	})
	cs := loadCases(t)
	hit := false
	for i := range cs {
		if cs[i].Critical {
			cs[i].Recorded = "other"
			hit = true
			if cs[i].Label == "other" {
				cs[i].Recorded = "rate_request"
			}
			break
		}
	}
	if !hit {
		t.Fatal("fixture has no critical case")
	}
	rec := &recorder{TB: t}
	tally(rec, cs)
	if len(rec.errs) == 0 {
		t.Fatal("critical miss passed with floors at zero")
	}
}

// A label with no cases scores zero and fails.
func TestEmptyLabelFails(t *testing.T) {
	var kept []Case
	for _, c := range loadCases(t) {
		if c.Label != "detention" {
			kept = append(kept, c)
		}
	}
	total, right := tally(&recorder{TB: t}, kept)
	got := float64(right["detention"]) / float64(max(total["detention"], 1))
	if got >= floors["detention"] {
		t.Fatalf("detention with no cases scored %.2f", got)
	}
}
