//go:build windows

package d3d11

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
)

// emptyPool releases the kit in the pool, which another test left, and has
// the test restore what it replaces.
func emptyPool(t *testing.T) {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	pool.Lock()
	k := pool.kit
	pool.kit = nil
	pool.Unlock()
	if k != nil {
		k.free()
	}
	gone := deviceGone
	t.Cleanup(func() { deviceGone = gone })
}

func newRendererOrSkip(t *testing.T) *Renderer {
	t.Helper()
	r, err := New(hiddenWindow(t, 64, 64))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	return r
}

// Renderers made and released one after the other make one device and one
// set of shaders.
func TestDevicePoolReusesTheDevice(t *testing.T) {
	emptyPool(t)
	devices, shaders := devicesMade.Load(), shadersMade.Load()
	for range 4 {
		newRendererOrSkip(t).Release()
	}
	if n := devicesMade.Load() - devices; n != 1 {
		t.Errorf("%d devices made for 4 renderers one after the other, want 1", n)
	}
	if n := shadersMade.Load() - shaders; n != 1 {
		t.Errorf("%d sets of shaders made for 4 renderers one after the other, want 1", n)
	}
}

// A renderer that takes the device of another draws as one with a device of
// its own, and leaves nothing of its own on the device: what the device is
// held by, as it goes back to the pool, stays the same.
func TestDevicePoolKeepsNothingOfAWindow(t *testing.T) {
	emptyPool(t)
	s := gputest.BorderedCardScene(false)
	var refs []uintptr
	for i := range 4 {
		r, err := New(hiddenWindow(t, s.Width, s.Height))
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		pix, stride := r.readBack(t)
		gputest.CompareStraightEdges(t, "d3d11-pooled-device", pix, stride, s)
		r.Release()
		pool.Lock()
		k := pool.kit
		pool.Unlock()
		if k == nil {
			t.Fatalf("round %d: the device did not go to the pool", i)
		}
		call(k.device, 1) // AddRef
		refs = append(refs, call(k.device, release))
	}
	for _, n := range refs[2:] {
		if n != refs[1] {
			t.Errorf("references of the pooled device after each round: %v, want the same from the second on", refs)
			break
		}
	}
}

// A device that was removed makes a new one, in the pool or as it is released.
func TestDevicePoolDropsARemovedDevice(t *testing.T) {
	emptyPool(t)
	devices := devicesMade.Load()
	r := newRendererOrSkip(t)
	deviceGone = func(uintptr) bool { return true }
	r.Release() // removed as it is released: not kept
	pool.Lock()
	kept := pool.kit != nil
	pool.Unlock()
	if kept {
		t.Error("a removed device went to the pool")
	}
	deviceGone = func(uintptr) bool { return false }
	newRendererOrSkip(t).Release() // kept
	deviceGone = func(uintptr) bool { return true }
	newRendererOrSkip(t).Release() // removed while in the pool: made anew, and not kept
	if n := devicesMade.Load() - devices; n != 3 {
		t.Errorf("%d devices made, want 3 (none was reused)", n)
	}
}

// The pool holds one device: of two renderers released, one device goes.
func TestDevicePoolHoldsOne(t *testing.T) {
	emptyPool(t)
	devices := devicesMade.Load()
	a, b := newRendererOrSkip(t), newRendererOrSkip(t)
	a.Release()
	b.Release()
	newRendererOrSkip(t).Release() // takes the one in the pool
	if n := devicesMade.Load() - devices; n != 2 {
		t.Errorf("%d devices made for two renderers alive at once and one after them, want 2", n)
	}
}
