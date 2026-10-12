// Package evidence checks that each extracted value
// comes with a quote that is really in the source,
// and that a number's quote actually holds it.
//
// Move 23: confidence plus evidence span for every
// field.
package evidence

import (
	"slices"
	"strconv"
	"strings"

	"github.com/udaykishore-resu/unattended/llm"
)

type Field[T any] struct {
	Value      T      `json:"value"`
	Confidence string `json:"confidence"`
	Evidence   string `json:"evidence"`
}

func norm(s string) string {
	return strings.Join(
		strings.Fields(strings.ToLower(s)), " ")
}

// grounded is an evidence-presence check: the quote
// appears somewhere in the text the model was sent.
// It does not show the quote supports the value.
func grounded(sent, evidence string) bool {
	e := norm(evidence)
	return e != "" && strings.Contains(norm(sent), e)
}

// numIn reports whether v appears in the quote as a
// whole number, after dropping thousands commas.
func numIn(evidence string, v int64) bool {
	q := strings.ReplaceAll(evidence, ",", "")
	nums := strings.FieldsFunc(q, func(r rune) bool {
		return r < '0' || r > '9'
	})
	return slices.Contains(nums,
		strconv.FormatInt(v, 10))
}

// Sent returns the exact text the model saw: every
// user message of the request, joined. Check quotes
// against this, never against the raw email, so a
// redaction step cannot make every quote fail.
func Sent(req llm.Request) string {
	var parts []string
	for _, m := range req.Messages {
		if m.Role == "user" {
			parts = append(parts, m.Content)
		}
	}
	return strings.Join(parts, "\n")
}

// CheckInt runs both checks on a numeric field and
// returns one problem string per failure, for the
// review queue or Move 21's repair note.
func CheckInt(name, sent string, f Field[int64]) []string {
	var p []string
	if !grounded(sent, f.Evidence) {
		p = append(p, name+": evidence not in source")
	} else if !numIn(f.Evidence, f.Value) {
		p = append(p, name+": value "+
			strconv.FormatInt(f.Value, 10)+
			" not in its evidence")
	}
	return p
}
