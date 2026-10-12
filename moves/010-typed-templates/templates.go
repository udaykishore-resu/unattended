// Package templates renders prompts from one typed
// struct, so a missing field fails at render time.
//
// Move 10: templated prompts with typed variables.
package templates

import (
	"errors"
	"strings"
	"text/template"
)

type ClassifyVars struct {
	From, Subject, Body string
	OpenLoads           []string
}

var classifyTmpl = template.Must(
	template.New("classify").
		Funcs(template.FuncMap{"clip": clip}).
		Option("missingkey=error").
		Parse(classifyBody))

func RenderClassify(v ClassifyVars) (string, error) {
	if strings.TrimSpace(v.Body) == "" {
		return "", errors.New("classify: empty body")
	}
	var b strings.Builder
	err := classifyTmpl.Execute(&b, v)
	return b.String(), err
}

// classifyBody is the template text. The book
// embeds it as in Move 9; it is inline here so the
// move stays one file. It fences the email in
// <email> tags.
const classifyBody = `Classify this email for Halcyon ops.
From: {{.From}}
Subject: {{.Subject}}
Open loads for this sender: {{range .OpenLoads}}{{.}} {{end}}
<email>
{{clip .Body 4000}}
</email>
`

// clip cuts s to at most n runes on a rune boundary
// and appends "[truncated]", so the model knows the
// text is incomplete. Named only in prose; minimal.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "[truncated]"
}
