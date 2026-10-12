package fanout

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/udaykishore-resu/unattended/llm"
)

func TestFanOut(t *testing.T) {
	long := strings.Repeat("r", 500)
	f := &llm.FakeModel{}
	var loads []Load
	for i := range 20 {
		loads = append(loads, Load{ID: fmt.Sprintf("HF-%d", i)})
		// The model gets the load ID wrong and ignores maxLength.
		f.Responses = append(f.Responses, llm.Response{Text: fmt.Sprintf(
			`{"load_id":"HF-WRONG","status":"at_risk","reason":%q,"quote":"x"}`, long)})
	}
	got := FanOut(context.Background(), f, loads)
	for i, g := range got {
		if g.LoadID != loads[i].ID {
			t.Errorf("finding %d has load %s; IDs come from our data", i, g.LoadID)
		}
		if len(g.Reason) != 200 {
			t.Errorf("reason not clipped: %d", len(g.Reason))
		}
	}
	if len(f.Calls) != 20 {
		t.Fatalf("calls = %d", len(f.Calls))
	}
}

func TestFailedCheckGetsFixedMessage(t *testing.T) {
	f := &llm.FakeModel{Err: fmt.Errorf("dial db-prod-3: refused")}
	got := FanOut(context.Background(), f, []Load{{ID: "HF-1"}})
	if got[0] != (Finding{LoadID: "HF-1", Status: "error", Reason: "check failed"}) {
		t.Fatalf("got %+v", got[0])
	}
}

func TestClipRunes(t *testing.T) {
	if got := clip("héllo", 2); got != "hé" {
		t.Fatalf("got %q", got)
	}
}

// Partial failures, odd answers and a cancelled run:
// every load gets exactly one finding, and nothing
// that isn't a real answer is read as "ok".
func TestMissingResultIsNeverClean(t *testing.T) {
	f := &llm.FakeModel{Responses: []llm.Response{
		{Text: `{"status":"ok","reason":"on time"}`},
		{Text: `{}`},                // no status
		{Text: `{"status":"fine"}`}, // not in the enum
		{Text: `not json`},          // decode failure
		{Text: `{"status":"at_risk","reason":"truck down"}`},
	}}
	// The fake hands out responses in arrival order, not
	// per load, so check the multiset of statuses.
	var loads []Load
	for i := range 5 {
		loads = append(loads, Load{ID: fmt.Sprintf("HF-%d", i)})
	}
	got := FanOut(context.Background(), f, loads)
	if len(got) != 5 {
		t.Fatalf("got %d findings", len(got))
	}
	count := map[string]int{}
	ids := map[string]bool{}
	for _, g := range got {
		count[g.Status]++
		ids[g.LoadID] = true
	}
	if count["ok"] != 1 || count["at_risk"] != 1 || count["error"] != 3 || len(ids) != 5 {
		t.Fatalf("statuses %v, ids %v", count, ids)
	}
}

// A cancelled run returns at once, and every check
// that didn't answer is an error, never ok.
func TestCancelledRunIsAllErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := ctxModel{}
	var loads []Load
	for i := range 30 {
		loads = append(loads, Load{ID: fmt.Sprintf("HF-%d", i)})
	}
	for _, g := range FanOut(ctx, m, loads) {
		if g.Status != "error" {
			t.Fatalf("cancelled check reads as %q", g.Status)
		}
	}
}

// ctxModel answers ok unless ctx is done, as a real
// client would.
type ctxModel struct{}

func (ctxModel) Complete(ctx context.Context, _ llm.Request) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, err
	}
	return llm.Response{Text: `{"status":"ok"}`, StopReason: llm.StopEnd}, nil
}

// The model can't create duplicates or swap loads:
// every finding's ID is the load it was sent for.
func TestNoDuplicateFindings(t *testing.T) {
	f := &llm.FakeModel{}
	var loads []Load
	for i := range 10 {
		loads = append(loads, Load{ID: fmt.Sprintf("HF-%d", i)})
		f.Responses = append(f.Responses, llm.Response{Text: `{"load_id":"HF-0","status":"ok"}`})
	}
	seen := map[string]bool{}
	for _, g := range FanOut(context.Background(), f, loads) {
		if seen[g.LoadID] {
			t.Fatalf("duplicate finding for %s", g.LoadID)
		}
		seen[g.LoadID] = true
	}
}

// The judge must list back every unknown and rank
// only loads it was sent.
func TestCheckVerdict(t *testing.T) {
	fs := []Finding{{LoadID: "HF-1", Status: "ok"}, {LoadID: "HF-2", Status: "error"},
		{LoadID: "HF-3", Status: "at_risk"}, {LoadID: "HF-4", Status: "error"}}
	good := Verdict{CallFirst: []string{"HF-3"}, Unknown: []string{"HF-4", "HF-2"}}
	if err := CheckVerdict(fs, good); err != nil {
		t.Fatal(err)
	}
	dropped := Verdict{CallFirst: []string{"HF-3"}, Unknown: []string{"HF-2"}}
	if err := CheckVerdict(fs, dropped); err == nil {
		t.Fatal("judge dropped an unknown")
	}
	invented := Verdict{CallFirst: []string{"HF-9"}, Unknown: []string{"HF-2", "HF-4"}}
	if err := CheckVerdict(fs, invented); err == nil {
		t.Fatal("judge ranked a load it never saw")
	}
}
