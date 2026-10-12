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
