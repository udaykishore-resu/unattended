package flaky

import (
	"errors"
	"testing"
)

// The re-run passed, but the failed run's log and
// runner survive, and the model gets one per commit.
func TestRerunKeepsFailureEvidence(t *testing.T) {
	const name = "TestAssignLoad_Concurrent"
	runs := []Run{
		{name, "a1", false, "ci-runner-7", "panic: timeout after 5s"},
		{name, "a1", true, "ci-runner-2", ""}, // re-run passed
		{name, "a1", false, "ci-runner-9", "panic: timeout after 5s (again)"},
		{name, "b2", false, "ci-runner-3", "fixture: carrier row missing"},
		{name, "b2", true, "ci-runner-3", ""},
		{name, "c3", false, "ci-runner-1", "DATA RACE"},
		{name, "c3", true, "ci-runner-4", ""},
		{name, "d4", false, "ci-runner-5", "fourth"},
	}
	if Flaky(runs)[name] != 3 {
		t.Fatalf("flips = %d", Flaky(runs)[name])
	}
	got := Failures(runs, name, 3)
	if len(got) != 3 || got[0].SHA != "a1" || got[1].SHA != "b2" || got[2].SHA != "c3" {
		t.Fatalf("got %+v", got)
	}
	if got[0].Log != "panic: timeout after 5s" || got[0].Runner != "ci-runner-7" {
		t.Fatalf("evidence lost: %+v", got[0])
	}
}

func TestDataRaceIsNeverQuarantined(t *testing.T) {
	if err := Quarantine("data race"); !errors.Is(err, ErrRace) {
		t.Fatal(err)
	}
	if err := Quarantine("timing"); err != nil {
		t.Fatal(err)
	}
}
