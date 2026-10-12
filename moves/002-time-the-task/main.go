// Command timetask reads a "task,seconds" CSV on
// stdin and prints p50 and p90 per task.
//
// Move 2: time the task before you automate it.
//
//	go run ./moves/002-time-the-task < log.csv
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
)

// The book's code starts here.

func main() {
	in := csv.NewReader(os.Stdin)
	in.FieldsPerRecord = -1 // allow ragged rows
	rows, err := in.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	by := map[string][]float64{}
	bad := 0
	for _, r := range rows {
		if len(r) < 2 {
			bad++
			continue
		}
		s, err := strconv.ParseFloat(r[1], 64)
		if err != nil {
			bad++ // the header and typos land here
			continue
		}
		by[r[0]] = append(by[r[0]], s)
	}
	log.Printf("skipped %d rows", bad)
	for _, t := range slices.Sorted(maps.Keys(by)) {
		xs := by[t]
		slices.Sort(xs)
		fmt.Printf("%-12s n=%-3d p50=%4.0fs p90=%4.0fs\n",
			t, len(xs), rank(xs, 50), rank(xs, 90))
	}
}

func rank(xs []float64, p int) float64 {
	k := (p*len(xs) + 99) / 100
	return xs[k-1]
}
