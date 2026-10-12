// Package deletion removes one rule at a time and
// reports how accuracy moves.
//
// Move 19: the deletion test.
package deletion

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
)

// Eval runs a system prompt over a fixed case set
// and returns accuracy from 0 to 1.
type Eval func(context.Context, string) (float64, error)

func ablate(ctx context.Context, w io.Writer,
	head string, rules []string, eval Eval) error {
	build := func(rs []string) string {
		return head + "\n" + strings.Join(rs, "\n")
	}
	base, err := eval(ctx, build(rules))
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "baseline %.3f\n", base)
	for i, r := range rules {
		rest := slices.Delete(slices.Clone(rules), i, i+1)
		acc, err := eval(ctx, build(rest))
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%+.3f -%.40s\n", acc-base, r)
	}
	return nil
}
