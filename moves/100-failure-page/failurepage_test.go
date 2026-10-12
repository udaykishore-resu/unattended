package failurepage

import (
	"os"
	"strings"
	"testing"
)

func load(t *testing.T) (string, map[string]bool) {
	t.Helper()
	b, err := os.ReadFile("testdata/invoice-matcher.md")
	if err != nil {
		t.Fatal(err)
	}
	flags, err := FlagsInCode("testdata/src")
	if err != nil {
		t.Fatal(err)
	}
	return string(b), flags
}

func TestCompletePagePasses(t *testing.T) {
	page, flags := load(t)
	if errs := Check(page, flags); len(errs) != 0 {
		t.Fatal(errs)
	}
	if !flags["invoice_matcher"] || !flags["invoice_matcher_carrier"] {
		t.Fatalf("flags = %v", flags)
	}
}

// The kill switch moved to a new config system: the flag
// the page names is gone from code, and CI fails.
func TestStaleFlagFails(t *testing.T) {
	page, flags := load(t)
	delete(flags, "invoice_matcher")
	if errs := Check(page, flags); len(errs) != 1 ||
		!strings.Contains(errs[0].Error(), "not found in code") {
		t.Fatalf("errs = %v", errs)
	}
}

// Each line the on-call engineer needs must be filled:
// the second way to stop, who may act, how to confirm,
// and the escalation path.
func TestMissingStopAndEscalationLinesFail(t *testing.T) {
	page, flags := load(t)
	for _, key := range []string{"Second way:", "Who may flip it:",
		"Stopped when:", "Escalate:"} {
		i := strings.Index(page, key)
		j := i + strings.Index(page[i:], "\n")
		cut := page[:i] + key + page[j:]
		errs := Check(cut, flags)
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), key) {
			t.Errorf("%s: errs = %v", key, errs)
		}
	}
}

func TestTemplatePlaceholdersFail(t *testing.T) {
	page, flags := load(t)
	page = strings.Replace(page, "Escalate: Theo", "Escalate: <who>", 1)
	if errs := Check(page, flags); len(errs) == 0 {
		t.Fatal("an unfilled template passed")
	}
}

func TestGenericFailureLineFails(t *testing.T) {
	page, flags := load(t)
	i := strings.Index(page, "KNOWN FAILURE MODES\n") + len("KNOWN FAILURE MODES\n")
	j := strings.Index(page, "\nRECOVERY")
	page = page[:i] + "Bad output, investigate\n" + page[j:]
	if errs := Check(page, flags); len(errs) != 1 {
		t.Fatalf("errs = %v", errs)
	}
}
