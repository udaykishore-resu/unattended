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
	"os"
	"slices"
	"strconv"
)

func main() {
	rows, err := csv.NewReader(os.Stdin).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	by := map[string][]float64{}
	for _, r := range rows {
		s, err := strconv.ParseFloat(r[1], 64)
		if err != nil {
			continue // skip the header and typos
		}
		by[r[0]] = append(by[r[0]], s)
	}
	for task, xs := range by {
		slices.Sort(xs)
		n := len(xs)
		fmt.Printf("%-12s n=%-3d p50=%4.0fs p90=%4.0fs\n",
			task, n, xs[n/2], xs[n*9/10])
	}
}
