package cost

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// run executes the fixture and one query inside a
// throwaway schema and returns workflow|total|per_win.
func run(t *testing.T, query string) []string {
	t.Helper()
	url := os.Getenv("UNATTENDED_TEST_PSQL_URL")
	if url == "" {
		t.Skip("set UNATTENDED_TEST_PSQL_URL")
	}
	if _, err := exec.LookPath("psql"); err != nil {
		t.Skip("psql not installed")
	}
	fixture, err := os.ReadFile("testdata/fixture.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("move88_%d", time.Now().UnixNano())
	script := "CREATE SCHEMA " + schema + "; SET search_path = " + schema + ";\n" +
		string(fixture) + "\n\\pset format unaligned\n\\pset tuples_only on\n" +
		query + "\nDROP SCHEMA " + schema + " CASCADE;\n"
	cmd := exec.Command("psql", url, "-X", "-q", "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql: %v\n%s", err, out)
	}
	var rows []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(l, "|") {
			rows = append(rows, l)
		}
	}
	return rows
}

func parse(t *testing.T, row string) (string, float64, float64) {
	t.Helper()
	f := strings.Split(row, "|")
	total, err1 := strconv.ParseFloat(f[1], 64)
	per, err2 := strconv.ParseFloat(f[2], 64)
	if err1 != nil || err2 != nil {
		t.Fatalf("row %q", row)
	}
	return f[0], total, per
}

// Hand-priced from the fixture (rates per million):
//
//	r1: 2000*1 + 5000*0.1 + 5000*1.25 + 100*4 = 9150
//	r2: small 2000*1 + 50*4 = 2200; large 2000*10 + 50*40 = 22000
//	r3: 1000; r4: 1000; r5 is outside the window.
const (
	wantTotal  = (9150 + 2200 + 22000 + 1000 + 1000) / 1e6
	wantPerWin = wantTotal / 2 // winners: r1, and r3's newest outcome
)

func TestCostPerWin(t *testing.T) {
	q, err := os.ReadFile("cost.sql")
	if err != nil {
		t.Fatal(err)
	}
	rows := run(t, string(q))
	if len(rows) != 1 {
		t.Fatalf("rows = %q", rows)
	}
	wf, total, per := parse(t, rows[0])
	if wf != "triage" || math.Abs(total-wantTotal) > 1e-9 ||
		math.Abs(per-wantPerWin) > 1e-9 {
		t.Fatalf("got %s total=%v per_win=%v; want %v, %v",
			wf, total, per, wantTotal, wantPerWin)
	}
}

// The draft query joined run_outcomes directly. With
// two outcome rows per run it counts that run's cost
// twice and its wins twice: the review's bug.
func TestRawJoinDoublesCost(t *testing.T) {
	q, err := os.ReadFile("cost.sql")
	if err != nil {
		t.Fatal(err)
	}
	s := string(q)
	i := strings.Index(s, "), latest AS")
	j := strings.Index(s, "\nSELECT r.workflow")
	raw := s[:i] + ")" + s[j:]
	raw = strings.Replace(raw, "LEFT JOIN latest o", "LEFT JOIN run_outcomes o", 1)
	_, total, _ := parse(t, run(t, raw)[0])
	if total <= wantTotal {
		t.Fatalf("raw join total %v; expected inflated above %v", total, wantTotal)
	}
}
