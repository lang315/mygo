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

func TestStressFeedsAboutTheRate(t *testing.T) {
	r := stress(100, 1000, 1)
	time.Sleep(200 * time.Millisecond)
	r.cancel()
	<-r.done
	got := r.take()
	// 200 at 1000 a second for 200 ms, give or take the scheduler.
	if len(got) < 100 || len(got) > 400 {
		t.Fatalf("%d changes in 200 ms at 1000 a second", len(got))
	}
	for _, x := range got {
		if x.id >= 100 {
			t.Fatalf("account %d of 100", x.id)
		}
	}
}
