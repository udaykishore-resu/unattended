package redact

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestRedactAndRehydrate(t *testing.T) {
	in := "Driver Sam (404-555-0117) emailed from sam.r42@rdx-trucking.test. " +
		"Call 404-555-0117 again, or 312.555.0199."
	out, vault := Redact(in)
	for _, raw := range []string{"404-555-0117", "sam.r42@rdx-trucking.test", "312.555.0199"} {
		if strings.Contains(out, raw) {
			t.Fatalf("%s leaked: %s", raw, out)
		}
	}
	// The same value always gets the same tag.
	if strings.Count(out, "[PHONE_1]") != 2 || !strings.Contains(out, "[PHONE_2]") ||
		!strings.Contains(out, "[EMAIL_1]") {
		t.Fatalf("placeholders: %s", out)
	}
	// The digits in the address did not become a phone.
	if len(vault) != 3 {
		t.Fatalf("vault = %v", vault)
	}
	reply := "Thanks [EMAIL_1]; we'll call [PHONE_2]."
	got := Rehydrate(reply, vault)
	if got != "Thanks sam.r42@rdx-trucking.test; we'll call 312.555.0199." {
		t.Fatalf("rehydrated: %s", got)
	}
}

// Brackets keep PHONE_1 from matching inside PHONE_10.
func TestTenthNumberSurvives(t *testing.T) {
	var parts []string
	for i := range 10 {
		parts = append(parts, fmt.Sprintf("404-555-01%02d", i))
	}
	out, vault := Redact(strings.Join(parts, " "))
	if back := Rehydrate(out, vault); back != strings.Join(parts, " ") {
		t.Fatalf("round trip broke:\n%s", back)
	}
}

func TestInventedPlaceholderIsCaught(t *testing.T) {
	_, vault := Redact("call 404-555-0117")
	got := Rehydrate("call [PHONE_1] or [PHONE_3]", vault)
	if !slices.Equal(Leftovers(got), []string{"[PHONE_3]"}) {
		t.Fatalf("leftovers = %v", Leftovers(got))
	}
}
