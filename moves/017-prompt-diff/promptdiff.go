// Package promptdiff compares two prompt versions
// on the same frozen cases and names what changed.
//
// Move 17: diff two prompt versions on the same 50
// inputs.
package promptdiff

import (
	"fmt"
	"io"
)

// Row is one case: its ID, the wanted label, and
// the answers from prompt A and prompt B.
type Row struct{ ID, Want, A, B string }

func report(w io.Writer, rows []Row) {
	n := map[string]int{}
	for _, r := range rows {
		a, b := r.A == r.Want, r.B == r.Want
		k := "still right"
		switch {
		case !a && b:
			k = "FIXED"
		case a && !b:
			k = "BROKE"
		case !a:
			k = "still wrong"
		}
		n[k]++
		if r.A != r.B {
			fmt.Fprintf(w, "%s %s: %s -> %s\n",
				k, r.ID, r.A, r.B)
		}
	}
	fmt.Fprintln(w, n)
}
