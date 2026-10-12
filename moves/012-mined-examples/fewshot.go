// Package fewshot picks few-shot examples from the
// correction log, capped per label.
//
// Move 12: few-shot examples mined from your own
// failure log.
package fewshot

// Fix is one reviewed correction.
type Fix struct{ Input, Got, Want string }

// pickShots keeps at most perLabel fixes per
// corrected label. Rows arrive newest first.
func pickShots(fixes []Fix, perLabel, limit int) []Fix {
	seen := map[string]int{}
	var out []Fix
	for _, f := range fixes {
		if len(out) == limit {
			break
		}
		if seen[f.Want] == perLabel {
			continue
		}
		seen[f.Want]++
		out = append(out, f)
	}
	return out
}
