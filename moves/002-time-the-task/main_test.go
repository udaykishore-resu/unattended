package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run feeds csv to main on stdin and returns stdout
// and what main logged.
func run(t *testing.T, csv string) (string, string) {
	t.Helper()
	in := filepath.Join(t.TempDir(), "in.csv")
	if err := os.WriteFile(in, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	var logged bytes.Buffer
	log.SetOutput(&logged)
	log.SetFlags(0)
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = f, w
	defer func() {
		os.Stdin, os.Stdout = oldIn, oldOut
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()
	main()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out), logged.String()
}

// Check calls: a 45 s median and a 410 s p90, the
// 9x gap the book says is "where judgment lives".
func TestHalcyonCheckCalls(t *testing.T) {
	csv := "task,seconds\n" +
		"checkcall,45\ncheckcall,40\ncheckcall,620\n" +
		"checkcall,44\ncheckcall,410\ncheckcall,46\n" +
		"checkcall,42\ncheckcall,48\ncheckcall,45\n" +
		"checkcall,50\n" +
		"pod,120\npod,150\npod,135\n"
	out, _ := run(t, csv)
	want := "checkcall    n=10  p50=  45s p90= 410s\n" +
		"pod          n=3   p50= 135s p90= 150s\n"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

// Nearest rank: xs[ceil(p*n/100)-1]. For even n the
// p50 is the lower middle value; below ten rows the
// p90 is the slowest case; at ten it is the 9th.
func TestRankNearestRank(t *testing.T) {
	seq := func(n int) []float64 {
		xs := make([]float64, n)
		for i := range xs {
			xs[i] = float64(i + 1)
		}
		return xs
	}
	cases := []struct {
		n, p int
		want float64
	}{
		{1, 50, 1}, {1, 90, 1},
		{2, 50, 1}, // lower middle, not an average
		{3, 50, 2},
		{4, 50, 2},
		{9, 90, 9},  // under ten rows: the max
		{10, 90, 9}, // not the max
		{10, 50, 5},
		{20, 90, 18},
		{100, 90, 90},
	}
	for _, c := range cases {
		if got := rank(seq(c.n), c.p); got != c.want {
			t.Errorf("rank(n=%d, p%d) = %v, want %v",
				c.n, c.p, got, c.want)
		}
	}
}

// Blank lines, short rows and typos are counted and
// skipped; they never panic the program.
func TestMalformedRowsCounted(t *testing.T) {
	csv := "task,seconds\n" +
		"\n" + // csv.Reader drops blank lines
		"checkcall\n" + // short row
		"pod,abc\n" + // typo
		"pod,120,late\n" + // extra field still parses
		"pod,130\n"
	out, logged := run(t, csv)
	if out != "pod          n=2   p50= 120s p90= 130s\n" {
		t.Errorf("out = %q", out)
	}
	if !strings.Contains(logged, "skipped 3 rows") {
		t.Errorf("logged = %q, want 3 skipped (header, short row, typo)", logged)
	}
}

// Output is sorted by task name, so two runs diff
// cleanly.
func TestSortedStableOutput(t *testing.T) {
	csv := "zeta,1\nalpha,2\nmid,3\nbeta,4\n"
	first, _ := run(t, csv)
	for range 20 {
		if again, _ := run(t, csv); again != first {
			t.Fatalf("output changed between runs:\n%s\n%s", first, again)
		}
	}
	var names []string
	for _, l := range strings.Split(strings.TrimSpace(first), "\n") {
		names = append(names, strings.Fields(l)[0])
	}
	if strings.Join(names, ",") != "alpha,beta,mid,zeta" {
		t.Fatalf("order = %v", names)
	}
}
