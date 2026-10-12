package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run feeds csv to main on stdin and returns stdout.
func run(t *testing.T, csv string) string {
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
	oldIn, oldOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = f, w
	defer func() { os.Stdin, os.Stdout = oldIn, oldOut }()
	main()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// Check calls: a 45 s median and a slow tail, the
// gap the book says is "where judgment lives".
func TestP50AndP90(t *testing.T) {
	var b strings.Builder
	b.WriteString("task,seconds\n")
	for i := range 9 {
		fmt.Fprintf(&b, "checkcall,%d\n", 40+i) // 40..48
	}
	b.WriteString("checkcall,410\n")
	b.WriteString("pod,120\npod,150\npod,135\n")
	out := run(t, b.String())

	if !strings.Contains(out, "checkcall    n=10  p50=  45s p90= 410s") {
		t.Errorf("checkcall line wrong:\n%s", out)
	}
	if !strings.Contains(out, "pod          n=3   p50= 135s p90= 150s") {
		t.Errorf("pod line wrong:\n%s", out)
	}
}
