// Package tiers scores an action's blast radius,
// not its tool name, to pick who approves it.
//
// Move 98: approval tiers by blast radius.
package tiers

type Action struct {
	Tool       string
	Cents      int64 // money moved, if any
	Outside    int   // outside parties touched
	Reversible bool
}

func TierFor(a Action) Tier {
	switch {
	case a.Cents > 500_000: // over $5,000
		return Never
	case a.Cents > 0:
		return TwoKey
	case a.Outside > 0, !a.Reversible:
		return Review
	}
	return Auto
}

// Tier matches the risk colors. Described in prose.
type Tier int

const (
	Auto   Tier = iota // Green: runs, gets logged
	Review             // Amber: one human approves
	TwoKey             // Red: two humans, one finance
	Never              // above the limit: draft only
)

func (t Tier) String() string {
	return [...]string{"Auto", "Review", "TwoKey", "Never"}[t]
}
