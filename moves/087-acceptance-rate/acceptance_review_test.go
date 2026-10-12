package acceptance

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// A small human-labelled sample: Ines's team read each
// pair and picked the fate by hand. Classify is a word-
// overlap heuristic; this checks how often it agrees.
var labelled = []struct {
	draft, sent string
	used        bool
	human       Fate
}{
	{draft, draft, true, Accepted},
	{draft, "  " + draft + "\n", true, Accepted},
	{draft, "", false, Ignored},
	{draft, "Hi Sam, load HF-1 is on schedule and will deliver Friday by 2pm. Thanks, Ines", true, Edited},
	{draft, "Hi Sam, HF-1 is on schedule, delivering Friday by noon. Best, Ines", true, Edited},
	{draft, "Hi Sam, load HF-1 is on schedule and will deliver Friday by noon. POD to follow. Thanks, Ines", true, Edited},
	{draft, "Sam - the truck broke down in Joplin; new ETA Saturday. Sorry. Ines", true, Rewrote},
	{draft, "Hello, we have no record of that load. Please send the PRO number. Ines", true, Rewrote},
	{"Your claim for load HF-9 was received and is under review.",
		"Your claim for load HF-9 was received and is under review by our team.", true, Edited},
	{"Your claim for load HF-9 was received and is under review.",
		"Claim denied: HF-9 delivered clean per signed POD. Ines", true, Rewrote},
	// The known blind spot: one word flips the meaning and
	// overlap stays high. A person calls this a rewrite;
	// the heuristic calls it an edit.
	{draft, "Hi Sam, load HF-1 is not on schedule and will not deliver Friday by noon. Thanks, Ines", true, Rewrote},
}

func TestAgreesWithHumanSample(t *testing.T) {
	agree := 0
	var misses []string
	for _, c := range labelled {
		if got := Classify(c.draft, c.sent, c.used); got == c.human {
			agree++
		} else {
			misses = append(misses, string(c.human)+"->"+string(got)+": "+c.sent)
		}
	}
	// 10 of 11; the miss must be the meaning flip, which
	// is why the book calls Classify a heuristic.
	if agree != len(labelled)-1 || len(misses) != 1 ||
		misses[0][:15] != "rewrote->edited" {
		t.Fatalf("agree %d/%d, misses %q", agree, len(labelled), misses)
	}
}

// The send tool records the body it actually sends,
// so an edit made in the send dialog counts.
func TestOnSendRecordsFinalBody(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	final := "Sam - the truck broke down in Joplin; new ETA Saturday. Sorry. Ines"
	if f := OnSend(log, "c-41", draft, final); f != Rewrote {
		t.Fatalf("fate = %s", f)
	}
	var ev map[string]any
	if err := json.Unmarshal(buf.Bytes(), &ev); err != nil {
		t.Fatal(err)
	}
	if ev["msg"] != "llm.outcome" || ev["call_id"] != "c-41" || ev["fate"] != "rewrote" {
		t.Fatalf("event = %v", ev)
	}
}
