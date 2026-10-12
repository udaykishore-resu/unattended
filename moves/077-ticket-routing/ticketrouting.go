// Package ticketrouting shows a draft reply only if
// it cites a help article that was in the prompt.
//
// Move 77: support ticket routing plus a draft reply.
package ticketrouting

// Result is the model's output: a queue from an
// enum (including "unsure"), an urgent flag, the
// draft, and the article IDs it used.
type Result struct {
	Queue  string   `json:"queue"`
	Urgent bool     `json:"urgent"`
	Draft  string   `json:"draft"`
	KBIDs  []string `json:"kb_ids"`
}

// Keep a draft only if it cites an article
// that was actually in the prompt.
func Usable(r Result, shown map[string]bool) bool {
	if r.Queue == "unsure" || len(r.KBIDs) == 0 {
		return false
	}
	for _, id := range r.KBIDs {
		if !shown[id] {
			return false
		}
	}
	return true
}
