package exceptionqueue

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// fixture builds a temp table (gone at ROLLBACK)
// with resolved, open and out-of-window rows.
const fixture = `
BEGIN;
CREATE TEMP TABLE invoice_exceptions (
  reason           text NOT NULL,
  minutes_to_clear %s,
  created_at       timestamptz NOT NULL
);
INSERT INTO invoice_exceptions VALUES
  ('detention', 30, now()), ('detention', 40, now()),
  ('detention', 50, now()), ('detention', 60, now()),
  ('detention', 70, now()),
  ('detention', NULL, now()), ('detention', NULL, now()),
  ('detention', 500, now() - interval '40 days'),
  ('tonu', 10, now()), ('tonu', 20, now()),
  ('tonu', 40, now()), ('tonu', 50, now()),
  ('lumper', NULL, now()), ('lumper', NULL, now()),
  ('lumper', NULL, now());
INSERT INTO invoice_exceptions
  SELECT 'fuel', 10, now() FROM generate_series(1, 10);
`

func TestQueryRanksResolvedWork(t *testing.T) {
	conn := os.Getenv("UNATTENDED_TEST_PSQL_URL")
	if conn == "" {
		t.Skip("set UNATTENDED_TEST_PSQL_URL to run against Postgres")
	}
	q, err := os.ReadFile("query.sql")
	if err != nil {
		t.Fatal(err)
	}
	// detention: 5 resolved, median 50 -> 250; 2 open;
	//   the 40-day-old row is outside the window.
	// tonu: 10,20,40,50 -> percentile_cont interpolates
	//   to 30 -> 120.
	// fuel: 10 x 10 min -> 100.
	// lumper: all open -> no median, ranked last.
	want := "detention,5,2,50\ntonu,4,0,30\n" +
		"fuel,10,0,10\nlumper,0,3,\n"
	for _, ty := range []string{"integer", "numeric",
		"double precision"} {
		script := strings.Replace(fixture, "%s", ty, 1) +
			string(q) + "\nROLLBACK;\n"
		cmd := exec.Command("psql", conn, "-X", "-q", "-A",
			"-t", "-F", ",", "-v", "ON_ERROR_STOP=1", "-f", "-")
		cmd.Stdin = strings.NewReader(script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: psql: %v\n%s", ty, err, out)
		}
		if string(out) != want {
			t.Errorf("%s: got\n%s\nwant\n%s", ty, out, want)
		}
	}
}
