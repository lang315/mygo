//go:build windows

package d3d11

import (
	"errors"
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
	gone, hardware, adapter := deviceGone, newHardware, defaultLUID
	t.Cleanup(func() { deviceGone, newHardware, defaultLUID = gone, hardware, adapter })
}

// pooledKit returns the kit a released renderer left in the pool, which
// the test then edits as if it came from another kind of device.
func pooledKit(t *testing.T) *kit {
	t.Helper()
	newRendererOrSkip(t).Release()
	pool.Lock()
	defer pool.Unlock()
	if pool.kit == nil {
		t.Fatal("the device did not go to the pool")
	}
	return pool.kit
}

func setKit(k *kit, warp bool, luid uint64) {
	pool.Lock()
	k.warp, k.luid = warp, luid
	pool.Unlock()
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
	for i := range 4 {
		r := newRendererOrSkip(t)
		// The adapter's LUID is read from the device, and is the default
		// adapter's for one the GPU made (a runner without a GPU makes WARP
		// or the basic render driver, whose LUID is not the default's: only
		// its being read is checked there). A wrong slot or layout would turn
		// the reuse of a GPU's device off without a word.
		if r.luid == 0 {
			t.Errorf("round %d: the device's adapter has no LUID", i)
		} else if def, ok := defaultLUID(); !r.warp && (!ok || def != r.luid) {
			t.Errorf("round %d: LUID %#x of the device's adapter, %#x (%v) of the default adapter", i, r.luid, def, ok)
		}
		r.Release()
	}
	if n := devicesMade.Load() - devices; n != 1 {
		t.Errorf("%d devices made for 4 renderers one after the other, want 1", n)
	}
	if n := shadersMade.Load() - shaders; n != 1 {
		t.Errorf("%d sets of shaders made for 4 renderers one after the other, want 1", n)
	}
}

// A renderer that takes the device of another draws as one with a device of
// its own. The references held on the pooled device stay the same round
// after round: they count what the device is held by in public, the swap
// chain and the composition objects among others, so a renderer that
// leaked one of those, as it could not before the device stayed alive,
// would add to them. The children of the device that hold it internally
// (textures, views, buffers) do not show in the count, and that the
// device goes back only after everything is released, not before, is not
// checked here.
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

// A renderer made while a kit of WARP is in the pool, and the GPU can make a
// device, takes the GPU's, and the kit of WARP is released.
func TestDevicePoolPrefersTheGPU(t *testing.T) {
	emptyPool(t)
	k := pooledKit(t)
	setKit(k, true, 0)
	hardwareTried := 0
	newHardware = func() (uintptr, uintptr, error) {
		hardwareTried++
		dev, ctx, _, err := newDevice() // stands for the GPU's device
		return dev, ctx, err
	}
	devices := devicesMade.Load()
	r := newRendererOrSkip(t)
	if hardwareTried != 1 {
		t.Errorf("the GPU was tried %d times, want 1", hardwareTried)
	}
	if n := devicesMade.Load() - devices; n != 1 {
		t.Errorf("%d devices made, want the GPU's (1)", n)
	}
	if r.warp {
		t.Error("the renderer has WARP, though the GPU made a device")
	}
	pool.Lock()
	left := pool.kit
	pool.Unlock()
	if left != nil {
		t.Error("the kit of WARP is still in the pool")
	}
	r.Release()
}

// Without a GPU, a renderer takes the kit of WARP from the pool.
func TestDevicePoolTakesWARPWhenThereIsNoGPU(t *testing.T) {
	emptyPool(t)
	k := pooledKit(t)
	setKit(k, true, 0)
	newHardware = func() (uintptr, uintptr, error) { return 0, 0, errors.New("no GPU") }
	devices := devicesMade.Load()
	r := newRendererOrSkip(t)
	if n := devicesMade.Load() - devices; n != 0 {
		t.Errorf("%d devices made, want the pooled WARP's", n)
	}
	r.Release()
}

// A kit of the GPU is taken while its adapter is the default one.
func TestDevicePoolKeepsTheGPUWhileItIsTheDefault(t *testing.T) {
	emptyPool(t)
	for _, tc := range []struct {
		name string
		luid uint64
		ok   bool
		made int32
	}{{"the default", 7, true, 0}, {"another is the default", 8, true, 1}, {"the default is unknown", 0, false, 1}} {
		k := pooledKit(t)
		setKit(k, false, 7)
		defaultLUID = func() (uint64, bool) { return tc.luid, tc.ok }
		devices := devicesMade.Load()
		r := newRendererOrSkip(t)
		if n := devicesMade.Load() - devices; n != tc.made {
			t.Errorf("%s: %d devices made, want %d", tc.name, n, tc.made)
		}
		r.Release()
		emptyPoolNow()
	}
}

func emptyPoolNow() {
	pool.Lock()
	k := pool.kit
	pool.kit = nil
	pool.Unlock()
	if k != nil {
		k.free()
	}
}

// A renderer made for a window again, as after a failure of its frames,
// makes its swap chain with the device that stayed, and draws.
func TestDevicePoolSameWindowAgain(t *testing.T) {
	emptyPool(t)
	s := gputest.BorderedCardScene(false)
	hwnd := hiddenWindow(t, s.Width, s.Height)
	for i := range 3 {
		r, err := New(hwnd)
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		if err := r.draw(s); err != nil {
			t.Fatalf("round %d: %v", i, err)
		}
		pix, stride := r.readBack(t)
		gputest.CompareStraightEdges(t, "d3d11-same-window-again", pix, stride, s)
		r.Release()
	}
}
