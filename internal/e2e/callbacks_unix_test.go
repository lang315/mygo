//go:build darwin || linux

package e2e

import (
	"reflect"
	"sync"
	_ "unsafe" // for go:linkname

	"github.com/ebitengine/purego"
)

// puregoCbs is purego's table of callbacks (syscall_unix.go): at most 2000
// for the whole process, never freed. The layout is purego's; callbackCount
// checks that it still counts (TestCallbackCounter).
//
//go:linkname puregoCbs github.com/ebitengine/purego.cbs
var puregoCbs struct {
	lock  sync.Mutex
	numFn int
	funcs [2000]reflect.Value
}

// callbackCount returns how many callbacks the process made, those of
// purego's objc package and of every package of MyGo among them. exact is
// false where the count is only that since the first call.
func callbackCount() (n int, exact bool) {
	puregoCbs.lock.Lock()
	defer puregoCbs.lock.Unlock()
	return puregoCbs.numFn, true
}

// makeCallback makes a callback, as the code under test does.
func makeCallback() { purego.NewCallback(func() {}) }
