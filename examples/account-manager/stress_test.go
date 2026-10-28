package main

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"
)

func TestChangeIsTheSameForASeed(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	r1, r2 := rand.New(rand.NewPCG(1, 2)), rand.New(rand.NewPCG(1, 2))
	seen := map[state]bool{}
	for range 1000 {
		x, y := change(r1, 50, now), change(r2, 50, now)
		if x != y {
			t.Fatalf("one seed, two changes: %+v, %+v", x, y)
		}
		if x.id < 0 || x.id >= 50 {
			t.Fatalf("account %d of 50", x.id)
		}
		switch x.state {
		case running:
			if !strings.HasPrefix(x.status, "Đang chạy: bước ") || x.at != 0 {
				t.Fatalf("running: %+v", x)
			}
		case succeeded, failed:
			if x.at != now.Unix() || x.ip == "" {
				t.Fatalf("ended: %+v", x)
			}
		default:
			t.Fatalf("state %d", x.state)
		}
		if x.friends < -5 || x.friends > 5 || x.groups < -1 || x.groups > 1 || x.note == "" {
			t.Fatalf("change %+v", x)
		}
		seen[x.state] = true
	}
	if len(seen) != 3 {
		t.Fatalf("states seen: %v", seen)
	}
}

// fed runs a feed of 100 accounts at 1000 a second through the ticks at
// the given times after its start, then stops it, and returns what it made.
func fed(t *testing.T, after ...time.Duration) []result {
	t.Helper()
	start := time.Unix(1_800_000_000, 0)
	ticks := make(chan time.Time)
	r := feed(100, 1000, 1, start, ticks)
	for _, d := range after {
		ticks <- start.Add(d)
	}
	// The feed took the last tick and made its changes before it looks at
	// its context again: no tick is waiting.
	r.cancel()
	<-r.done
	return r.take()
}

func TestStressFeedsTheRateOfItsTicks(t *testing.T) {
	var after []time.Duration
	for ms := 10; ms <= 200; ms += 10 {
		after = append(after, time.Duration(ms)*time.Millisecond)
	}
	got := fed(t, after...)
	if len(got) != 200 {
		t.Fatalf("%d changes in 200 ms at 1000 a second, want 200", len(got))
	}
	for _, x := range got {
		if x.id < 0 || x.id >= 100 {
			t.Fatalf("account %d of 100", x.id)
		}
	}
}

func TestStressCatchesUpAfterALateTick(t *testing.T) {
	ms := time.Millisecond
	if got := fed(t, 10*ms, 100*ms); len(got) != 100 {
		t.Fatalf("%d changes after a tick 90 ms late, want 100", len(got))
	}
}

func TestStressMakesNothingBeforeATick(t *testing.T) {
	if got := fed(t); len(got) != 0 {
		t.Fatalf("%d changes without a tick", len(got))
	}
}

func TestStressStopsOnCancel(t *testing.T) {
	start := time.Unix(1_800_000_000, 0)
	ticks := make(chan time.Time)
	r := feed(100, 1000, 1, start, ticks)
	ticks <- start.Add(10 * time.Millisecond)
	r.cancel()
	<-r.done
	select {
	case ticks <- start.Add(time.Second):
		t.Fatal("a stopped feed took a tick")
	default:
	}
	if got := r.take(); len(got) != 10 {
		t.Fatalf("%d changes, want the 10 of the tick before the cancel", len(got))
	}
}
