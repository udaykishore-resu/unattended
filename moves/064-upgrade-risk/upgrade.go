// Package upgrade collects the facts a dependency
// risk note is written from: the jump size, every
// version in between and which of them lack notes,
// our importers, and the vulns the upgrade fixes.
//
// Move 64: dependency upgrades with a plain-English
// risk note. The book lists these facts in prose;
// this is a minimal implementation of each.
package upgrade

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type ver struct {
	maj, min, pat int
	pre           bool
}

func parse(v string) (ver, bool) {
	v = strings.TrimSuffix(strings.TrimPrefix(v, "v"), "+incompatible")
	core, _, pre := strings.Cut(v, "-")
	p := strings.Split(core, ".")
	if len(p) != 3 {
		return ver{}, false
	}
	var n [3]int
	for i, s := range p {
		x, err := strconv.Atoi(s)
		if err != nil {
			return ver{}, false
		}
		n[i] = x
	}
	return ver{n[0], n[1], n[2], pre}, true
}

func cmp(a, b ver) int {
	if a.maj != b.maj {
		return a.maj - b.maj
	}
	if a.min != b.min {
		return a.min - b.min
	}
	return a.pat - b.pat
}

var pseudo = regexp.MustCompile(`\d{14}-[0-9a-f]{12}$`)

// Jump returns patch, minor, major or human.
// Halcyon's review rule, stricter than semver: any
// v0 change counts as major, because v0 promises no
// compatibility, and a pseudo-version goes to a
// person.
func Jump(from, to string) string {
	if pseudo.MatchString(from) || pseudo.MatchString(to) {
		return "human"
	}
	a, ok1 := parse(from)
	b, ok2 := parse(to)
	switch {
	case !ok1 || !ok2:
		return "human"
	case a.maj == 0 || b.maj == 0 || a.maj != b.maj:
		return "major"
	case a.min != b.min:
		return "minor"
	}
	return "patch"
}

// Between returns every release after from, up to
// and including to, oldest first. Pre-releases are
// skipped.
func Between(all []string, from, to string) []string {
	a, _ := parse(from)
	b, _ := parse(to)
	var out []string
	for _, v := range all {
		x, ok := parse(v)
		if ok && !x.pre && cmp(x, a) > 0 && cmp(x, b) <= 0 {
			out = append(out, v)
		}
	}
	slices.SortFunc(out, func(p, q string) int {
		x, _ := parse(p)
		y, _ := parse(q)
		return cmp(x, y)
	})
	return out
}

// Gaps names versions in between that have no
// release notes, so the note can say so instead of
// calling the jump low risk.
func Gaps(between []string, notes map[string]string) []string {
	var out []string
	for _, v := range between {
		if strings.TrimSpace(notes[v]) == "" {
			out = append(out, v)
		}
	}
	return out
}

// Importers reads `go list -json ./...` output and
// returns our packages that import mod or any of its
// packages, test imports included.
func Importers(goList []byte, mod string) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(goList))
	var out []string
	for {
		var p struct {
			ImportPath                         string
			Imports, TestImports, XTestImports []string
		}
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		all := slices.Concat(p.Imports, p.TestImports, p.XTestImports)
		if slices.ContainsFunc(all, func(i string) bool {
			return i == mod || strings.HasPrefix(i, mod+"/")
		}) {
			out = append(out, p.ImportPath)
		}
	}
}

// Fixed returns vuln IDs reported for the old version
// and not the new one. Both lists must come from
// scans against the same vuln database snapshot.
func Fixed(before, after []string) []string {
	var out []string
	for _, id := range before {
		if !slices.Contains(after, id) {
			out = append(out, id)
		}
	}
	return out
}
