package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

// stress changes rows as a busy run would, rate times a second, on random
// ones of the first accounts IDs, through the batches the workers' results
// take: the rows change all the time, for the stats to measure.
func stress(accounts, rate int, seed uint64) *runner {
	tick := time.NewTicker(10 * time.Millisecond)
	r := feed(accounts, rate, seed, time.Now(), tick.C)
	go func() {
		<-r.done
		tick.Stop()
	}()
	return r
}

// feed is stress with the ticks it counts time by, and the start it counts
// from, given: a test sends its own.
func feed(accounts, rate int, seed uint64, start time.Time, ticks <-chan time.Time) *runner {
	ctx, cancel := context.WithCancel(context.Background())
	r := &runner{cancel: cancel, done: make(chan struct{})}
	accounts = max(1, accounts)
	go func() {
		defer close(r.done)
		rnd := rand.New(rand.NewPCG(seed, 0))
		sent := 0
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticks:
				// As many as are due since the start, for a slow tick not
				// to lower the rate.
				for due := int(now.Sub(start).Seconds() * float64(rate)); sent < due; sent++ {
					r.push(change(rnd, accounts, now))				}
			}
		}
	}()
	return r
}

// steps are the statuses of a job on its way, made once: change makes
// thousands a second.
var steps = func() (s [7]string) {
	for i := range s {
		s[i] = fmt.Sprintf("Đang chạy: bước %d/7", i+1)
	}
	return s
}()

// change is what a busy run does to a random one of accounts in a step:
// a job on it moves a step on or ends, its friends and groups change, and
// its note says when.
func change(r *rand.Rand, accounts int, now time.Time) result {
	c := result{id: r.IntN(accounts), friends: r.IntN(11) - 5, groups: r.IntN(3) - 1, note: "Cập nhật " + now.Format("15:04:05")}
	switch step := r.IntN(8); {
	case step < 7:
		c.state, c.status = running, steps[step]
	case r.IntN(10) == 0:
		c.state, c.status, c.at = failed, "Lỗi: checkpoint", now.Unix()
	default:
		c.state, c.status, c.at = succeeded, "Kiểm tra: Thành công", now.Unix()
	}
	if c.at != 0 {
		c.ip = fmt.Sprintf("113.%d.%d.%d", r.IntN(256), r.IntN(256), r.IntN(256))
	}
	return c
}
