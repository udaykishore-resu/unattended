package runcaps

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestStepCap(t *testing.T) {
	r := &Run{Limits: Limits{Steps: 3, Tokens: 1 << 30}}
	var err error
	n := 0
	for err == nil {
		err = r.Charge(llm.Usage{InputTokens: 10})
		n++
	}
	if n != 3 || !errors.Is(err, ErrBudget) || !strings.Contains(err.Error(), "3 steps") {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

// With prompt caching on, InputTokens shrinks to the
// uncached tail while the context keeps growing. The
// cap still trips, because all four fields count.
func TestTokenCapCountsCachedInput(t *testing.T) {
	r := &Run{Limits: Limits{Steps: 100, Tokens: 50_000}}
	u := llm.Usage{InputTokens: 200, CacheReadTokens: 30_000, OutputTokens: 300}
	if err := r.Charge(u); err != nil {
		t.Fatal(err)
	}
	if err := r.Charge(u); !errors.Is(err, ErrBudget) ||
		!strings.Contains(err.Error(), "tokens") {
		t.Fatalf("got %v", err)
	}
}

func TestWallClockSaysWhy(t *testing.T) {
	ctx, cancel := WithWall(context.Background(), Limits{Wall: 10 * time.Millisecond})
	defer cancel()
	<-ctx.Done()
	err := context.Cause(ctx)
	if !errors.Is(err, ErrBudget) || !strings.Contains(err.Error(), "wall clock") {
		t.Fatalf("cause = %v", err)
	}
}
