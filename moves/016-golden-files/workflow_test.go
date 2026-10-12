package golden

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenWorkflow runs TestPromptGolden in a child
// process against scratch copies of testdata, to show
// how the printed test behaves when a golden exists,
// is missing, is stale or cannot be written, and what
// -update does in each case. Every failure is loud.
func TestGoldenWorkflow(t *testing.T) {
	if os.Getenv("GOLDEN_CHILD") != "" {
		t.Skip("child process")
	}
	type tc struct {
		name    string
		setup   func(t *testing.T, dir string)
		update  bool
		wantOK  bool
		wantOut string
		check   func(t *testing.T, dir string)
	}
	cases := []tc{
		{name: "all goldens present", wantOK: true},
		{name: "golden missing", wantOK: false, wantOut: "no such file",
			setup: func(t *testing.T, dir string) { rm(t, dir, "spanish.golden") }},
		{name: "golden missing, -update creates it", update: true, wantOK: true,
			setup: func(t *testing.T, dir string) { rm(t, dir, "spanish.golden") },
			check: func(t *testing.T, dir string) {
				in := read(t, filepath.Join(dir, "testdata", "spanish.input"))
				got := read(t, filepath.Join(dir, "testdata", "spanish.golden"))
				if got != Render(in) {
					t.Fatal("-update wrote the wrong bytes")
				}
			}},
		{name: "golden stale", wantOK: false, wantOut: "prompt changed; run -update",
			setup: func(t *testing.T, dir string) {
				write(t, filepath.Join(dir, "testdata", "spanish.golden"), "old prompt")
			}},
		{name: "golden stale, -update rewrites it", update: true, wantOK: true,
			setup: func(t *testing.T, dir string) {
				write(t, filepath.Join(dir, "testdata", "spanish.golden"), "old prompt")
			}},
		{name: "golden cannot be written", update: true, wantOK: false,
			wantOut: "is a directory",
			setup: func(t *testing.T, dir string) {
				// A directory where the file should be: WriteFile fails
				// even for root, so the failure cannot pass as "unchanged".
				rm(t, dir, "spanish.golden")
				if err := os.Mkdir(filepath.Join(dir, "testdata", "spanish.golden"), 0o755); err != nil {
					t.Fatal(err)
				}
			}},
		{name: "golden cannot be read", wantOK: false, wantOut: "is a directory",
			setup: func(t *testing.T, dir string) {
				rm(t, dir, "spanish.golden")
				if err := os.Mkdir(filepath.Join(dir, "testdata", "spanish.golden"), 0o755); err != nil {
					t.Fatal(err)
				}
			}},
		{name: "no inputs at all", wantOK: false, wantOut: "no inputs",
			setup: func(t *testing.T, dir string) {
				for _, n := range []string{"spanish", "closing-tag", "no-load-list"} {
					rm(t, dir, n+".input")
				}
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			copyTestdata(t, dir)
			if c.setup != nil {
				c.setup(t, dir)
			}
			args := []string{"-test.run=^TestPromptGolden$", "-test.count=1"}
			if c.update {
				args = append(args, "-update")
			}
			cmd := exec.Command(os.Args[0], args...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GOLDEN_CHILD=1")
			out, err := cmd.CombinedOutput()
			if ok := err == nil; ok != c.wantOK {
				t.Fatalf("ok=%v, want %v:\n%s", ok, c.wantOK, out)
			}
			if c.wantOut != "" && !bytes.Contains(out, []byte(c.wantOut)) {
				t.Fatalf("output lacks %q:\n%s", c.wantOut, out)
			}
			if c.check != nil {
				c.check(t, dir)
			}
		})
	}
}

func copyTestdata(t *testing.T, dir string) {
	t.Helper()
	src, err := filepath.Glob("testdata/*")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, s := range src {
		write(t, filepath.Join(dir, "testdata", filepath.Base(s)), read(t, s))
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rm(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, "testdata", name)); err != nil && !strings.Contains(err.Error(), "no such file") {
		t.Fatal(err)
	}
}
