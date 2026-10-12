package sampling

import "errors"

// ErrPlanted: the executor refuses a planted item.
var ErrPlanted = errors.New("planted item: not executed")

// Item is one output on its way to the executor.
// Planted marks a known-bad item seeded to check that
// reviewers catch it.
type Item struct {
	ID      string
	Planted bool
}

// Execute is the executor's choke point: a planted
// item that slips past review is refused here, so it
// is never sent, applied or paid, and it is not
// counted in metrics.
func Execute(it Item, do func() error,
	count func(id string)) error {
	if it.Planted {
		return ErrPlanted
	}
	if err := do(); err != nil {
		return err
	}
	count(it.ID)
	return nil
}

// Weighted is the stratified error estimate: each
// stratum's sampled error rate times its real volume,
// summed, over the total volume.
func Weighted(errRate map[string]float64,
	volume map[string]int) float64 {
	var errs, all float64
	for s, v := range volume {
		errs += errRate[s] * float64(v)
		all += float64(v)
	}
	if all == 0 {
		return 0
	}
	return errs / all
}
