package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// last is the state each account was left in by results.
func last(results []result) map[int]state {
	m := map[int]state{}
	for _, r := range results {
		m[r.id] = r.state
	}
	return m
}

func TestRunRunsEachAccountLoopsTimes(t *testing.T) {
	var n atomic.Int32
	r := run([]int{1, 2, 3}, 1, 8, func(ctx context.Context, id int) result {
		n.Add(1)
		return result{id: id, state: succeeded}
	})
	<-r.done
	if n.Load() != 6 {
		t.Fatalf("%d jobs ran, want 3 accounts × 2 loops", n.Load())
	}
	got := r.take()
	if len(got) != 12 {
		t.Fatalf("%d results, want a running and a done one for each of 6 jobs", len(got))
	}
	for id, s := range last(got) {
		if s != succeeded {
			t.Errorf("account %d left %d", id, s)
		}
	}
	if r.take() != nil {
		t.Error("results taken twice")
	}
}

func TestStopLeavesNoAccountRunning(t *testing.T) {
	ids := make([]int, 100)
	for i := range ids {
		ids[i] = i
	}
	r := run(ids, 0, 10, fakeWork("Đăng nhập"))
	r.cancel()
	<-r.done
	for id, s := range last(r.take()) {
		if s == running {
			t.Errorf("account %d still running after Stop", id)
		}
	}
}

func TestRunStartsNoMoreWorkersThanAccounts(t *testing.T) {
	for _, tc := range []struct{ threads, accounts, want int }{
		{1_000_000, 3, 3},
		{1_000_000, 1_000_000, 1000},
		{10, 100, 10},
		{1, 0, 1},
	} {
		if got := workers(tc.threads, tc.accounts); got != tc.want {
			t.Errorf("workers(%d, %d) = %d, want %d", tc.threads, tc.accounts, got, tc.want)
		}
	}
}

// A loop starts once the one before it ended, as fhtool's do: one account
// in two jobs at once would be two browsers on it.
func TestRunNeverRunsAnAccountTwiceAtOnce(t *testing.T) {
	var mu sync.Mutex
	at, most := map[int]int{}, 0
	r := run([]int{1, 2, 3}, 3, 10, func(ctx context.Context, id int) result {
		mu.Lock()
		at[id]++
		most = max(most, at[id])
		mu.Unlock()
		time.Sleep(2 * time.Millisecond)
		mu.Lock()
		at[id]--
		mu.Unlock()
		return result{id: id, state: succeeded}
	})
	<-r.done
	if most > 1 {
		t.Fatalf("an account ran in %d jobs at once", most)
	}
}
