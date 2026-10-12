package enrich

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type src struct {
	rbErr    error
	deploys  []Deploy
	gotSince time.Time
}

func (s *src) Runbook(context.Context, string) (string, error) {
	if s.rbErr != nil {
		return "", s.rbErr
	}
	return "Step 2: check DB pool", nil
}

func (s *src) Deploys(_ context.Context, _ string, since time.Time) ([]Deploy, error) {
	s.gotSince = since
	return s.deploys, nil
}

func (s *src) Note(context.Context, Enriched) (string, error) {
	return "step 2 applies; deploy changed pool size, possibly related", nil
}

var fired = time.Date(2026, 3, 9, 2, 14, 0, 0, time.UTC)

func TestEnrich(t *testing.T) {
	d := Deploy{"ratequote", "pool-size", fired.Add(-11 * time.Minute)}
	s := &src{deploys: []Deploy{d}}
	e := Enrich(context.Background(), s, Alert{"QuoteLatencyHigh", "ratequote", fired})
	if e.Runbook == "" || !reflect.DeepEqual(e.Deploys, []Deploy{d}) || len(e.Missing) != 0 {
		t.Fatalf("got %+v", e)
	}
	if !s.gotSince.Equal(fired.Add(-2 * time.Hour)) {
		t.Fatalf("since = %v", s.gotSince)
	}
}

// "Runbook: lookup failed" is useful. A page that
// never arrived is not.
func TestFailedSourceIsNamedAndPageGoesOut(t *testing.T) {
	s := &src{rbErr: errors.New("wiki 502")}
	e := Enrich(context.Background(), s, Alert{Name: "QuoteLatencyHigh", FiredAt: fired})
	if !reflect.DeepEqual(e.Missing, []string{"runbook"}) || e.Alert.Name != "QuoteLatencyHigh" {
		t.Fatalf("got %+v", e)
	}
}
