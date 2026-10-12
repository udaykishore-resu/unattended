// Package scoretasks ranks automation ideas.
//
// Move 3: score tasks on frequency x pain x
// reversibility. Multiply, don't add.
package scoretasks

import "fmt"

type Task struct {
	Name    string
	PerWeek float64 // how often it happens
	Minutes float64 // median hands-on time
	Pain    int     // 1-5, rated by the doers
	Undo    int     // 1-5, 5 = trivially undone
}

func (t Task) Valid() error {
	if t.Pain < 1 || t.Pain > 5 ||
		t.Undo < 1 || t.Undo > 5 {
		return fmt.Errorf("%s: rate 1-5", t.Name)
	}
	return nil
}

// Score multiplies, so one weak factor sinks it.
func Score(t Task) float64 {
	hours := t.PerWeek * t.Minutes / 60
	return hours * float64(t.Pain) * float64(t.Undo)
}

func FirstProject(t Task) bool {
	return t.Undo >= 3 && t.PerWeek >= 5
}
