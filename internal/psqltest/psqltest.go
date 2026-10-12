// Package psqltest runs SQL in a real Postgres
// through the psql CLI (psql 16+ for \bind), so the
// integration tests can run a move's query constants
// verbatim with no Go driver dependency. It is not
// part of the book. Each test gets a fresh schema.
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test -tags integration ./moves/...
package psqltest

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// DB is one test's schema in the test database.
type DB struct {
	t           *testing.T
	url, schema string
}

var seq atomic.Int64

// Env lists the variable that holds the test
// database URL.
var Env = []string{"UNATTENDED_TEST_PSQL_URL"}

// New creates a fresh schema and runs ddl in it. The
// test is skipped when none of Env is set.
func New(t *testing.T, ddl string) *DB {
	t.Helper()
	var url string
	for _, e := range Env {
		if url = os.Getenv(e); url != "" {
			break
		}
	}
	if url == "" {
		t.Skip("set UNATTENDED_TEST_PSQL_URL to run against Postgres")
	}
	d := &DB{t: t, url: url, schema: fmt.Sprintf(
		"t%d_%d", time.Now().UnixNano(), seq.Add(1))}
	d.Run("CREATE SCHEMA " + d.schema + ";")
	t.Cleanup(func() {
		d.Run("DROP SCHEMA " + d.schema + " CASCADE;")
	})
	d.Run(d.path() + ddl + ";")
	return d
}

func (d *DB) path() string {
	return "SET search_path = " + d.schema + ";\n"
}

func (d *DB) cmd(script string) *exec.Cmd {
	c := exec.Command("psql", d.url, "-X", "-A", "-t",
		"-v", "ON_ERROR_STOP=1")
	c.Stdin = strings.NewReader(script)
	return c
}

// Run runs script and returns psql's output: rows,
// then command tags such as "INSERT 0 1".
func (d *DB) Run(script string) string {
	d.t.Helper()
	out, err := d.cmd(d.path() + script).CombinedOutput()
	if err != nil {
		d.t.Fatalf("psql: %v\n%s", err, out)
	}
	return strings.TrimSpace(
		strings.TrimPrefix(string(out), "SET\n"))
}

// Try is Run that returns the error instead of
// failing the test.
func (d *DB) Try(script string) (string, error) {
	out, err := d.cmd(d.path() + script).CombinedOutput()
	return string(out), err
}

// Bind runs one query verbatim with \bind args.
func (d *DB) Bind(q string, args ...string) string {
	d.t.Helper()
	return d.Run(q + " " + BindArgs(args...) + "\n")
}

// BindArgs renders \bind ... \g for psql.
func BindArgs(args ...string) string {
	var b strings.Builder
	b.WriteString(`\bind`)
	for _, a := range args {
		b.WriteString(" '" +
			strings.ReplaceAll(a, "'", "''") + "'")
	}
	b.WriteString(` \g`)
	return b.String()
}

// Session is a psql process running in the
// background, for races between two connections.
type Session struct {
	c   *exec.Cmd
	out strings.Builder
}

// Start runs script in a second connection without
// waiting for it.
func (d *DB) Start(script string) *Session {
	d.t.Helper()
	s := &Session{c: d.cmd(d.path() + script)}
	s.c.Stdout, s.c.Stderr = &s.out, &s.out
	if err := s.c.Start(); err != nil {
		d.t.Fatal(err)
	}
	return s
}

// Wait waits for the session and returns its output.
func (s *Session) Wait(t *testing.T) string {
	t.Helper()
	if err := s.c.Wait(); err != nil {
		t.Fatalf("psql session: %v\n%s", err, s.out.String())
	}
	return strings.TrimSpace(
		strings.TrimPrefix(s.out.String(), "SET\n"))
}
