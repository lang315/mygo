//go:build windows && arm64

package wincb

import (
	"sync"
	"syscall"
)

// entries is how many callbacks New makes (see table_arm64.s).
const entries = 1024

// entrySize is the size of an entry of cbTable, in bytes.
const entrySize = 8

var (
	cbTargets [entries]uintptr // the entry of syscall.NewCallback of each
	mu        sync.Mutex
	next      int
	made      = map[uintptr]uintptr{} // the entry of syscall.NewCallback -> the one New returned
)

func cbTable()
func cbCommon()
func cbTableAddr() uintptr

// New returns the address native code calls fn at, as syscall.NewCallback
// does, but an entry that keeps F8-F15 (see the package). Callbacks are
// never freed, like syscall.NewCallback's, and one function value gets one.
func New(fn any) uintptr {
	target := syscall.NewCallback(fn)
	mu.Lock()
	defer mu.Unlock()
	if entry, ok := made[target]; ok {
		return entry
	}
	if next == entries {
		panic("wincb: too many callbacks")
	}
	cbTargets[next] = target
	entry := cbTableAddr() + uintptr(next)*entrySize
	made[target] = entry
	next++
	return entry
}
