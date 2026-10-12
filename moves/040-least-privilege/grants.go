// Package grants makes an agent refuse to start if
// its database role has more than it should.
//
// Move 40: least-privilege credentials per agent.
package grants

import (
	"context"
	"database/sql"
	"fmt"
)

var forbidden = [][2]string{{"shipments", "UPDATE"},
	{"shipments", "DELETE"}, {"invoices", "SELECT"}}

func CheckGrants(ctx context.Context,
	db *sql.DB) error {
	const q = `SELECT has_table_privilege($1, $2)`
	for _, f := range forbidden {
		var has bool
		err := db.QueryRowContext(ctx, q,
			f[0], f[1]).Scan(&has)
		if err != nil {
			return err
		}
		if has {
			return fmt.Errorf("has %s on %s", f[1], f[0])
		}
	}
	return nil
}
