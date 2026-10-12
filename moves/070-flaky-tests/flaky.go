// Package flaky finds tests that both passed and
// failed on the same commit, and keeps the evidence
// from the failed runs. A pass on re-run is evidence
// of a flake, not proof the failure was harmless.
//
// Move 70: flaky-test triage.
package flaky

import "errors"

// Run is one test result: a test name, a commit SHA,
// whether it passed, and for a failure, the runner it
// ran on and its output, kept even after a re-run
// passes.
type Run struct {
	Test, SHA string
	Passed    bool
	Runner    string
	Log       string
}

func Flaky(runs []Run) map[string]int {
	type key struct{ test, sha string }
	seen := map[key][2]bool{}
	for _, r := range runs {
		k := key{r.Test, r.SHA}
		s := seen[k]
		s[0] = s[0] || r.Passed
		s[1] = s[1] || !r.Passed
		seen[k] = s
	}
	flips := map[string]int{}
	for k, s := range seen {
		if s[0] && s[1] {
			flips[k.test]++
		}
	}
	return flips
}

// Failures returns up to n failed runs of a test, one
// per commit, with their logs and runners: the
// "output from three different flips" the model
// reads. Described in prose; minimal.
func Failures(runs []Run, test string, n int) []Run {
	var out []Run
	shas := map[string]bool{}
	for _, r := range runs {
		if r.Test != test || r.Passed || shas[r.SHA] {
			continue
		}
		shas[r.SHA] = true
		out = append(out, r)
		if len(out) == n {
			break
		}
	}
	return out
}

// ErrRace: our rule is that a flake whose cause is a
// data race is never quarantined; it goes to the code
// owner as a bug.
var ErrRace = errors.New("flaky: data race goes to the code owner, not quarantine")

// Quarantine checks our quarantine rule for a cause
// from the fixed enum.
func Quarantine(cause string) error {
	if cause == "data race" {
		return ErrRace
	}
	return nil
}
