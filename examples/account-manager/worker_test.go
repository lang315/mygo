package main

import (
	"context"
	"sync/atomic"
	"testing"
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
	var at, most atomic.Int32
	block := make(chan struct{})
	r := run([]int{1, 2, 3}, 0, 1_000_000, func(ctx context.Context, id int) result {
		n := at.Add(1)
		for {
			m := most.Load()
			if n <= m || most.CompareAndSwap(m, n) {
				break
			}
		}
		<-block
		at.Add(-1)
		return result{id: id, state: succeeded}
	})
	close(block)
	<-r.done
	if most.Load() > 3 {
		t.Fatalf("%d jobs at once for 3 accounts", most.Load())
	}
}
