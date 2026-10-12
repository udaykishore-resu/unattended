//go:build integration

package idem

import (
	"strings"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/psqltest"
)

// Runs the four queries verbatim in Postgres:
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test -tags integration ./moves/052-idempotency-keys/
func TestQueriesInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	expire := "UPDATE ai_steps SET lease_until = now() - interval '1 second';"
	check := func(got, want string) {
		t.Helper()
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	check(db.Bind(claimQ, "k", future, "A"), "INSERT 0 1")
	check(db.Bind(claimQ, "k", future, "B"), "INSERT 0 0") // live lease
	check(db.Bind(releaseQ, "k", "B"), "UPDATE 0")         // not B's to drop
	db.Run(expire)
	check(db.Bind(claimQ, "k", future, "B"), "INSERT 0 1") // takeover
	check(db.Bind(releaseQ, "k", "A"), "UPDATE 0")         // stale holder
	check(db.Bind(savedQ, "k"), "")                        // NULL: in flight
	// B finishes, then stale A: the first output stays.
	check(db.Bind(finishQ, "k", "from B"), `\x66726f6d2042`+"\nUPDATE 1")
	check(db.Bind(finishQ, "k", "from A"), `\x66726f6d2042`+"\nUPDATE 1")
	// A finished key is never taken over or released.
	db.Run(expire)
	check(db.Bind(claimQ, "k", future, "C"), "INSERT 0 0")
	check(db.Bind(releaseQ, "k", "B"), "UPDATE 0")
	check(db.Bind(savedQ, "k"), `\x66726f6d2042`)
}

// Two connections claim one new key at the same
// moment: the row lock serialises them and exactly
// one wins.
func TestConcurrentClaimInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	a := db.Start("BEGIN;\n" + claimQ + " " +
		psqltest.BindArgs("k", future, "A") +
		"\nSELECT pg_sleep(0.5);\nCOMMIT;\n")
	time.Sleep(150 * time.Millisecond)
	b := db.Bind(claimQ, "k", future, "B") // blocks until A commits
	outA := a.Wait(t)
	if !strings.Contains(outA, "INSERT 0 1") || b != "INSERT 0 0" {
		t.Fatalf("A=%q B=%q", outA, b)
	}
}
