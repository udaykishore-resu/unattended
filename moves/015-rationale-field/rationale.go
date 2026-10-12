// Package rationale keeps the model's notes in a
// field that the rest of the system never sees.
//
// Move 15: keep the model's rationale in a field
// code never reads.
package rationale

import (
	"encoding/json"
	"log/slog"
	"strings"
)

// notesLog is bound at startup to the store that
// already holds raw inputs, with that store's access
// list and retention. Notes quote customer text, so
// they never go to the general log pipeline. Named
// only in prose; it discards until bound.
var notesLog = slog.New(slog.DiscardHandler)

const maxNotes = 600 // bytes; strict mode may not cap it

// Verdict is what the model returns.
type Verdict struct {
	Notes    string `json:"notes"`
	Category string `json:"category"`
	LoadID   string `json:"load_id"`
}

// Decision is what the rest of the system sees.
type Decision struct{ Category, LoadID string }

func decide(raw []byte, trace string) (Decision, error) {
	var v Verdict
	if err := json.Unmarshal(raw, &v); err != nil {
		return Decision{}, err
	}
	if len(v.Notes) > maxNotes {
		n := v.Notes[:maxNotes]
		v.Notes = strings.ToValidUTF8(n, "")
	}
	notesLog.Debug("llm notes",
		"trace", trace, "notes", v.Notes)
	return Decision{v.Category, v.LoadID}, nil
}
