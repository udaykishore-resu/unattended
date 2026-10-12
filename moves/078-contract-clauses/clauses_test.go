package clauses

import (
	"reflect"
	"slices"
	"testing"
)

func days(n int) *int { return &n }

// Payment terms on page 6 said 30 days; a clause on
// page 33 changed them to 60. Ask for every match:
// both payment clauses are checked, not just the first.
func TestPage33(t *testing.T) {
	pages := make([]string, 41)
	pages[5] = "Carrier shall be paid within thirty (30) days of invoice."
	pages[32] = "Notwithstanding Section 4, payment shall be made within sixty (60) days."
	pages[11] = "Liability is capped at $100,000 per occurrence."
	cs := []Clause{
		{"payment_terms", "paid within thirty (30) days", 6, days(30)},
		{"payment_terms", "payment shall be made within sixty (60) days", 33, days(60)},
		{"liability_cap", "Liability is capped at $100,000", 12, nil},
	}
	want := []Flag{{"pay", "over 45 days"}}
	if got := Review(cs, pages); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

// The loop must run over every clause, and the
// missing-kind check runs once, after it. A flag on
// the last clause still fires; the missing cap is
// listed first.
func TestEveryClauseCheckedMissingFirst(t *testing.T) {
	pages := []string{"Either party may terminate on 30 days notice.",
		"Indemnity applies.", "Payment due within ninety (90) days."}
	cs := []Clause{
		{"termination", "terminate on 30 days notice", 1, nil},
		{"indemnity", "Indemnity applies.", 2, nil},
		{"payment_terms", "within ninety (90) days", 3, days(90)},
	}
	want := []Flag{{"cap", "no cap found"}, {"pay", "over 45 days"}}
	if got := Review(cs, pages); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestInventedQuoteAndMissingCap(t *testing.T) {
	pages := []string{"Net 30.", "Termination on 60 days notice."}
	cs := []Clause{{"termination", "either party may terminate at will", 2, nil}}
	want := []Flag{{"cap", "no cap found"}, {"quote", "not on page"}}
	if got := Review(cs, pages); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

// strings.Contains(page, "") is true for every page,
// so an empty quote must be rejected explicitly.
func TestEmptyQuoteRejected(t *testing.T) {
	pages := []string{"Liability is capped at $1."}
	for _, q := range []string{"", "  \n"} {
		cs := []Clause{{"liability_cap", q, 1, nil}}
		got := Review(cs, pages)
		want := []Flag{{"cap", "no cap found"}, {"quote", "not on page"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("quote %q: got %v", q, got)
		}
	}
}

// Page offsets and clauses that span a page break
// both fail the quote check: the safe direction.
func TestOffsetAndSpanningQuotesFlagged(t *testing.T) {
	pages := []string{"Liability is capped", "at $50,000. Fees are due"}
	cs := []Clause{
		{"liability_cap", "Liability is capped at $50,000", 1, nil}, // spans 1-2
		{"liability_cap", "Liability is capped", 2, nil},            // off by one
		{"liability_cap", "Liability is capped", 0, nil},
		{"liability_cap", "Liability is capped", 3, nil},
	}
	got := Review(cs, pages)
	if len(got) != 5 || got[0] != (Flag{"cap", "no cap found"}) {
		t.Fatalf("got %v", got)
	}
}

// The 45-day rule runs on the number Go reads from
// the quote. The model's number only has to agree.
func TestRuleRunsOnParsedEvidence(t *testing.T) {
	pages := []string{"Invoices are payable within sixty (60) days. Net 30 for fuel."}
	cases := []struct {
		name string
		c    Clause
		want []Flag
	}{
		{"model dropped the number", Clause{"payment_terms", "payable within sixty (60) days", 1, nil},
			[]Flag{{"pay", "over 45 days"}, {"pay", "check days"}}},
		{"model says 30, quote says 60", Clause{"payment_terms", "payable within sixty (60) days", 1, days(30)},
			[]Flag{{"pay", "over 45 days"}, {"pay", "check days"}}},
		{"agree, under the line", Clause{"payment_terms", "Net 30", 1, days(30)}, nil},
		{"spelled out only", Clause{"payment_terms", "within sixty", 1, days(60)},
			[]Flag{{"pay", "check days"}}},
	}
	for _, c := range cases {
		got := Review([]Clause{c.c, {"liability_cap", "Invoices", 1, nil}}, pages)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestQuoteDays(t *testing.T) {
	cases := map[string]int{
		"sixty (60) days":                    60,
		"within 45 calendar days of invoice": 45,
		"Net 30":                             30,
		"30 days, extended to 60 days":       60,
		"1 day":                              1,
	}
	for q, want := range cases {
		if got, ok := quoteDays(q); !ok || got != want {
			t.Errorf("%q: got %d %v", q, got, ok)
		}
	}
	if _, ok := quoteDays("upon receipt"); ok {
		t.Error("no number should be not-ok")
	}
}

// A scan with no text layer: fail loudly.
func TestThinPages(t *testing.T) {
	pages := []string{"Real text on this page, plenty of it.", " \n ", "x"}
	if got := Thin(pages, 10); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("got %v", got)
	}
}
