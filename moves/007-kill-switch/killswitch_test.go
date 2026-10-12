package killswitch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

type humanQ struct{ items []Item }

func (q *humanQ) ToHuman(_ context.Context, it Item, _ string) error {
	q.items = append(q.items, it)
	return nil
}

// flagDB answers the flag query with on, or err.
func flagDB(on bool, err error, delay time.Duration) sqlfake.Handler {
	return func(q string, args []any) ([]string, [][]any, error) {
		time.Sleep(delay)
		if err != nil {
			return nil, nil, err
		}
		return []string{"enabled"}, [][]any{{on}}, nil
	}
}

func TestOffRoutesToHuman(t *testing.T) {
	cases := map[string]sqlfake.Handler{
		"flag off": flagDB(false, nil, 0),
		"db error": flagDB(true, errors.New("conn refused"), 0),
		"no row": func(string, []any) ([]string, [][]any, error) {
			return []string{"enabled"}, nil, nil
		},
	}
	for name, h := range cases {
		acted := 0
		act = func(context.Context, Item) error { acted++; return nil }
		q := &humanQ{}
		g := Gate{DB: sqlfake.Open(h), Name: "ops_tagger"}
		// Ten items with the flag off: all ten reach people.
		for range 10 {
			if err := Handle(context.Background(), g, q, Item{ID: "x"}); err != nil {
				t.Fatal(err)
			}
		}
		if acted != 0 || len(q.items) != 10 {
			t.Errorf("%s: acted=%d human=%d, want 0 and 10", name, acted, len(q.items))
		}
	}
}

func TestOnActs(t *testing.T) {
	acted := 0
	act = func(context.Context, Item) error { acted++; return nil }
	q := &humanQ{}
	g := Gate{DB: sqlfake.Open(flagDB(true, nil, 0)), Name: "ops_tagger"}
	if err := Handle(context.Background(), g, q, Item{ID: "x"}); err != nil {
		t.Fatal(err)
	}
	if acted != 1 || len(q.items) != 0 {
		t.Fatalf("acted=%d human=%d", acted, len(q.items))
	}
}

func TestSlowDatabaseMeansOff(t *testing.T) {
	g := Gate{DB: sqlfake.Open(flagDB(true, nil, 0)), Name: "ops_tagger"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // a dead context stands in for a database past its timeout
	if g.Enabled(ctx) {
		t.Fatal("a timed-out read must fail closed")
	}
}
