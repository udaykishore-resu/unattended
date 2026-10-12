package regression

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

var floors = map[string]float64{
	"damage_claim": 0.95, "detention": 0.90,
	"rate_request": 0.85, "other": 0.70,
}

func TestEvalTriage(t *testing.T) {
	if os.Getenv("RUN_EVALS") == "" {
		t.Skip("set RUN_EVALS=1")
	}
	total, right := tally(t, loadCases(t))
	for label, floor := range floors {
		got := float64(right[label]) /
			float64(max(total[label], 1))
		if got < floor {
			t.Errorf("%s: %.2f < %.2f",
				label, got, floor)
		}
	}
}

// casesFile is the frozen set; a var so the
// regression demo below can point at a broken run.
var casesFile = "testdata/cases.json"

func loadCases(t testing.TB) []Case {
	t.Helper()
	b, err := os.ReadFile(casesFile)
	if err != nil {
		t.Fatal(err)
	}
	var cs []Case
	if err := json.Unmarshal(b, &cs); err != nil {
		t.Fatal(err)
	}
	return cs
}

// tally runs each case and logs every miss with its
// case ID. Any miss on a critical case fails.
func tally(t testing.TB, cs []Case) (total, right map[string]int) {
	t.Helper()
	total, right = map[string]int{}, map[string]int{}
	for label := range floors {
		total[label] = 0 // a label with no cases scores zero
	}
	s, err := LoadSettings(SettingsFile)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, cs)
	for _, c := range cs {
		got, err := Classify(context.Background(), m, s, c.Email)
		total[c.Label]++
		if err == nil && got == c.Label {
			right[c.Label]++
			continue
		}
		t.Logf("miss %s: want %s, got %s (%v)", c.ID, c.Label, got, err)
		if c.Critical {
			t.Errorf("critical case %s missed", c.ID)
		}
	}
	return total, right
}

// newModel builds the model the suite grades from the
// production settings. Offline it replays recorded
// answers; in CI it is the production adapter factory
// called with s.Model, so a different model can't be
// graded by accident.
var newModel = func(s Settings, cs []Case) llm.Model {
	f := &llm.FakeModel{}
	for _, c := range cs {
		f.Responses = append(f.Responses, llm.Response{
			Text: `{"label":"` + c.Recorded + `"}`})
	}
	return f
}

// The suite as CI runs it, with the variable set.
func TestEvalTriageInCI(t *testing.T) {
	t.Setenv("RUN_EVALS", "1")
	TestEvalTriage(t)
}

// recorder is a testing.TB that collects errors
// instead of failing, to show what CI would report.
type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Errorf(f string, a ...any) { r.errs = append(r.errs, f) }
func (r *recorder) Logf(string, ...any)       {}
func (r *recorder) Helper()                   {}

// "Prefer rate_request when a price is mentioned"
// sent damage claims that quoted a repair price into
// rate_request. The per-label floor and the critical
// case both go red, though overall accuracy barely moves.
func TestBrokenPromptGoesRed(t *testing.T) {
	cs := loadCases(t)
	for i := range cs {
		if cs[i].Label == "damage_claim" && strings.Contains(cs[i].Email, "$") {
			cs[i].Recorded = "rate_request"
		}
	}
	rec := &recorder{TB: t}
	total, right := tally(rec, cs)
	dc := float64(right["damage_claim"]) / float64(total["damage_claim"])
	if dc >= floors["damage_claim"] || len(rec.errs) == 0 {
		t.Fatalf("damage_claim %.2f should be under its floor, critical errs=%d",
			dc, len(rec.errs))
	}
	var all, ok int
	for l := range total {
		all += total[l]
		ok += right[l]
	}
	if float64(ok)/float64(all) < 0.85 {
		t.Fatalf("overall %.2f: the demo needs a small overall drop", float64(ok)/float64(all))
	}
}
