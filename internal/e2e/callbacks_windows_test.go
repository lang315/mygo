//go:build windows

package e2e

import (
	"sync"
	"syscall"
)

// syscall.NewCallback gives out the entries of a table of at most 2000 in
// order, each at the address stride bytes after the last; the runtime does
// not say how many it gave. callbackCount makes one callback to see where
// the table is, so the count is that of the callbacks made since it first
// ran, less its own. The runtime keeps one entry for each function value
// (compileCallback's cache): a function that captures nothing is one static
// value, whose callback is the same every time and costs a slot once, so the
// probes capture a variable of their own, as the functions that the code
// under test makes for each window would, which is what leaks.
var winMeter struct {
	once         sync.Once
	mu           sync.Mutex
	base, stride uintptr
	probes       int
}

func probeCallback() uintptr {
	x := new(int)
	return syscall.NewCallback(func(uintptr) uintptr { return uintptr(*x) })
}

func callbackCount() (n int, exact bool) {
	m := &winMeter
	m.once.Do(func() {
		p0, p1 := probeCallback(), probeCallback()
		m.base, m.stride = p1, p1-p0
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	p := probeCallback()
	m.probes++
	return int((p-m.base)/m.stride) - m.probes, false
}

// makeCallback makes a callback, as the code under test does.
func makeCallback() { probeCallback() }
