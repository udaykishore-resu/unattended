package dryrun

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type memStore struct {
	loads  map[string]Load
	writes int
}

func (m *memStore) GetLoad(_ context.Context, id string) (Load, error) {
	l, ok := m.loads[id]
	if !ok {
		return Load{}, errors.New("not found")
	}
	return l, nil
}

func (m *memStore) SetStatus(_ context.Context, id, s string) error {
	m.writes++
	l := m.loads[id]
	l.Status = s
	m.loads[id] = l
	return nil
}

func store() *memStore {
	return &memStore{loads: map[string]Load{
		"HF-1": {"HF-1", "in_transit"}}}
}

func TestPlanShowsOldAndNew(t *testing.T) {
	p, err := planSetStatus(context.Background(), store(),
		SetStatusIn{"HF-1", "delivered"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Changes["status"] != [2]any{"in_transit", "delivered"} || len(p.Notes) != 0 {
		t.Fatalf("plan = %+v", p)
	}
	same, _ := planSetStatus(context.Background(), store(),
		SetStatusIn{"HF-1", "in_transit"})
	if len(same.Notes) != 1 || same.Notes[0] != "no-op: same status" {
		t.Fatalf("no-op not flagged: %+v", same)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	s := store()
	out, err := SetStatus(context.Background(), s,
		SetStatusIn{"HF-1", "delivered"}, true)
	if err != nil || !strings.HasPrefix(out, "DRY RUN, nothing changed.") {
		t.Fatalf("%q %v", out, err)
	}
	if s.writes != 0 || s.loads["HF-1"].Status != "in_transit" {
		t.Fatal("dry run wrote")
	}
}

func TestLiveRunSharesThePlanChecks(t *testing.T) {
	s := store()
	if _, err := SetStatus(context.Background(), s,
		SetStatusIn{"HF-404", "delivered"}, false); err == nil {
		t.Fatal("a plan that fails must fail the live run too")
	}
	if _, err := SetStatus(context.Background(), s,
		SetStatusIn{"HF-1", "delivered"}, false); err != nil || s.writes != 1 {
		t.Fatalf("live: %v writes=%d", err, s.writes)
	}
}
