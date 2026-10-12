// Package flaky finds tests that both passed and
// failed on the same commit.
//
// Move 70: flaky-test triage.
package flaky

// Run is one test result: a test name, a commit SHA
// and whether it passed.
type Run struct {
	Test, SHA string
	Passed    bool
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
