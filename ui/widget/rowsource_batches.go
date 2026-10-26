package widget

import "sync/atomic"

// AddRowsInBatches adds rows as AddRows does, but in a goroutine, batch rows
// at a time, so the tables over the source show them as they come and the app
// can tell how far it is: progress(done, total) runs on the event goroutine
// (as Post) after each batch, ending at (total, total); for no rows it runs
// once with (0, 0). Bind it to a ProgressBar:
//
//	bar := widget.StatusProgress()
//	src.AddRowsInBatches(rows, 5000, func(done, total int) {
//		bar.SetFraction(float64(done) / float64(total))
//	})
//
// A batch of less than 1, or more than the rows, is one batch. The returned
// cancel stops the load after the batch in hand: the rows of the batches
// after it are not added and progress is not called again. rows must not be
// changed until the last batch is added. A table redraws its rows for each
// batch, which costs it as many rows as it holds: with a million rows keep
// the batches large (thousands).
func (s *RowSource[T, K]) AddRowsInBatches(rows []T, batch int, progress func(done, total int)) (cancel func()) {
	var stopped atomic.Bool
	total := len(rows)
	if batch < 1 || batch > total {
		batch = max(total, 1)
	}
	report := func(done int) {
		if progress != nil && !stopped.Load() {
			Post(func() {
				if !stopped.Load() {
					progress(done, total)
				}
			})
		}
	}
	go func() {
		if total == 0 {
			report(0)
			return
		}
		for done := 0; done < total && !stopped.Load(); {
			end := min(done+batch, total)
			s.AddRows(rows[done:end])
			done = end
			report(done)
		}
	}()
	return func() { stopped.Store(true) }
}
