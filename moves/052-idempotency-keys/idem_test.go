package idem

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/internal/sqlfake"
)

// steps emulates the ai_steps table and its four queries.
type steps struct {
	mu   sync.Mutex
	rows map[string]*row
}

type row struct {
	lease time.Time
	out   []byte
}

func (s *steps) h(q string, args []any) ([]string, [][]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := args[0].(string)
	r := s.rows[key]
	switch q {
	case claimQ:
		lease := args[1].(time.Time)
		if r == nil {
			s.rows[key] = &row{lease: lease}
			return nil, [][]any{{1}}, nil
		}
		if r.out == nil && r.lease.Before(time.Now()) {
			r.lease = lease // take over an expired lease
			return nil, [][]any{{1}}, nil
		}
		return nil, nil, nil
	case savedQ:
		return []string{"output"}, [][]any{{r.out}}, nil
	case releaseQ:
		r.lease = time.Now().Add(-time.Second)
		return nil, [][]any{{1}}, nil
	case finishQ:
		r.out = args[1].([]byte)
		return nil, [][]any{{1}}, nil
	}
	return nil, nil, errors.New("unknown query")
}

func newDB() *steps { return &steps{rows: map[string]*row{}} }

// The key is built from carrier and invoice number.
// A prompt fix (v7 -> v8) does not change it, so
// replaying failed runs never sends the email again.
func TestKeyStableAcrossPromptVersions(t *testing.T) {
	st := newDB()
	db := sqlfake.Open(st.h)
	sends := 0
	step := func(promptVer string) func(context.Context) ([]byte, error) {
		return func(context.Context) ([]byte, error) {
			sends++ // the side effect: an email to the carrier
			return []byte("sent with " + promptVer), nil
		}
	}
	key := Key("dispute-email", "carrier:RDX", "invoice:40117")
	ctx := context.Background()

	out1, err := Once(ctx, db, key, step("v7"))
	if err != nil {
		t.Fatal(err)
	}
	// Same business identity after a prompt fix: same key.
	if Key("dispute-email", "carrier:RDX", "invoice:40117") != key {
		t.Fatal("key changed")
	}
	out2, err := Once(ctx, db, key, step("v8"))
	if err != nil {
		t.Fatal(err)
	}
	if sends != 1 || string(out2) != string(out1) || string(out2) != "sent with v7" {
		t.Fatalf("sends=%d out1=%s out2=%s", sends, out1, out2)
	}
}

func TestLengthPrefix(t *testing.T) {
	if Key("s", "ab", "c") == Key("s", "a", "bc") {
		t.Fatal(`"ab"+"c" and "a"+"bc" must differ`)
	}
}

// A failed step releases its lease, so a retry runs.
func TestFailureReleases(t *testing.T) {
	db := sqlfake.Open(newDB().h)
	ctx := context.Background()
	key := Key("summarize", "event:e-9")
	if _, err := Once(ctx, db, key, func(context.Context) ([]byte, error) {
		return nil, errors.New("timeout")
	}); err == nil {
		t.Fatal("want the step error")
	}
	out, err := Once(ctx, db, key, func(context.Context) ([]byte, error) {
		return []byte("ok"), nil
	})
	if err != nil || string(out) != "ok" {
		t.Fatalf("retry: %s %v", out, err)
	}
}

// A second worker that finds a live lease gets ErrInFlight.
func TestInFlight(t *testing.T) {
	st := newDB()
	db := sqlfake.Open(st.h)
	key := Key("x", "y")
	st.rows[key] = &row{lease: time.Now().Add(time.Hour)}
	_, err := Once(context.Background(), db, key, func(context.Context) ([]byte, error) {
		t.Fatal("must not run")
		return nil, nil
	})
	if !errors.Is(err, ErrInFlight) {
		t.Fatalf("got %v", err)
	}
}
