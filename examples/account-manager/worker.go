package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

// result is what a worker did to an account, for the main thread to show.
type result struct {
	id     int
	state  state
	status string
	at     int64 // when the job ended, 0 while it runs or once stopped
	ip     string
	// What a stress run changes besides: friends and groups added, and
	// the note, when not "".
	friends, groups int
	note            string
}

// runner runs jobs on accounts in goroutines, as fhtool's Start does. The
// workers never touch the rows: they add their results to a batch the
// main thread takes a few times a second.
type runner struct {
	cancel context.CancelFunc
	done   chan struct{} // closed once every worker returned
	mu     sync.Mutex
	batch  []result
}

// run runs work on the accounts ids, loops+1 times over, in threads
// goroutines, or one for each account when there are fewer. A loop starts
// once the one before it ended, for no account to be in two jobs at once.
func run(ids []int, loops, threads int, work func(ctx context.Context, id int) result) *runner {
	ctx, cancel := context.WithCancel(context.Background())
	r := &runner{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(r.done)
		defer cancel()
		for range loops + 1 {
			if ctx.Err() != nil {
				return
			}
			r.loop(ctx, ids, threads, work)
		}
	}()
	return r
}

// loop runs work once on each of ids, and returns when the workers did.
func (r *runner) loop(ctx context.Context, ids []int, threads int, work func(ctx context.Context, id int) result) {
	jobs := make(chan int)
	go func() {
		defer close(jobs)
		for _, id := range ids {
			select {
			case jobs <- id:
			case <-ctx.Done():
				return
			}
		}
	}()
	var wg sync.WaitGroup
	for range workers(threads, len(ids)) {
		wg.Go(func() {
			for id := range jobs {
				r.push(result{id: id, state: running, status: "Đang chạy..."})
				// A job stopped still reports, for its row not to stay
				// running.
				r.push(work(ctx, id))
				if ctx.Err() != nil {
					return
				}
			}
		})
	}
	wg.Wait()
}

// maxWorkers caps the goroutines of a run: past it, a batch of results
// changes so many rows that the view sorts them all again at every tick.
const maxWorkers = 1000

// workers is how many goroutines run the jobs on accounts: threads, but
// no more than there are accounts, nor maxWorkers, for "Số luồng" at
// 1,000,000 not to start as many goroutines.
func workers(threads, accounts int) int { return max(1, min(threads, accounts, maxWorkers)) }

func (r *runner) push(res result) {
	r.mu.Lock()
	r.batch = append(r.batch, res)
	r.mu.Unlock()
}

// take returns the results since it was last called.
func (r *runner) take() []result {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.batch
	r.batch = nil
	return b
}

// fakeWork stands for one of fhtool's actions: 50 to 300 ms, one in ten
// failing.
func fakeWork(action string) func(ctx context.Context, id int) result {
	return func(ctx context.Context, id int) result {
		select {
		case <-time.After(time.Duration(50+rand.IntN(250)) * time.Millisecond):
		case <-ctx.Done():
			return result{id: id, state: idle, status: "Đã dừng"}
		}
		ip := fmt.Sprintf("113.%d.%d.%d", rand.IntN(256), rand.IntN(256), rand.IntN(256))
		if rand.IntN(10) == 0 {
			return result{id: id, state: failed, status: "Lỗi: checkpoint", at: time.Now().Unix(), ip: ip}
		}
		return result{id: id, state: succeeded, status: action + ": Thành công", at: time.Now().Unix(), ip: ip}
	}
}
