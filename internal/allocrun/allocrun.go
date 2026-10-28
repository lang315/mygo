// Package allocrun counts the allocations of a function for tests that
// fail above a number, as testing.AllocsPerRun does, without failing for
// the mallocs of others.
package allocrun

import "testing"

// tries is how many counts Least takes at most.
const tries = 5

// Least is testing.AllocsPerRun(runs, f) taken again until two counts agree
// or tries counts were taken, and the least of them. testing.AllocsPerRun
// counts the mallocs of every goroutine and of the runtime, so the mallocs
// of an earlier test's timer or goroutine, or of the runtime starting a
// thread (a loaded runner does), fail a test of a function that is steady.
// Such mallocs only add to a count; what f allocates in every run is in
// each. A function that allocates nothing is counted once.
func Least(runs int, f func()) float64 {
	best := testing.AllocsPerRun(runs, f)
	for try := 1; best > 0 && try < tries; try++ {
		n := testing.AllocsPerRun(runs, f)
		if n == best {
			break
		}
		best = min(best, n)
	}
	return best
}
