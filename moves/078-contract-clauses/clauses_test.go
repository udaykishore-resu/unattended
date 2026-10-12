package clauses

import (
	"reflect"
	"testing"
)

func days(n int) *int { return &n }

// Payment terms on page 6 said 30 days; a clause on
// page 33 changed them to 60. Ask for every match.
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

func TestInventedQuoteAndMissingCap(t *testing.T) {
	pages := []string{"Net 30.", "Termination on 60 days notice."}
	cs := []Clause{{"termination", "either party may terminate at will", 2, nil}}
	want := []Flag{{"quote", "not on page"}, {"cap", "no cap found"}}
	if got := Review(cs, pages); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}
