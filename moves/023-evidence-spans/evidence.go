// Package evidence checks that each extracted value
// comes with a quote that is really in the source.
//
// Move 23: confidence plus evidence span for every
// field.
package evidence

import "strings"

type Field[T any] struct {
	Value      T      `json:"value"`
	Confidence string `json:"confidence"`
	Evidence   string `json:"evidence"`
}

func norm(s string) string {
	return strings.Join(
		strings.Fields(strings.ToLower(s)), " ")
}

func grounded(src, evidence string) bool {
	e := norm(evidence)
	return e != "" && strings.Contains(norm(src), e)
}
