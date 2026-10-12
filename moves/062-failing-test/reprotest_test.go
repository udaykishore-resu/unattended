package reprotest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	name = "TestQuoteTwoStops"
	mark = "BUG-4127"
)

func ev(action, output string) string {
	return `{"Action":"` + action + `","Test":"` + name +
		`","Output":` + quote(output) + "}\n"
}

func quote(s string) string {
	b := []byte{'"'}
	for _, r := range s {
		if r == '\n' {
			b = append(b, `\n`...)
			continue
		}
		b = append(b, string(r)...)
	}
	return string(append(b, '"'))
}

func TestVerdicts(t *testing.T) {
	cases := map[string]struct{ out, want string }{
		"clean failure": {ev("run", "") +
			ev("output", "quote_test.go:12: BUG-4127: total = 410000, want 412500\n") +
			ev("fail", ""), "reproduces"},
		"passes": {ev("run", "") + ev("pass", ""), "passes"},
		"panic": {ev("run", "") +
			ev("output", "panic: runtime error: index out of range\n") +
			ev("fail", ""), "broken"},
		// A build error never reaches the named test.
		"build error": {`{"Action":"output","Package":"q","Output":"undefined: Stop"}` +
			"\n" + `{"Action":"fail","Package":"q"}` + "\n", "broken"},
	}
	for label, c := range cases {
		if got := Classify([]byte(c.out), name, mark); got != c.want {
			t.Errorf("%s: got %s, want %s", label, got, c.want)
		}
	}
}

// Real `go test -json` output, from a throwaway
// module with one failing test.
func TestRealGoTestJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go tool")
	}
	dir := t.TempDir()
	write := func(n, s string) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module q\n\ngo 1.26\n")
	write("q.go", "package q\n\nfunc Total(stops int) int { return 400000 }\n")
	write("q_test.go", "package q\n\nimport \"testing\"\n\n"+
		"func "+name+"(t *testing.T) {\n"+
		"\tif got := Total(2); got != 412500 {\n"+
		"\t\tt.Fatalf(\"BUG-4127: total = %d, want 412500\", got)\n\t}\n}\n")
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(goBin); err != nil {
		if goBin, err = exec.LookPath("go"); err != nil {
			t.Skip("go tool not found")
		}
	}
	cmd := exec.Command(goBin, "test", "-json", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOTOOLCHAIN=local")
	out, _ := cmd.Output() // a failing test exits non-zero
	if got := Classify(out, name, mark); got != "reproduces" {
		t.Fatalf("got %s from:\n%s", got, out)
	}
}
