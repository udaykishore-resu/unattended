package review

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

// table emulates reviews and the three queries (the
// SQL itself runs in Postgres in
// review_integration_test.go).
type table struct {
	mu   sync.Mutex
	rows []*rec
}

type rec struct {
	id                 int64
	kind, draft, final string
	state              string
	seed               bool
	due                time.Time
	by                 string
	at                 time.Time
}

func (tb *table) h(q string, args []any) ([]string, [][]any, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	switch q {
	case claimQ:
		open := []*rec{}
		for _, r := range tb.rows {
			if r.state == "open" && (r.by == "" || r.at.Before(time.Now().Add(-15*time.Minute))) {
				open = append(open, r)
			}
		}
		if len(open) == 0 {
			return []string{"id"}, nil, nil
		}
		sort.Slice(open, func(i, j int) bool { return open[i].due.Before(open[j].due) })
		r := open[0]
		r.by, r.at = args[0].(string), time.Now()
		return []string{"id", "kind", "draft", "evidence", "due_by"},
			[][]any{{r.id, r.kind, r.draft, "ev", r.due}}, nil
	case decideQ:
		for _, r := range tb.rows {
			if r.id == args[0].(int64) && r.state == "open" && r.by == args[1].(string) {
				r.state = args[2].(string)
				if f, ok := args[3].(string); ok {
					r.final = f
				}
				return nil, [][]any{{1}}, nil
			}
		}
		return nil, nil, nil
	case approvedQ:
		for _, r := range tb.rows {
			if r.id == args[0].(int64) && r.state == "approved" {
				return []string{"final", "seed"}, [][]any{{r.final, r.seed}}, nil
			}
		}
		return []string{"final", "seed"}, nil, nil
	}
	return nil, nil, errors.New("unknown query")
}

func newTable(rs ...*rec) *table {
	for i, r := range rs {
		r.id, r.state = int64(i+1), "open"
	}
	return &table{rows: rs}
}

// Marcus walks away mid-review; after 15 minutes the
// backup reviewer claims the same memo and approves
// it. Marcus's late "approve" changes nothing, and
// only one decision is executed.
func TestLapsedClaimCannotCommit(t *testing.T) {
	now := time.Now()
	tb := newTable(&rec{kind: "credit-memo", draft: "Lakeside Foods $4,120", due: now})
	db := sqlfake.Open(tb.h)
	ctx := context.Background()

	m, err := Claim(ctx, db, "marcus")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Claim(ctx, db, "priya"); !errors.Is(err, ErrEmpty) {
		t.Fatalf("live claim handed out twice: %v", err)
	}
	tb.rows[0].at = now.Add(-16 * time.Minute) // the claim lapses
	p, err := Claim(ctx, db, "priya")
	if err != nil || p.ID != m.ID {
		t.Fatalf("reclaim: %+v %v", p, err)
	}
	if err := Decide(ctx, db, p.ID, "priya", "approved", "Lakeside Foods $4,120 (edited)"); err != nil {
		t.Fatal(err)
	}
	if err := Decide(ctx, db, m.ID, "marcus", "approved", "Lakeside Foods $4,120"); !errors.Is(err, ErrClaimLost) {
		t.Fatalf("late decision: %v", err)
	}
	if err := Decide(ctx, db, p.ID, "priya", "rejected", ""); !errors.Is(err, ErrClaimLost) {
		t.Fatalf("second decision: %v", err)
	}
	var sent []string
	err = Execute(ctx, db, p.ID, func(_ context.Context, s string) error {
		sent = append(sent, s)
		return nil
	})
	if err != nil || len(sent) != 1 || sent[0] != "Lakeside Foods $4,120 (edited)" {
		t.Fatalf("sent=%v err=%v", sent, err)
	}
}

// A lapsed claim nobody took can still be decided by
// its holder: one decision either way.
func TestLapsedButUntakenStillCommits(t *testing.T) {
	tb := newTable(&rec{kind: "credit-memo", draft: "d", due: time.Now()})
	db := sqlfake.Open(tb.h)
	ctx := context.Background()
	it, _ := Claim(ctx, db, "marcus")
	tb.rows[0].at = time.Now().Add(-time.Hour)
	if err := Decide(ctx, db, it.ID, "marcus", "rejected", ""); err != nil {
		t.Fatal(err)
	}
}

// Seeds are refused by the executor even when
// approved; rejected items are never sent.
func TestExecutorRefusesSeedsAndUnapproved(t *testing.T) {
	tb := newTable(
		&rec{kind: "credit-memo", draft: "planted bad", due: time.Now(), seed: true},
		&rec{kind: "credit-memo", draft: "real", due: time.Now().Add(time.Minute)},
	)
	db := sqlfake.Open(tb.h)
	ctx := context.Background()
	send := func(context.Context, string) error {
		t.Fatal("must not send")
		return nil
	}
	s, _ := Claim(ctx, db, "marcus")
	Decide(ctx, db, s.ID, "marcus", "approved", "planted bad") // rubber stamp
	if err := Execute(ctx, db, s.ID, send); !errors.Is(err, ErrSeed) {
		t.Fatalf("seed: %v", err)
	}
	r, _ := Claim(ctx, db, "marcus")
	Decide(ctx, db, r.ID, "marcus", "rejected", "")
	if err := Execute(ctx, db, r.ID, send); !errors.Is(err, ErrNotApproved) {
		t.Fatalf("rejected: %v", err)
	}
}
