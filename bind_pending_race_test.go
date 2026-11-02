//go:build mygo_static && !js

package mygo

// Copy into the repo root (package mygo); run: go test -tags mygo_static -run TestRev338dClosedRace .

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

type raceSvc struct{}

func (raceSvc) R() {}

// A goroutine's Bind (no table) races App.Run's checkPending. pendBind reads
// closed under the lock, unlocks, then registerPending locks again: when
// checkPending runs in between, the service lands in the list after the check,
// Bind returns nil, nothing panics, and the service has no methods.
func TestRev338dClosedRace(t *testing.T) {
	saved := staticFallback
	staticFallback = nil
	defer func() { staticFallback = saved }()
	silent, n := 0, 20000
	for i := range n {
		name := fmt.Sprintf("Race%d", i)
		pending.Lock()
		pending.list, pending.closed = nil, false
		pending.Unlock()
		var bindErr error
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); bindErr = bind(name, raceSvc{}, false) }()
		runtime.Gosched()
		ranOK := func() (ok bool) {
			defer func() { ok = recover() == nil }()
			checkPending()
			return
		}()
		wg.Wait()
		pending.Lock()
		stuck := len(pending.list) > 0
		pending.Unlock()
		if ranOK && bindErr == nil && stuck {
			silent++
		}
		unbind(name)
	}
	pending.Lock()
	pending.list, pending.closed = nil, false
	pending.Unlock()
	t.Logf("silent (Bind nil, Run passed, service stuck without methods): %d of %d", silent, n)
	if silent > 0 {
		t.Errorf("BUG: %d silent binds", silent)
	}
}
