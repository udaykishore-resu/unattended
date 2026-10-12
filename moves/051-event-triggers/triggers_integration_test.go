//go:build integration

package triggers

import (
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/internal/psqltest"
)

// Runs insertEvent verbatim in Postgres:
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test -tags integration ./moves/051-event-triggers/
func TestInsertEventInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	if got := db.Bind(insertEvent, "mail", "m-1", "a"); got != "INSERT 0 1" {
		t.Fatalf("first: %q", got)
	}
	// A redelivery is dropped by the (source, event_id) key.
	if got := db.Bind(insertEvent, "mail", "m-1", "a"); got != "INSERT 0 0" {
		t.Fatalf("redelivery: %q", got)
	}
	// Same ID from another source is a different event.
	if got := db.Bind(insertEvent, "github", "m-1", "b"); got != "INSERT 0 1" {
		t.Fatalf("other source: %q", got)
	}
	// An empty ID is refused by the table too.
	if out, err := db.Try(insertEvent + " " + psqltest.BindArgs("mail", "", "c") + "\n"); err == nil {
		t.Fatalf("empty id stored: %s", out)
	}
}

// Without the unique key the same insert is an error,
// not a silent duplicate: the constraint is required.
func TestInsertNeedsTheUniqueKey(t *testing.T) {
	db := psqltest.New(t, `CREATE TABLE events (
  source text, event_id text, body bytea)`)
	out, err := db.Try(insertEvent + " " + psqltest.BindArgs("mail", "m-1", "a") + "\n")
	if err == nil || !strings.Contains(out, "no unique or exclusion constraint") {
		t.Fatalf("got %v %s", err, out)
	}
}
