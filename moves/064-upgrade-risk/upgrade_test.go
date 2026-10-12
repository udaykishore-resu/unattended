package upgrade

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestJumpHalcyonRule(t *testing.T) {
	cases := []struct{ from, to, want string }{
		{"v1.4.1", "v1.4.3", "patch"},
		{"v1.4.3", "v1.6.0", "minor"},
		{"v1.9.0", "v2.0.0", "major"},
		// Halcyon policy: v0 promises nothing, so any
		// v0 change is reviewed as major.
		{"v0.3.1", "v0.3.2", "major"},
		{"v0.9.0", "v1.0.0", "major"},
		{"v1.2.3", "v1.2.4-0.20260301120000-abcdef123456", "human"},
		{"v0.0.0-20260101000000-0123456789ab", "v0.1.0", "human"},
		{"v2.1.0+incompatible", "v2.2.0+incompatible", "minor"},
	}
	for _, c := range cases {
		if got := Jump(c.from, c.to); got != c.want {
			t.Errorf("Jump(%s, %s) = %s, want %s", c.from, c.to, got, c.want)
		}
	}
}

// Breaking changes hide in the middle releases: the
// facts cover every version in between, and code
// names the ones whose notes are missing.
func TestEveryVersionInBetween(t *testing.T) {
	all := []string{"v5.7.0", "v5.5.0", "v5.5.4", "v5.6.0-rc.1", "v5.6.0",
		"v5.5.5", "v5.7.1", "v4.18.0"}
	got := Between(all, "v5.5.4", "v5.7.0")
	want := []string{"v5.5.5", "v5.6.0", "v5.7.0"}
	if !slices.Equal(got, want) {
		t.Fatalf("between = %v", got)
	}
	notes := map[string]string{"v5.5.5": "fix", "v5.7.0": "pool closes idle conns"}
	if g := Gaps(got, notes); !slices.Equal(g, []string{"v5.6.0"}) {
		t.Fatalf("gaps = %v", g)
	}
}

func TestFixedVulns(t *testing.T) {
	got := Fixed([]string{"GO-2026-0101", "GO-2026-0202"}, []string{"GO-2026-0202"})
	if !slices.Equal(got, []string{"GO-2026-0101"}) {
		t.Fatal(got)
	}
}

// Importer discovery from the real `go list -json`:
// direct imports, subpackages, and test-only imports
// all count; a package that doesn't import it is out.
func TestImportersFromGoList(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go tool")
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module halcyon.test/dispatch\n\ngo 1.26\n\n" +
			"require example.com/pgdriver v1.0.0\n\n" +
			"replace example.com/pgdriver => ./pgdriver\n",
		"pgdriver/go.mod":       "module example.com/pgdriver\n\ngo 1.26\n",
		"pgdriver/driver.go":    "package pgdriver\n\nconst Name = \"pg\"\n",
		"pgdriver/pool/pool.go": "package pool\n\nconst Size = 4\n",
		"store/store.go":        "package store\n\nimport _ \"example.com/pgdriver\"\n",
		"relay/relay.go":        "package relay\n\nimport _ \"example.com/pgdriver/pool\"\n",
		"quote/quote.go":        "package quote\n",
		"quote/quote_test.go":   "package quote\n\nimport _ \"example.com/pgdriver\"\n",
		"api/api.go":            "package api\n",
	}
	for n, s := range files {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not found")
	}
	cmd := exec.Command(goBin, "list", "-json", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOTOOLCHAIN=local", "GOPROXY=off")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	got, err := Importers(out, "example.com/pgdriver")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"halcyon.test/dispatch/quote", "halcyon.test/dispatch/relay",
		"halcyon.test/dispatch/store"}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("importers = %v", got)
	}
}
