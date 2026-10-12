package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errTimeout = errors.New("timeout")

func TestShortOutageHeals(t *testing.T) {
	n := 0
	err := Retry(context.Background(), 5, time.Millisecond, 10*time.Millisecond,
		func(context.Context) error {
			n++
			if n < 3 {
				return errTimeout
			}
			return nil
		})
	if err != nil || n != 3 {
		t.Fatalf("err=%v tries=%d", err, n)
	}
}

func TestPermanentStopsAtOnce(t *testing.T) {
	n := 0
	err := Retry(context.Background(), 5, time.Millisecond, 10*time.Millisecond,
		func(context.Context) error {
			n++
			return Permanent{errors.New("400: customer not found")}
		})
	var p Permanent
	if n != 1 || !errors.As(err, &p) {
		t.Fatalf("tries=%d err=%v", n, err)
	}
}

func TestGivesUpWithLastError(t *testing.T) {
	n := 0
	err := Retry(context.Background(), 3, time.Millisecond, 2*time.Millisecond,
		func(context.Context) error { n++; return errTimeout })
	if n != 3 || !errors.Is(err, errTimeout) {
		t.Fatalf("tries=%d err=%v", n, err)
	}
}

func TestContextCancelStopsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Retry(ctx, 5, time.Hour, time.Hour,
		func(context.Context) error { return errTimeout })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
