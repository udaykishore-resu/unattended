package normalize

import (
	"testing"
	"time"
)

func chicago(t *testing.T) *time.Location {
	chi, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	return chi
}

func TestParseDate(t *testing.T) {
	chi := chicago(t)
	want := time.Date(2026, 3, 4, 0, 0, 0, 0, chi)
	for _, raw := range []string{
		"2026-03-04", "03/04/2026", "3/4/2026", " Mar 4, 2026 ",
		"March 4, 2026", "4 Mar 2026",
	} {
		got, err := ParseDate(raw, "US", chi)
		if err != nil || !got.Equal(want) {
			t.Errorf("%q: got %v, %v", raw, got, err)
		}
	}
}

// The Rotterdam case: the same string means different
// days, and only the shipper's stored order decides.
func TestDayOrderComesFromTheShipper(t *testing.T) {
	us, err := ParseDate("03/04/2026", "US", time.UTC)
	if err != nil || us.Month() != time.March || us.Day() != 4 {
		t.Fatalf("US: %v %v", us, err)
	}
	eu, err := ParseDate("03/04/2026", "EU", time.UTC)
	if err != nil || eu.Month() != time.April || eu.Day() != 3 {
		t.Fatalf("EU: %v %v", eu, err)
	}
	// 13/04 is a valid EU date and never a US one: no fallback.
	if _, err := ParseDate("13/04/2026", "US", time.UTC); err == nil {
		t.Fatal("US shipper: month 13 must fail, not flip to EU")
	}
	// No order on the shipper record: fail closed.
	for _, order := range []string{"", "us", "ISO"} {
		if _, err := ParseDate("2026-03-04", order, time.UTC); err == nil {
			t.Errorf("order %q accepted", order)
		}
	}
}

// Parsed in the shipper's zone, a midnight pickup
// stays on its own day.
func TestZoneKeepsTheDay(t *testing.T) {
	chi := chicago(t)
	got, err := ParseDate("2026-03-04", "US", chi)
	if err != nil || got.In(chi).Day() != 4 {
		t.Fatalf("day = %d, err = %v", got.In(chi).Day(), err)
	}
}

func TestUnparsedIsAnError(t *testing.T) {
	for _, raw := range []string{
		"approx Thursday", "04.03.2026", "", "3/12", "3/4/26",
		"02/30/2026", "2026-02-29", "Feb 30, 2026", "00/10/2026",
	} {
		if _, err := ParseDate(raw, "US", time.UTC); err == nil {
			t.Errorf("%q parsed; want an error", raw)
		}
	}
}

func TestParseMoney(t *testing.T) {
	cases := []struct {
		raw, cur, loc string
		want          int64
	}{
		{"$2,150.00", "USD", "US", 215000},
		{"2150", "usd", "US", 215000},
		{"1,240.5", "USD", "US", -1}, // one decimal digit
		{"1.240,00", "CAD", "EU", 124000},
		{"CAD 1.240,00", "CAD", "EU", 124000},
		{"1.240,00", "CAD", "US", -1},  // EU form under a US rule
		{"1,240.00", "CAD", "EU", -1},  // US form under an EU rule
		{"1.240", "USD", "US", -1},     // three decimals: ambiguous
		{"-75.00", "USD", "US", -7500}, // credit line
		{"(75.00)", "USD", "US", -1},   // accounting style: refuse
		{"75.00-", "USD", "US", -1},
		{"12,34,567.00", "USD", "US", -1},
		{"€1.240,00", "EUR", "EU", -1}, // unsupported currency
		{"1,240.00", "", "US", -1},     // no currency field
		{"9999999999999", "USD", "US", -1},
		{"999,999,999,999.99", "USD", "US", 99999999999999},
	}
	for _, c := range cases {
		got, err := ParseMoney(c.raw, c.cur, c.loc)
		if c.want == -1 {
			if err == nil {
				t.Errorf("%q %s %s: got %+v, want error", c.raw, c.cur, c.loc, got)
			}
			continue
		}
		if err != nil || got.Cents != c.want {
			t.Errorf("%q %s %s: got %+v, %v", c.raw, c.cur, c.loc, got, err)
		}
	}
	if _, err := ParseMoney("1.00", "USD", "FR"); err == nil {
		t.Error("unknown locale accepted")
	}
}

// Synthesis story (Part III): the golden set had no
// foreign-currency case. This is the row that would
// have caught the deleted parser.
func TestCADGoldenCase(t *testing.T) {
	got, err := ParseMoney("CAD 1.240,00", "CAD", "EU")
	if err != nil || got != (Money{124000, "CAD"}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestParseWeightLb(t *testing.T) {
	if lb, err := ParseWeightLb(42000, "lbs"); err != nil || lb != 42000 {
		t.Fatalf("lbs: %d %v", lb, err)
	}
	if lb, err := ParseWeightLb(1000, "kg"); err != nil || lb != 2205 {
		t.Fatalf("kg: %d %v", lb, err)
	}
	for _, u := range []string{"tons", "T", "tonne", "cwt", ""} {
		if _, err := ParseWeightLb(20, u); err == nil {
			t.Errorf("unit %q accepted", u)
		}
	}
	if _, err := ParseWeightLb(-5, "lb"); err == nil {
		t.Error("negative weight accepted")
	}
}
