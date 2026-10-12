//go:build integration

package review

import (
	"strings"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/psqltest"
)

// Runs claimQ and decideQ verbatim in Postgres:
//
//	UNATTENDED_TEST_PSQL_URL=postgres://postgres@localhost:5433/unattended_test \
//	  go test -tags integration ./moves/059-review-queue/
const fixture = `INSERT INTO reviews (kind, draft, evidence, due_by) VALUES
 ('credit-memo', 'Lakeside $4,120', 'inv 40117', now() + interval '1 hour'),
 ('credit-memo', 'Okafor $310',     'inv 40200', now() + interval '2 hours');`

func id(out string) string { // first column of the first row
	return strings.SplitN(strings.SplitN(out, "\n", 2)[0], "|", 2)[0]
}

func TestClaimAndDecideInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	db.Run(fixture)
	check := func(got, want string) {
		t.Helper()
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	// Due first is handed out first; a live claim is
	// not handed out again.
	check(id(db.Bind(claimQ, "marcus")), "1")
	check(id(db.Bind(claimQ, "priya")), "2")
	check(db.Bind(claimQ, "theo"), "UPDATE 0")

	// Marcus's claim lapses and Priya's backup takes it.
	db.Run("UPDATE reviews SET claimed_at = now() - interval '16 minutes' WHERE id = 1;")
	check(id(db.Bind(claimQ, "priya")), "1")
	check(db.Bind(decideQ, "1", "marcus", "approved", "x"), "UPDATE 0") // lapsed
	check(db.Bind(decideQ, "1", "priya", "approved", "edited"), "UPDATE 1")
	check(db.Bind(decideQ, "1", "priya", "rejected", ""), "UPDATE 0") // decided
	check(db.Run("SELECT state, final, decided_by FROM reviews WHERE id = 1;"),
		"approved|edited|priya")
	// A decided item is never handed out again.
	db.Run("UPDATE reviews SET claimed_at = now() - interval '1 day';")
	check(id(db.Bind(claimQ, "theo")), "2")
}

// Two submissions race on one item (a double click,
// two tabs): the row lock serialises them and the
// second finds the item no longer open.
func TestConcurrentDecisionsInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	db.Run(fixture)
	db.Bind(claimQ, "marcus")
	a := db.Start("BEGIN;\n" + decideQ + " " +
		psqltest.BindArgs("1", "marcus", "approved", "first") +
		"\nSELECT pg_sleep(0.5);\nCOMMIT;\n")
	time.Sleep(150 * time.Millisecond)
	b := db.Bind(decideQ, "1", "marcus", "rejected", "")
	outA := a.Wait(t)
	if !strings.Contains(outA, "UPDATE 1") || b != "UPDATE 0" {
		t.Fatalf("A=%q B=%q", outA, b)
	}
	if got := db.Run("SELECT state, final FROM reviews WHERE id = 1;"); got != "approved|first" {
		t.Fatalf("row = %q", got)
	}
}

// Two reviewers claim at the same moment: SKIP LOCKED
// gives them different items.
func TestConcurrentClaimsInPostgres(t *testing.T) {
	db := psqltest.New(t, Schema)
	db.Run(fixture)
	a := db.Start("BEGIN;\n" + claimQ + " " + psqltest.BindArgs("marcus") +
		"\nSELECT pg_sleep(0.5);\nCOMMIT;\n")
	time.Sleep(150 * time.Millisecond)
	b := id(db.Bind(claimQ, "priya"))
	outA := id(strings.TrimPrefix(a.Wait(t), "BEGIN\n"))
	if outA != "1" || b != "2" {
		t.Fatalf("marcus=%q priya=%q", outA, b)
	}
}
