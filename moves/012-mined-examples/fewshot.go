// Package fewshot picks few-shot examples from the
// correction log, capped per label.
//
// Move 12: few-shot examples mined from your own
// failure log.
package fewshot

// Fix is one reviewed correction.
type Fix struct{ Input, Got, Want string }

// pickShots keeps at most perLabel fixes per
// corrected label and skips repeated inputs.
// Rows arrive newest first.
func pickShots(fixes []Fix, perLabel, limit int) []Fix {
	seen := map[string]int{}
	dup := map[string]bool{}
	var out []Fix
	for _, f := range fixes {
		if len(out) == limit {
			break
		}
		if seen[f.Want] == perLabel || dup[f.Input] {
			continue
		}
		seen[f.Want]++
		dup[f.Input] = true
		out = append(out, f)
	}
	return out
}

// split is the held-out rule from the book's text,
// minimal: corrections before cutoff may become
// examples; the rest are the eval set, so no email
// is both a shot and a graded case. Rows hold a
// day number (days since the log began).
type Row struct {
	Fix
	Day int
}

func split(rows []Row, cutoff int) (dev, eval []Fix) {
	for _, r := range rows {
		if r.Day < cutoff {
			dev = append(dev, r.Fix)
		} else {
			eval = append(eval, r.Fix)
		}
	}
	return dev, eval
}
