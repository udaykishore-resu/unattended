package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestBadPolicyRefusedBeforeAnyCall(t *testing.T) {
	for _, c := range []struct {
		tries      int
		base, ceil time.Duration
	}{
		{0, time.Millisecond, time.Second},
		{-3, time.Millisecond, time.Second},
		{3, 0, time.Second},
		{3, -time.Millisecond, time.Second},
		{3, time.Millisecond, 0},
		{3, time.Second, time.Millisecond}, // ceil below base
	} {
		called := false
		err := Retry(context.Background(), c.tries, c.base, c.ceil,
			func(context.Context) error { called = true; return nil })
		if !errors.Is(err, ErrPolicy) || called {
			t.Fatalf("%+v: err=%v called=%v", c, err, called)
		}
	}
}

// With many tries, base<<i would overflow
// time.Duration (and rand.N would panic on a
// negative value). The cap is applied first.
func TestManyTriesNoOverflow(t *testing.T) {
	ceil := 2 * time.Millisecond
	n := 0
	err := Retry(context.Background(), 70, time.Millisecond, ceil,
		func(context.Context) error { n++; return errTimeout })
	if n != 70 || !errors.Is(err, errTimeout) {
		t.Fatalf("tries=%d err=%v", n, err)
	}
}

// No sleep after the last attempt: giving up is
// immediate.
func TestNoSleepAfterLastTry(t *testing.T) {
	start := time.Now()
	Retry(context.Background(), 1, time.Hour, time.Hour,
		func(context.Context) error { return errTimeout })
	if time.Since(start) > time.Second {
		t.Fatal("slept after the last try")
	}
}

// Cancel while Retry sleeps between attempts.
func TestCancelDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	time.AfterFunc(20*time.Millisecond, cancel)
	start := time.Now()
	err := Retry(ctx, 5, time.Hour, time.Hour,
		func(context.Context) error { n++; return errTimeout })
	if !errors.Is(err, context.Canceled) || n != 1 || time.Since(start) > 5*time.Second {
		t.Fatalf("err=%v tries=%d", err, n)
	}
}

// A Permanent wrapped in another error still stops
// retries at once.
func TestWrappedPermanent(t *testing.T) {
	n := 0
	err := Retry(context.Background(), 5, time.Millisecond, time.Millisecond,
		func(context.Context) error {
			n++
			return fmt.Errorf("invoice 40117: %w",
				Permanent{errors.New("400: customer not found")})
		})
	var p Permanent
	if n != 1 || !errors.As(err, &p) {
		t.Fatalf("tries=%d err=%v", n, err)
	}
}
