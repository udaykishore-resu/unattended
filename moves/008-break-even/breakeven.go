// Package breakeven prices attempts in plain-input
// tokens and sets a token budget per kept result.
//
// Move 8: calculate break-even in tokens, not hours.
package breakeven

import "github.com/udaykishore-resu/unattended/llm"

// Ratios are prices relative to plain input,
// read off your vendor's rate card.
type Ratios struct{ Read, Write, Out float64 }

// Equiv prices one attempt in plain-input tokens.
func Equiv(u llm.Usage, r Ratios) float64 {
	return float64(u.InputTokens) +
		float64(u.CacheReadTokens)*r.Read +
		float64(u.CacheWriteTokens)*r.Write +
		float64(u.OutputTokens)*r.Out
}

// PerKept counts failed attempts too.
func PerKept(perTry, tries, kept float64) float64 {
	return perTry * tries / kept
}

// Budget: most plain-input tokens a kept result
// may cost. priceIn is per plain input token.
func Budget(savedMin, reviewMin, wagePerMin,
	priceIn float64) float64 {
	return (savedMin - reviewMin) * wagePerMin / priceIn
}
