//go:build darwin

package darwin

import "testing"

// A function of the main queue may run a loop of its own, as a frame showing
// a modal dialog does (drawSoon): the loop runs dispatchCallback again for
// what waits in the main loop's queue (loop.drain), which dispatchCallback
// runs after this queue, and which may have been posted before it began, its
// signal taken by it. So the source is signaled before each function runs,
// not only when others of this queue wait.
func TestDrainMainQueueSignalsBeforeEachFunction(t *testing.T) {
	signals := 0
	var seen []int
	mainQueueMu.Lock()
	mainQueue = append(mainQueue,
		func() { seen = append(seen, signals) },
		func() { seen = append(seen, signals) },
	)
	mainQueueMu.Unlock()
	drainMainQueueWith(func() { signals++ })
	if len(seen) != 2 || seen[0] < 1 || seen[1] < 2 {
		t.Errorf("the signals counted when the functions ran = %v, want at least 1 and 2", seen)
	}
	mainQueueMu.Lock()
	left := len(mainQueue)
	mainQueueMu.Unlock()
	if left != 0 {
		t.Errorf("%d functions left in the queue", left)
	}
}

// Each function runs once, in the order posted, whether or not one of them
// posts another while it runs, and the queue is not locked while one runs
// (a nested loop posts and drains it again).
func TestDrainMainQueueRunsEachOnceInOrderAndUnlocked(t *testing.T) {
	var order []int
	var nested func()
	nested = func() {
		order = append(order, 1)
		// What a nested loop does: post, and drain the queue it is in.
		mainQueueMu.Lock()
		mainQueue = append(mainQueue, func() { order = append(order, 3) })
		mainQueueMu.Unlock()
		drainMainQueueWith(func() {})
		order = append(order, 2)
	}
	mainQueueMu.Lock()
	mainQueue = append(mainQueue, nested, func() { order = append(order, 4) })
	mainQueueMu.Unlock()
	drainMainQueueWith(func() {})
	// The nested drain runs what waits behind the function that opened it
	// (4, posted before 3), then 3, before the function returns; none twice.
	want := []int{1, 4, 3, 2}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}
