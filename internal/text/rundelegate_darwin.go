//go:build darwin

package text

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The leaves of rundelegate_darwin_*.s.
func runDelegateDealloc()
func runDelegateAscent()
func runDelegateDescent()
func runDelegateWidth()
func runDelegateFuncs() (dealloc, ascent, descent, width uintptr)

// runDelegateCallbacks is a CTRunDelegateCallbacks, version 1.
type runDelegateCallbacks struct {
	version                         int
	dealloc, ascent, descent, width uintptr
}

// delegates are what makes Core Text's run delegates, which give inline
// objects their size, the process's.
type delegates struct {
	create    uintptr // CTRunDelegateCreate
	attribute uintptr // kCTRunDelegateAttributeName
	callbacks runDelegateCallbacks
}

var runDelegates delegates

// delegateMetrics are the metrics of the objects of the text an engine lays
// out, pinned while its typesetter lives (shapeText). Each engine has its own, which
// its System's lock guards: Systems lay out at once (a Tester's own,
// #1730), and Core Text reaches the metrics through each delegate's
// pointer.
type delegateMetrics struct {
	metrics [][3]float64
	pinner  runtime.Pinner
}

// initDelegates finds Core Text's run delegates in the framework text.
func initDelegates(text uintptr, optional func(lib uintptr, name string) uintptr) {
	d := &runDelegates
	if sym, err := purego.Dlsym(text, "CTRunDelegateCreate"); err == nil {
		d.create = sym
	}
	d.attribute = optional(text, "kCTRunDelegateAttributeName")
	dealloc, ascent, descent, width := runDelegateFuncs()
	d.callbacks = runDelegateCallbacks{version: 1, dealloc: dealloc, ascent: ascent, descent: descent, width: width}
}

// reset lets go of the metrics of the last text laid out, and makes room
// for n objects.
func (d *delegateMetrics) reset(n int) {
	d.pinner.Unpin()
	d.metrics = make([][3]float64, n)
	if n > 0 {
		d.pinner.Pin(&d.metrics[0])
	}
}

// delegate returns a run delegate, owned, giving object k its height
// above the baseline, its descent below it and its width.
func (d *delegateMetrics) delegate(k int, width, height, descent float32) uintptr {
	g := &runDelegates
	if g.create == 0 || g.attribute == 0 || k >= len(d.metrics) {
		return 0
	}
	d.metrics[k] = [3]float64{float64(height), float64(descent), float64(width)}
	r, _, _ := purego.SyscallN(g.create, uintptr(unsafe.Pointer(&g.callbacks)), uintptr(unsafe.Pointer(&d.metrics[k])))
	return r
}
