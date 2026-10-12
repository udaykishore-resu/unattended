package durable

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

type journal struct {
	mu   sync.Mutex
	rows map[string][]byte
}

func (j *journal) h(q string, args []any) ([]string, [][]any, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	k := args[0].(string) + "/" + args[1].(string)
	switch q {
	case getStep:
		if b, ok := j.rows[k]; ok {
			return []string{"result"}, [][]any{{b}}, nil
		}
		return []string{"result"}, nil, nil // sql.ErrNoRows
	case putStep:
		j.rows[k] = args[2].([]byte)
		return nil, [][]any{{1}}, nil
	}
	return nil, nil, errors.New("unknown query")
}

type quote struct {
	Carrier string `json:"carrier"`
	Cents   int64  `json:"cents"`
}

// The quote flow: ask three carriers, then book.
func quoteFlow(ctx context.Context, r *Run, emails *int, crashAfter int) error {
	for i, c := range []string{"RDX", "Blue Mesa", "Okafor"} {
		if i == crashAfter {
			return errors.New("node drained")
		}
		_, err := Step(ctx, r, fmt.Sprintf("ask-carrier-%d", i),
			func(context.Context) (quote, error) {
				*emails++ // the side effect
				return quote{c, int64(200000 + i*1000)}, nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// A node drain at hour four must not resend the
// carrier emails.
func TestReplayReturnsJournaledResults(t *testing.T) {
	j := &journal{rows: map[string][]byte{}}
	r := &Run{ID: "quote-77", DB: sqlfake.Open(j.h)}
	emails := 0
	ctx := context.Background()

	if err := quoteFlow(ctx, r, &emails, 2); err == nil {
		t.Fatal("want the drain")
	}
	if err := quoteFlow(ctx, r, &emails, -1); err != nil {
		t.Fatal(err)
	}
	if emails != 3 {
		t.Fatalf("emails sent = %d, want 3 (one per carrier)", emails)
	}
	// The journaled value comes back typed.
	q, err := Step(ctx, r, "ask-carrier-1", func(context.Context) (quote, error) {
		t.Fatal("must replay from the journal")
		return quote{}, nil
	})
	if err != nil || q.Carrier != "Blue Mesa" || q.Cents != 201000 {
		t.Fatalf("q=%+v err=%v", q, err)
	}
}

func TestStepErrorIsNotJournaled(t *testing.T) {
	j := &journal{rows: map[string][]byte{}}
	r := &Run{ID: "r", DB: sqlfake.Open(j.h)}
	_, err := Step(context.Background(), r, "s", func(context.Context) (int, error) {
		return 0, errors.New("carrier API 503")
	})
	if err == nil || len(j.rows) != 0 {
		t.Fatalf("err=%v rows=%d", err, len(j.rows))
	}
}
