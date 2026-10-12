// Package scoretasks ranks automation ideas.
//
// Move 3: score tasks on frequency x pain x
// reversibility. Multiply, don't add.
package scoretasks

type Task struct {
	Name    string
	PerWeek float64 // how often it happens
	Minutes float64 // median hands-on time
	Pain    int     // 1-5, rated by the doers
	Undo    int     // 1-5, 5 = trivially undone
}

// Score multiplies, so one weak factor sinks it.
func Score(t Task) float64 {
	hours := t.PerWeek * t.Minutes / 60
	return hours * float64(t.Pain) * float64(t.Undo)
}

// FirstProject drops hard-to-undo or rare tasks.
func FirstProject(t Task) bool {
	return t.Undo >= 3 && t.PerWeek >= 5
}
