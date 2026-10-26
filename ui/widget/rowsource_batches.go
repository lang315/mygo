package widget

import (
	"slices"
	"sync/atomic"
)

// AddRowsInBatches adds rows as AddRows does, but in a goroutine, batch rows
// at a time, so the tables over the source show them as they come and the app
// can tell how far it is: progress(done, total) runs on the event goroutine
// (as Post) after each batch, ending at (total, total); for no rows it runs
// once with (0, 0). Bind it to a ProgressBar:
//
//	bar := widget.StatusProgress()
//	src.AddRowsInBatches(rows, 5000, func(done, total int) {
//		bar.SetFraction(float64(done) / float64(max(total, 1)))
//	})
//
// A batch of less than 1, or more than the rows, is one batch. rows must not
// be changed until the load ends. Two loads at once interleave their batches.
//
// The returned cancel stops the load after the batch in hand and waits for
// the goroutine to end, so rows are added no more once it returns, and no
// report runs after it, even one already posted. Call it before SetRows or
// DeleteRows if the load must not bring rows back, and before reusing rows.
// It may be called from progress and more than once, but not from the key
// func, UpdateRow's fn or a view's build, which run under the package's lock
// the loader needs. A panic in the key func ends the load and is reported as
// a handler's is (OnPanic); cancel then returns at once.
//
// The source grows once, at the start, for all the rows, so the batches
// append in place. A table adds the rows that arrive between two frames to its
// view by merging them, which costs a pass over the rows it holds; it sorts
// them all anew when the source's array changed or more than an eighth of its
// rows arrived. Batches of thousands keep a frame's share of that small.
func (s *RowSource[T, K]) AddRowsInBatches(rows []T, batch int, progress func(done, total int)) (cancel func()) {
	var cancelled atomic.Bool
	ended := make(chan struct{})
	total := len(rows)
	if batch < 1 || batch > total {
		batch = max(total, 1)
	}
	tell := func(done int) {
		if progress != nil && !cancelled.Load() {
			Post(func() {
				if !cancelled.Load() {
					progress(done, total)
				}
			})
		}
	}
	go func() {
		defer close(ended)
		run(func() {
			if total == 0 {
				tell(0)
				return
			}
			s.change(func() { s.rows = slices.Grow(s.rows, total) })
			for done := 0; done < total && !cancelled.Load(); {
				end := min(done+batch, total)
				s.AddRows(rows[done:end])
				done = end
				tell(done)
			}
		})
	}()
	return func() {
		cancelled.Store(true)
		<-ended
	}
}
