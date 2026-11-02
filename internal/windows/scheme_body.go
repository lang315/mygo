//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

var _ platform.SchemeBodyWriter = (*schemeResponse)(nil)

var (
	procGetProcessHeap = kernel32.NewProc("GetProcessHeap")
	procHeapAlloc      = kernel32.NewProc("HeapAlloc")
	procHeapReAlloc    = kernel32.NewProc("HeapReAlloc")
	procHeapFree       = kernel32.NewProc("HeapFree")
)

var (
	errBodyClosed = errors.New("mygo: the response was finished or not started")
	errBodyMemory = errors.New("mygo: out of memory for the response")
)

// responseBody is a response's body as it arrives, in memory of the process's
// heap, not of Go's: WebView2 wants the whole body in a stream, which is made
// from this at the end. The goroutine serving the request writes it (the
// platform.SchemeBodyWriter), so Go's heap sees one buffer of the protocol
// package's, reused, not a copy of the body.
//
// Only memory is shared between the goroutine and the main thread: no COM
// object is touched off the main thread (a stream from SHCreateMemStream has no
// documented thread rule, so none is relied on). The mutex orders a write and
// the end of the response, which the contract does not let overlap anyway.
type responseBody struct {
	mu   sync.Mutex
	heap uintptr
	p    uintptr // the block, 0 before open and once taken
	n    int     // bytes written
	size int     // bytes allocated
	open bool
	used bool  // start ran: a body that ended does not start again
	err  error // the first write that could not be kept: the response is not the body
}

// start makes the body writable; hint is the size expected, or 0.
func (b *responseBody) start(hint int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.used {
		return // a response that ended does not start again
	}
	b.open, b.used = true, true
	if hint > 0 {
		b.grow(hint)
	}
}

// grow makes room for at least need bytes. It must hold b.mu.
func (b *responseBody) grow(need int) error {
	if need <= b.size {
		return nil
	}
	size := max(need, b.size*2, 4096)
	if b.heap == 0 {
		b.heap, _, _ = procGetProcessHeap.Call()
	}
	var p uintptr
	if b.p == 0 {
		p, _, _ = procHeapAlloc.Call(b.heap, 0, uintptr(size))
	} else {
		p, _, _ = procHeapReAlloc.Call(b.heap, 0, b.p, uintptr(size))
	}
	if p == 0 {
		b.err = errBodyMemory // a failed ReAlloc keeps the old block
		return errBodyMemory
	}
	b.p, b.size = p, size
	return nil
}

// startOnce opens a body that Respond did not (a responder that is written
// to on the main thread before it responded).
func (b *responseBody) startOnce() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.used {
		b.open, b.used = true, true
	}
}

// write appends p. It fails once the body was taken or freed.
func (b *responseBody) write(p []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return errBodyClosed
	}
	if len(p) == 0 {
		return nil
	}
	if err := b.grow(b.n + len(p)); err != nil {
		return err
	}
	copy(unsafe.Slice((*byte)(native(b.p)), b.size)[b.n:], p)
	b.n += len(p)
	return nil
}

// take ends the body and returns its bytes, valid until free.
func (b *responseBody) take() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.open = false
	if b.n == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(native(b.p)), b.n)
}

// free ends the body and gives its memory back; the bytes take returned are
// invalid after it.
func (b *responseBody) free() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.open = false
	if b.p != 0 {
		procHeapFree.Call(b.heap, 0, b.p)
		b.p, b.n, b.size = 0, 0, 0
	}
}

// failure is why the body is not the response's: a write that found no
// memory for it.
func (b *responseBody) failure() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}
