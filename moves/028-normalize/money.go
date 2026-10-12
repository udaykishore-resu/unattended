package normalize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Money is an amount in integer minor units (cents)
// with the ISO code the model copied as its own field.
type Money struct {
	Cents    int64
	Currency string
}

// Halcyon handles these two-decimal currencies only.
var currencies = map[string]bool{"USD": true, "CAD": true}

// One strict pattern per locale. A leading minus is
// allowed; parentheses, a trailing minus, a lone
// decimal digit and mixed separators are not. At most
// 12 integer digits, so the result fits in int64.
var amount = map[string]*regexp.Regexp{
	"US": regexp.MustCompile(
		`^(-)?\$?(\d{1,3}(?:,\d{3}){0,3}|\d{1,12})(?:\.(\d{2}))?$`),
	"EU": regexp.MustCompile(
		`^(-)?(\d{1,3}(?:\.\d{3}){0,3}|\d{1,12})(?:,(\d{2}))?$`),
}

// ParseMoney reads an amount written the shipper's way
// (locale "US": 1,240.00; "EU": 1.240,00). The currency
// code comes from its own field; "$" alone is not a
// currency, since USD and CAD both use it.
func ParseMoney(raw, currency, locale string) (Money, error) {
	re, ok := amount[locale]
	if !ok {
		return Money{}, fmt.Errorf("no locale %q on shipper", locale)
	}
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if !currencies[cur] {
		return Money{}, fmt.Errorf("unsupported currency %q", currency)
	}
	s := strings.TrimSpace(raw)
	s = strings.TrimSpace(strings.TrimPrefix(s, cur))
	m := re.FindStringSubmatch(s)
	if m == nil {
		return Money{}, fmt.Errorf("unparsed amount %q (%s)", raw, locale)
	}
	whole := strings.NewReplacer(",", "", ".", "").Replace(m[2])
	n, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return Money{}, err
	}
	cents := n * 100
	if m[3] != "" {
		c, _ := strconv.ParseInt(m[3], 10, 64)
		cents += c
	}
	if m[1] == "-" {
		cents = -cents
	}
	return Money{cents, cur}, nil
}

// ParseWeightLb converts a copied weight to whole
// pounds. "tons" is refused: short, metric and long
// tons differ, so the shipper has to say which.
func ParseWeightLb(value int64, unit string) (int64, error) {
	if value <= 0 || value > 1_000_000 {
		return 0, fmt.Errorf("weight %d out of range", value)
	}
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "lb", "lbs", "pounds":
		return value, nil
	case "kg", "kgs", "kilograms":
		// 1 lb = 0.45359237 kg exactly; round half up.
		return (value*100_000_000 + 45_359_237/2) / 45_359_237, nil
	case "ton", "tons", "t", "tonne", "tonnes":
		return 0, fmt.Errorf("ambiguous unit %q: ask the shipper", unit)
	default:
		return 0, fmt.Errorf("unsupported unit %q", unit)
	}
}
