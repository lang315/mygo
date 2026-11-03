package widget

import "testing"

// updateLocked is UpdateRow's body for a caller that holds the package's
// lock: it must not take it again.
func TestUpdateLockedRunsUnderTheLock(t *testing.T) {
	src := NewRowSource(acctKey)
	src.SetRows([]acct{{1, "a", 1}})
	before := src.Version(1)
	mu.Lock()
	wins, ok := src.updateLocked(1, func(a *acct) { a.name = "b" })
	mu.Unlock()
	for _, w := range wins {
		invalidate(w)
	}
	got, _ := src.Row(1)
	if !ok || got.name != "b" {
		t.Fatalf("updateLocked: ok %v row %+v, want the row changed", ok, got)
	}
	if src.Version(1) == before {
		t.Error("updateLocked did not bump the row's generation")
	}
	mu.Lock()
	_, ok = src.updateLocked(99, func(a *acct) { t.Error("fn ran for a key the source does not hold") })
	mu.Unlock()
	if ok {
		t.Error("updateLocked reported a row for an unknown key")
	}
}
