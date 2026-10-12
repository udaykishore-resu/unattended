package reprotest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// goTestJSON writes a throwaway module and runs the
// real `go test -json` on the named test, the way the
// sandbox runner does.
func goTestJSON(t *testing.T, files map[string]string) []byte {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module q\n\ngo 1.26\n"
	for n, s := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(goBin); err != nil {
		if goBin, err = exec.LookPath("go"); err != nil {
			t.Skip("go tool not found")
		}
	}
	cmd := exec.Command(goBin, "test", "-json", "-count=1",
		"-run", "^"+name+"$", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOTOOLCHAIN=local")
	out, _ := cmd.Output() // failing tests exit non-zero
	return out
}

const code = "package q\n\nfunc Total(stops int) int { return 400000 }\n"

// The generated test: assertion message starts with
// the ticket ID, so only this assertion can produce
// the mark.
func genTest(body string) string {
	return "package q\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
		"var _ = os.Exit\n\n" +
		"func " + name + "(t *testing.T) {\n" + body +
		"\tif got := Total(2); got != 412500 {\n" +
		"\t\tt.Fatalf(\"BUG-4127: total = %d, want 412500\", got)\n\t}\n}\n"
}

func TestRealFailureModes(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go tool")
	}
	cases := []struct {
		label string
		files map[string]string
		want  string
	}{
		{"clean failure on the named assertion",
			map[string]string{"q.go": code, "q_test.go": genTest("")},
			"reproduces"},
		{"fixture fails before the assertion",
			map[string]string{"q.go": code, "q_test.go": genTest(
				"\tif _, err := os.ReadFile(\"testdata/carrier.json\"); err != nil {\n" +
					"\t\tt.Fatal(err)\n\t}\n")},
			"broken"},
		{"teardown fails, assertion would pass",
			map[string]string{
				"q.go": "package q\n\nfunc Total(stops int) int { return 412500 }\n",
				"q_test.go": genTest(
					"\tt.Cleanup(func() { t.Error(\"cleanup: temp dir busy\") })\n")},
			"broken"},
		{"build failure never reaches the test",
			map[string]string{"q.go": code, "q_test.go": genTest("\tundefinedHelper()\n")},
			"broken"},
		{"package-level failure in TestMain",
			map[string]string{"q.go": code, "q_test.go": genTest(""),
				"main_test.go": "package q\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
					"func TestMain(m *testing.M) { os.Exit(1) }\n"},
			"broken"},
		{"panic in the package under test",
			map[string]string{
				"q.go":      "package q\n\nfunc Total(stops int) int { var m map[int]int; m[stops] = 1; return 0 }\n",
				"q_test.go": genTest("")},
			"broken"}, // the book's "Breaks when": a person decides
		{"bug already fixed",
			map[string]string{
				"q.go":      "package q\n\nfunc Total(stops int) int { return 412500 }\n",
				"q_test.go": genTest("")},
			"passes"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			out := goTestJSON(t, c.files)
			if got := Classify(out, name, mark); got != c.want {
				t.Fatalf("got %s, want %s from:\n%s", got, c.want, out)
			}
		})
	}
}

// A failure in another test in the same output does
// not count, and neither does an empty mark.
func TestOtherTestAndEmptyMark(t *testing.T) {
	other := `{"Action":"output","Test":"TestOther","Output":"BUG-4127: x\n"}` + "\n" +
		`{"Action":"fail","Test":"TestOther"}` + "\n"
	if got := Classify([]byte(other), name, mark); got != "broken" {
		t.Fatalf("other test: got %s", got)
	}
	own := ev("output", "quote_test.go:12: total = 1\n") + ev("fail", "")
	if got := Classify([]byte(own), name, ""); got != "broken" {
		t.Fatalf("empty mark: got %s", got)
	}
}
