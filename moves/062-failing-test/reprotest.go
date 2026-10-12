// Package reprotest classifies `go test -json`
// output for a generated test: reproduces, passes or
// broken. A failure counts as a reproduction only if
// the named test failed with the assertion's mark
// (the ticket ID) in its own output.
//
// Move 62: generate a failing test from a bug report.
package reprotest

import (
	"bytes"
	"encoding/json"
	"strings"
)

func Classify(out []byte, test, mark string) string {
	seen := map[string]bool{}
	for _, ln := range bytes.Split(out, []byte("\n")) {
		var e struct{ Action, Test, Output string }
		err := json.Unmarshal(ln, &e)
		if err != nil || e.Test != test {
			continue
		}
		seen[e.Action] = true
		if strings.Contains(e.Output, "panic:") {
			seen["panic"] = true
		}
		if mark != "" &&
			strings.Contains(e.Output, mark) {
			seen["mark"] = true
		}
	}
	switch {
	case seen["fail"] && seen["mark"] && !seen["panic"]:
		return "reproduces"
	case seen["pass"]:
		return "passes"
	}
	return "broken"
}
