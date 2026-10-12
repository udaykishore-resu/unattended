// Package reprotest classifies `go test -json`
// output for a generated test: reproduces, passes or
// broken.
//
// Move 62: generate a failing test from a bug report.
package reprotest

import (
	"bytes"
	"encoding/json"
	"strings"
)

func Classify(out []byte, test string) string {
	seen := map[string]bool{}
	nl := []byte("\n")
	for _, ln := range bytes.Split(out, nl) {
		var e struct{ Action, Test, Output string }
		err := json.Unmarshal(ln, &e)
		if err != nil || e.Test != test {
			continue
		}
		seen[e.Action] = true
		if strings.Contains(e.Output, "panic:") {
			seen["panic"] = true
		}
	}
	if seen["fail"] && !seen["panic"] {
		return "reproduces"
	} else if seen["pass"] {
		return "passes"
	}
	return "broken"
}
