package grants

import (
	"context"
	"errors"
	"testing"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

// role answers has_table_privilege from a grant set,
// at table level, as Postgres does with two args.
func role(grants map[[2]string]bool) *sqlfakeDB {
	return &sqlfakeDB{grants}
}

type sqlfakeDB struct{ grants map[[2]string]bool }

func (r *sqlfakeDB) handler(q string, args []any) ([]string, [][]any, error) {
	k := [2]string{args[0].(string), args[1].(string)}
	return []string{"has"}, [][]any{{r.grants[k]}}, nil
}

func TestNarrowRoleBoots(t *testing.T) {
	// agent_support: SELECT on shipments; a column grant
	// like UPDATE (pickup_start) is not table-level.
	r := role(map[[2]string]bool{{"shipments", "SELECT"}: true})
	db := sqlfake.Open(r.handler)
	if err := CheckGrants(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}

func TestWideRoleRefusesToStart(t *testing.T) {
	// It shared the platform team's account and could read invoices.
	r := role(map[[2]string]bool{{"invoices", "SELECT"}: true})
	err := CheckGrants(context.Background(), sqlfake.Open(r.handler))
	if err == nil || err.Error() != "has SELECT on invoices" {
		t.Fatalf("got %v", err)
	}
}

func TestQueryErrorFailsBoot(t *testing.T) {
	db := sqlfake.Open(func(string, []any) ([]string, [][]any, error) {
		return nil, nil, errors.New("permission denied")
	})
	if err := CheckGrants(context.Background(), db); err == nil {
		t.Fatal("a failed check must not pass")
	}
}
