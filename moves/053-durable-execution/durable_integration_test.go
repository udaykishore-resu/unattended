//go:build integration

package durable

import (
	"strings"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/psqltest"
)

// Runs the journal and run-claim queries verbatim in
// Postgres:
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test -tags integration ./moves/053-durable-execution/
func TestJournalInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	if got := db.Bind(putStep, "q-1", "ask-0", "first"); got != "INSERT 0 1" {
		t.Fatalf("put: %q", got)
	}
	// A second write for the same step is refused, so
	// a racing replay cannot overwrite the journal.
	out, err := db.Try(putStep + " " + psqltest.BindArgs("q-1", "ask-0", "second") + "\n")
	if err == nil || !strings.Contains(out, "duplicate key") {
		t.Fatalf("second put: %v %s", err, out)
	}
	if got := db.Bind(getStep, "q-1", "ask-0"); got != `\x6669727374` {
		t.Fatalf("get: %q", got)
	}
}

func TestRunClaimInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	check := func(got, want string) {
		t.Helper()
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	check(db.Bind(claimRunQ, "q-1", "w1", future), "INSERT 0 1")
	check(db.Bind(claimRunQ, "q-1", "w2", future), "INSERT 0 0") // busy
	check(db.Bind(claimRunQ, "q-1", "w1", future), "INSERT 0 1") // renew
	db.Run("UPDATE wf_runs SET lease_until = now() - interval '1 second';")
	check(db.Bind(claimRunQ, "q-1", "w2", future), "INSERT 0 1") // takeover
}
