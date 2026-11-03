package widget

import "testing"

// A panic in a column function is returned with its stack, not propagated:
// the caller unlocks, then reports it.
func TestRunColumnFnRecovers(t *testing.T) {
	v, stack, panicked := runColumnFn(func() { panic("boom") })
	if !panicked || v != "boom" || len(stack) == 0 {
		t.Fatalf("runColumnFn = %v, %d bytes of stack, %v; want boom, a stack, true", v, len(stack), panicked)
	}
	if _, _, panicked := runColumnFn(func() {}); panicked {
		t.Error("runColumnFn reported a panic for a function that returned")
	}
}

// The recovery is a defer of runColumnFn itself, so a caller that holds the
// package's lock gets it back to unlock: the lock is free afterwards.
func TestRunColumnFnLeavesTheLockToTheCaller(t *testing.T) {
	func() {
		mu.Lock()
		defer mu.Unlock()
		runColumnFn(func() { panic("boom") })
	}()
	mu.Lock() // hangs if the panic left the lock held
	mu.Unlock()
}
