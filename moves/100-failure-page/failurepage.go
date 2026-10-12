// Package failurepage is the CI check for the "how
// this automation fails" page: it fails the build when
// the page is unfinished, misses a section, or names
// a kill switch flag that no longer exists in code.
//
// Move 100: write the "how this automation fails"
// page before launch. The book prints the page
// template; this check is described in its prose.
package failurepage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// required lists each section and the lines it must
// fill in. STOP IT names two ways to stop, who may
// act, and how to confirm it worked; OWNERS names
// the escalation path.
var required = map[string][]string{
	"STOP IT": {"Kill switch:", "Second way:",
		"Who may flip it:", "Stopped when:", "Effect:",
		"Safe to leave off for:"},
	"WHAT NORMAL LOOKS LIKE": {"Runs per hour:",
		"Review rate:", "Dashboard:"},
	"KNOWN FAILURE MODES": {"->"},
	"RECOVERY":            {"Replay from:", "Idempotent:"},
	"OWNERS":              {"Business:", "Code:", "Escalate:"},
}

var placeholder = regexp.MustCompile(`<[^>\n]+>`)
var flagRe = regexp.MustCompile("Kill switch:.*?flag `([a-z0-9_]+)`")

// Check returns every problem with the page. flags is
// the set of kill switch names defined in code.
func Check(page string, flags map[string]bool) []error {
	var errs []error
	if m := placeholder.FindString(page); m != "" {
		errs = append(errs, fmt.Errorf("unfilled %s", m))
	}
	secs := sections(page)
	for sec, keys := range required {
		body, ok := secs[sec]
		if !ok {
			errs = append(errs, fmt.Errorf("missing %s", sec))
			continue
		}
		for _, k := range keys {
			if !filled(body, k) {
				errs = append(errs,
					fmt.Errorf("%s: %q is empty", sec, k))
			}
		}
	}
	m := flagRe.FindStringSubmatch(secs["STOP IT"])
	switch {
	case m == nil:
		errs = append(errs, fmt.Errorf(
			"kill switch names no flag in backticks"))
	case !flags[m[1]]:
		errs = append(errs, fmt.Errorf(
			"kill switch flag %q not found in code", m[1]))
	}
	return errs
}

func sections(page string) map[string]string {
	out := map[string]string{}
	cur := ""
	for _, ln := range strings.Split(page, "\n") {
		if t := strings.TrimSpace(ln); t != "" &&
			t == strings.ToUpper(t) && required[t] != nil {
			cur = t
			out[cur] = ""
			continue
		}
		if cur != "" {
			out[cur] += ln + "\n"
		}
	}
	return out
}

// filled reports whether key appears with text after it.
func filled(body, key string) bool {
	i := strings.Index(body, key)
	if i < 0 {
		return false
	}
	rest := body[i+len(key):]
	if j := strings.IndexByte(rest, '\n'); j >= 0 {
		rest = rest[:j]
	}
	if key == "->" { // a failure line: symptom -> cause -> action
		return strings.Count(body[:i+len(key)+len(rest)], "->") >= 2
	}
	return strings.TrimSpace(rest) != ""
}

// FlagsInCode collects the Name of every kill switch
// Gate literal (Move 7) in the Go files under dir.
func FlagsInCode(dir string) (map[string]bool, error) {
	flags := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(p string,
		d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() ||
			!strings.HasSuffix(p, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok || !isGate(cl.Type) {
				return true
			}
			for _, el := range cl.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				k, _ := kv.Key.(*ast.Ident)
				v, _ := kv.Value.(*ast.BasicLit)
				if k != nil && k.Name == "Name" && v != nil {
					if s, err := strconv.Unquote(v.Value); err == nil {
						flags[s] = true
					}
				}
			}
			return true
		})
		return nil
	})
	return flags, err
}

func isGate(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name == "Gate"
	case *ast.SelectorExpr:
		return t.Sel.Name == "Gate"
	}
	return false
}
