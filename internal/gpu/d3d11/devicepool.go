//go:build windows

package d3d11

import (
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/internal/gpu/d3d11/device"
)

// Releasing the last reference of a Direct3D device takes tens of
// milliseconds (35 to 70 on CI runners), which closing a window waited for
// on the main thread, and a device with its shaders takes several to make
// for the next window. So a renderer that is released leaves its device
// and what belongs to the device alone, not to a window (shaders, input
// layout, states), in a pool of one, which the next renderer takes instead
// of making them. The pool is for the usual case, a window that stays
// open and dialogs that open and close in turn.
//
// A kit goes back only when the renderer let go of everything else it made
// on the device (ClearState and Flush ran, its textures, buffers, views
// and swap chain are released), the device has not been removed, and it
// is not a debug device (MYGO_D3D_DEBUG=1, whose layer should report what
// a renderer leaked, as it does without the pool). device.New always takes
// the default adapter, so a kit is the device it would make. A kit that
// finds the pool full is released as before. The pool holds a device until
// the app ends, as a window of another renderer would; the pool needs a
// mutex only for tests, as renderers are made and released on the main
// thread.
type kit struct {
	device, ctx, ctx1, factory uintptr
	warp                       bool

	vs, ps, layout         uintptr
	blend, holeBlend       uintptr
	raster, samp           uintptr
	passVS, downPS, blurPS uintptr
}

var pool struct {
	sync.Mutex
	kit *kit
}

// Replaced by tests: how a device is made, and whether one was removed.
var (
	newDevice   = device.New
	deviceGone  = deviceRemoved
	devicesMade atomic.Int32 // devices made, for tests
	shadersMade atomic.Int32 // sets of shaders made, for tests
)

const devGetDeviceRemovedReason = 39 // ID3D11Device

// deviceRemoved reports whether the device was removed or reset, which
// makes every call on it fail.
func deviceRemoved(dev uintptr) bool { return failed(call(dev, devGetDeviceRemovedReason)) }

// takeKit returns the part of the renderer that belongs to its device, and
// takes it away from the renderer.
func (r *Renderer) takeKit() *kit {
	k := &kit{r.device, r.ctx, r.ctx1, r.factory, r.warp,
		r.vs, r.ps, r.layout, r.blend, r.holeBlend, r.raster, r.samp, r.passVS, r.downPS, r.blurPS}
	r.device, r.ctx, r.ctx1, r.factory = 0, 0, 0, 0
	r.vs, r.ps, r.layout, r.blend, r.holeBlend, r.raster, r.samp, r.passVS, r.downPS, r.blurPS = 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	return k
}

// complete reports whether the kit has everything a renderer needs of it
// (ctx1 may be 0: a device without ID3D11DeviceContext1).
func (k *kit) complete() bool {
	return k.device != 0 && k.ctx != 0 && k.factory != 0 && k.vs != 0 && k.ps != 0 && k.layout != 0 &&
		k.blend != 0 && k.holeBlend != 0 && k.raster != 0 && k.samp != 0 && k.passVS != 0 && k.downPS != 0 && k.blurPS != 0
}

func (k *kit) free() {
	for _, p := range []*uintptr{&k.samp, &k.raster, &k.holeBlend, &k.blend, &k.layout,
		&k.blurPS, &k.downPS, &k.passVS, &k.ps, &k.vs, &k.factory, &k.ctx1, &k.ctx, &k.device} {
		free(p)
	}
}

// adopt takes the kit's device and its objects for the renderer.
func (r *Renderer) adopt(k *kit) {
	r.device, r.ctx, r.ctx1, r.factory, r.warp = k.device, k.ctx, k.ctx1, k.factory, k.warp
	r.vs, r.ps, r.layout, r.blend, r.holeBlend, r.raster, r.samp = k.vs, k.ps, k.layout, k.blend, k.holeBlend, k.raster, k.samp
	r.passVS, r.downPS, r.blurPS = k.passVS, k.downPS, k.blurPS
}

// pooled returns the kit in the pool, or nil.
func pooled() *kit {
	pool.Lock()
	defer pool.Unlock()
	k := pool.kit
	pool.kit = nil
	if k != nil && deviceGone(k.device) {
		k.free()
		return nil
	}
	return k
}

// keep puts a kit in the pool, or releases it when it cannot go there.
func keep(k *kit) {
	if device.Debug() || !k.complete() || deviceGone(k.device) {
		k.free()
		return
	}
	pool.Lock()
	if pool.kit == nil {
		pool.kit = k
		k = nil
	}
	pool.Unlock()
	if k != nil {
		k.free()
	}
}
