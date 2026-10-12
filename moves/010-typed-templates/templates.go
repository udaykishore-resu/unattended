// Package templates renders prompts from one typed
// struct, so a misspelled field fails at render time.
//
// Move 10: templated prompts with typed variables.
package templates

import (
	"errors"
	"regexp"
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
	v.From = defang(v.From)
	v.Subject = defang(v.Subject)
	v.Body = defang(v.Body)
	var b strings.Builder
	err := classifyTmpl.Execute(&b, v)
	return b.String(), err
}

// classifyBody is the template text. The book
// embeds it as in Move 9; it is inline here so the
// move stays one file. Every value that comes from
// the sender sits inside the <email> fence; open
// loads come from Halcyon's database and get their
// own tag.
const classifyBody = `Classify this email for Halcyon ops.
<open_loads>
{{range .OpenLoads}}{{.}} {{end}}
</open_loads>
<email>
From: {{.From}}
Subject: {{.Subject}}
{{clip .Body 4000}}
</email>
`

// fenceTag matches anything a reader could take for
// our own <email> or </email> tag, in any case and
// spacing: "</email>", "</EMAIL >", "< / email".
var fenceTag = regexp.MustCompile(`(?i)<\s*/?\s*email`)

// defang rewrites the "<" of every fence-like tag in
// a sender value to "&lt;", so the fence closes only
// where the template closes it. Named only in prose;
// minimal. It keeps the fence intact; it is not a
// prompt-injection defence: instruction-like text
// still reaches the model, inside the fence (Move 94).
func defang(s string) string {
	return fenceTag.ReplaceAllStringFunc(s,
		func(m string) string { return "&lt;" + m[1:] })
}

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
