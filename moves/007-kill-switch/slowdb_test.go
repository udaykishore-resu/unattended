package killswitch

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

// slowConnector is a minimal context-aware driver:
// each query answers enabled=true after delay, unless
// the context ends first, as pgx and lib/pq do.
type slowConnector struct{ delay time.Duration }

func (c slowConnector) Connect(context.Context) (driver.Conn, error) {
	return slowConn(c), nil
}
func (slowConnector) Driver() driver.Driver { return nil }

type slowConn struct{ delay time.Duration }

func (slowConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("slowConn: no prepare")
}
func (slowConn) Close() error { return nil }
func (slowConn) Begin() (driver.Tx, error) {
	return nil, errors.New("slowConn: no tx")
}

func (c slowConn) QueryContext(ctx context.Context, _ string,
	_ []driver.NamedValue) (driver.Rows, error) {
	select {
	case <-time.After(c.delay):
		return &oneRow{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type oneRow struct{ done bool }

func (*oneRow) Columns() []string { return []string{"enabled"} }
func (*oneRow) Close() error      { return nil }
func (r *oneRow) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0], r.done = true, true
	return nil
}

// A database slower than the 500 ms budget reads as
// "off", and the item goes to people.
func TestSlowDatabaseTimesOutToOff(t *testing.T) {
	fast := Gate{DB: sql.OpenDB(slowConnector{10 * time.Millisecond}),
		Name: "ops_tagger"}
	if !fast.Enabled(context.Background()) {
		t.Fatal("a 10 ms answer of true should read as on")
	}
	slow := Gate{DB: sql.OpenDB(slowConnector{2 * time.Second}),
		Name: "ops_tagger"}
	start := time.Now()
	if slow.Enabled(context.Background()) {
		t.Fatal("a 2 s database must read as off")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("Enabled took %v; the 500 ms timeout did not fire", d)
	}
	acted := 0
	act = func(context.Context, Item) error { acted++; return nil }
	q := &humanQ{}
	if err := Handle(context.Background(), slow, q, Item{ID: "x"}); err != nil {
		t.Fatal(err)
	}
	if acted != 0 || len(q.items) != 1 {
		t.Fatalf("acted=%d human=%d, want 0 and 1", acted, len(q.items))
	}
}

// The gate is read per action, not at start:
// flipping the flag between two items changes what
// the second one does, with no restart.
func TestCheckedPerAction(t *testing.T) {
	on := true
	g := Gate{DB: sqlfake.Open(func(string, []any) (
		[]string, [][]any, error) {
		return []string{"enabled"}, [][]any{{on}}, nil
	}), Name: "ops_tagger"}
	acted := 0
	act = func(context.Context, Item) error { acted++; return nil }
	q := &humanQ{}
	ctx := context.Background()
	if err := Handle(ctx, g, q, Item{ID: "1"}); err != nil {
		t.Fatal(err)
	}
	on = false // on-call flips the row
	if err := Handle(ctx, g, q, Item{ID: "2"}); err != nil {
		t.Fatal(err)
	}
	if acted != 1 || len(q.items) != 1 || q.items[0].ID != "2" {
		t.Fatalf("acted=%d human=%v", acted, q.items)
	}
}

// A NULL in enabled cannot scan into a bool, so it
// reads as off too.
func TestNullFlagMeansOff(t *testing.T) {
	g := Gate{DB: sqlfake.Open(func(string, []any) (
		[]string, [][]any, error) {
		return []string{"enabled"}, [][]any{{nil}}, nil
	}), Name: "ops_tagger"}
	if g.Enabled(context.Background()) {
		t.Fatal("NULL enabled must fail closed")
	}
}
