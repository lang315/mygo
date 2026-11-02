//go:build windows && (amd64 || arm64)

package windows

import (
	"sync"
	"time"
)

// DRAFT: phase timestamps of the opening of a window, to measure where its
// time goes. Off (nil check only) unless a test starts it.
var phases struct {
	sync.Mutex
	on    bool
	start time.Time
	names []string
	at    []time.Duration
}

func mark(name string) { markAt(name, time.Now()) }

func markAt(name string, t time.Time) {
	if !phases.on {
		return
	}
	phases.Lock()
	phases.names = append(phases.names, name)
	phases.at = append(phases.at, t.Sub(phases.start))
	phases.Unlock()
}

// TestMark records a phase from a test.
func TestMark(name string) { mark(name) }

// TestPhasesStart starts a new trace.
func TestPhasesStart() {
	phases.Lock()
	phases.on, phases.start = true, time.Now()
	phases.names, phases.at = nil, nil
	phases.Unlock()
}

// TestPhases returns the trace.
func TestPhases() (names []string, at []time.Duration) {
	phases.Lock()
	defer phases.Unlock()
	return append([]string(nil), phases.names...), append([]time.Duration(nil), phases.at...)
}
