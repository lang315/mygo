//go:build windows

package d3d11

import (
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

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
// a renderer leaked, as it does without the pool). The GPU always comes
// first, as device.New makes it: a kit of the GPU is taken only while its
// adapter is still the default one (the one device.New would take), and a
// kit of WARP only when the GPU cannot make a device, so a window that
// first got WARP (a remote session, a driver that was resetting) or
// retried the GPU after rendering failed finds it again as soon as it can
// (a kit of WARP that is not taken is released then, once). A kit that
// finds the pool full is released as before. The pool holds a device until
// the app ends, as a window of another renderer would; the pool needs a
// mutex only for tests, as renderers are made and released on the main
// thread.
type kit struct {
	device, ctx, ctx1, factory uintptr
	warp                       bool
	luid                       uint64 // of the device's adapter

	vs, ps, layout         uintptr
	blend, holeBlend       uintptr
	raster, samp           uintptr
	passVS, downPS, blurPS uintptr
}

var pool struct {
	sync.Mutex
	kit *kit
}

// Replaced by tests: how a device is made, with and without the software
// rasterizer, which adapter is the default, and whether a device was removed.
var (
	newDevice   = device.New
	newHardware = device.NewHardware
	defaultLUID = defaultAdapterLUID
	deviceGone  = deviceRemoved
	// softwareDefault reports whether the default adapter is a software one.
	softwareDefault = defaultAdapterIsSoftware
	devicesMade     atomic.Int32 // devices made, for tests
	shadersMade     atomic.Int32 // sets of shaders made, for tests
)

const devGetDeviceRemovedReason = 39 // ID3D11Device

// deviceRemoved reports whether the device was removed or reset, which
// makes every call on it fail.
func deviceRemoved(dev uintptr) bool { return failed(call(dev, devGetDeviceRemovedReason)) }

// takeKit returns the part of the renderer that belongs to its device, and
// takes it away from the renderer.
func (r *Renderer) takeKit() *kit {
	k := &kit{r.device, r.ctx, r.ctx1, r.factory, r.warp, r.luid,
		r.vs, r.ps, r.layout, r.blend, r.holeBlend, r.raster, r.samp, r.passVS, r.downPS, r.blurPS}
	r.device, r.ctx, r.ctx1, r.factory, r.luid = 0, 0, 0, 0, 0
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
	r.device, r.ctx, r.ctx1, r.factory, r.warp, r.luid = k.device, k.ctx, k.ctx1, k.factory, k.warp, k.luid
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

// reusable returns the pooled kit the renderer takes, or nil when it makes
// a device of its own, which it then may have started to make: r.device is
// set for the GPU's device made while a kit of WARP waited (the kit is
// released), so that it is not made twice. Only the GPU's device is kept
// for its adapter, and WARP for no GPU.
func reusable(r *Renderer) *kit {
	waitPrewarm()
	k := pooled()
	switch {
	case k == nil:
		return nil
	case k.warp:
		if dev, ctx, err := newHardware(); err == nil {
			k.free()
			r.device, r.ctx, r.warp = dev, ctx, false
			return nil
		}
		return k
	default:
		if luid, ok := defaultLUID(); ok && luid == k.luid {
			return k
		}
		k.free()
		return nil
	}
}

// adapterDesc is DXGI_ADAPTER_DESC.
type adapterDesc struct {
	Description                                 [128]uint16
	VendorID, DeviceID, SubSysID, Revision      uint32
	DedicatedVideoMemory, DedicatedSystemMemory uintptr
	SharedSystemMemory                          uintptr
	LUID                                        uint64
}

const dxgiAdapterGetDesc = 8 // IDXGIAdapter

// adapterLUID returns the locally unique ID of an adapter.
func adapterLUID(adapter uintptr) (uint64, bool) {
	var d adapterDesc
	if failed(call(adapter, dxgiAdapterGetDesc, uintptr(unsafe.Pointer(&d)))) {
		return 0, false
	}
	return d.LUID, true
}

var procCreateDXGIFactory1 = syscall.NewLazyDLL(device.SystemDir() + `\dxgi.dll`).NewProc("CreateDXGIFactory1")

var iidIDXGIFactory1 = guid{0x770aae78, 0xf26f, 0x4dba, [8]byte{0xa8, 0x29, 0x25, 0x3c, 0x83, 0xd1, 0xb3, 0x87}}

const dxgiFactoryEnumAdapters = 7 // IDXGIFactory

// defaultAdapterLUID returns the LUID of the adapter device.New takes, the
// first one DXGI lists.
func defaultAdapterLUID() (uint64, bool) {
	var factory, adapter uintptr
	if hr, _, _ := procCreateDXGIFactory1.Call(uintptr(unsafe.Pointer(&iidIDXGIFactory1)), uintptr(unsafe.Pointer(&factory))); failed(hr) {
		return 0, false
	}
	defer free(&factory)
	if failed(call(factory, dxgiFactoryEnumAdapters, 0, uintptr(unsafe.Pointer(&adapter)))) {
		return 0, false
	}
	defer free(&adapter)
	return adapterLUID(adapter)
}

// A window of native UI waits, on the main thread, for a device and its shaders
// that the first of them makes (11 ms for the device on a GPU, as many for the
// shaders and states, which windows after it find in the pool). Prewarm makes
// them in the background while the first window is created and shown, which
// the app starts as it runs (ui has the backend call it): a first frame that took
// 22 ms in the window's own time took 4. On a software adapter (Microsoft's
// Basic Render Driver, which VMs, remote sessions and the CI runners have) it
// does not start: there making a device is as much work for the CPU as the
// window's, and the two only wait for each other. The renderer that finds a
// prewarm running waits for it, as making a second device would double the work.
var prewarm struct {
	sync.Mutex
	done chan struct{}
}

// Prewarm starts making the device and shaders of the first renderer, unless
// the default adapter is a software one. It does nothing the second time, and
// does not wait.
func Prewarm() {
	prewarm.Lock()
	defer prewarm.Unlock()
	if prewarm.done != nil || softwareDefault() {
		return
	}
	done := make(chan struct{})
	prewarm.done = done
	go func() {
		defer close(done)
		r := &Renderer{}
		var err error
		if r.device, r.ctx, r.warp, err = newDevice(); err != nil {
			return
		}
		devicesMade.Add(1)
		if err := r.initDevice(); err != nil {
			r.takeKit().free()
			return
		}
		keep(r.takeKit())
	}()
}

// waitPrewarm returns once the prewarm, if one started, has put its kit in the
// pool (or failed to).
func waitPrewarm() {
	prewarm.Lock()
	done := prewarm.done
	prewarm.Unlock()
	if done != nil {
		<-done
	}
}

// DXGI_ADAPTER_DESC1.
type adapterDesc1 struct {
	Description                                 [128]uint16
	VendorID, DeviceID, SubSysID, Revision      uint32
	DedicatedVideoMemory, DedicatedSystemMemory uintptr
	SharedSystemMemory                          uintptr
	LUID                                        uint64
	Flags                                       uint32
}

const (
	dxgiFactory1EnumAdapters1 = 12 // IDXGIFactory1
	dxgiAdapter1GetDesc1      = 10 // IDXGIAdapter1
	dxgiAdapterFlagSoftware   = 2  // DXGI_ADAPTER_FLAG_SOFTWARE
	vendorMicrosoft           = 0x1414
)

// defaultAdapterIsSoftware reports whether the adapter device.New takes, the
// first one DXGI lists, is a software one: the flag says so, and Microsoft's
// Basic Render Driver says it only by its vendor (it is a driver of its own
// kind, and a device on it is not WARP's). It is false where that cannot be read.
func defaultAdapterIsSoftware() bool {
	var factory, adapter uintptr
	if hr, _, _ := procCreateDXGIFactory1.Call(uintptr(unsafe.Pointer(&iidIDXGIFactory1)), uintptr(unsafe.Pointer(&factory))); failed(hr) {
		return false
	}
	defer free(&factory)
	if failed(call(factory, dxgiFactory1EnumAdapters1, 0, uintptr(unsafe.Pointer(&adapter)))) {
		return false
	}
	defer free(&adapter)
	var d adapterDesc1
	if failed(call(adapter, dxgiAdapter1GetDesc1, uintptr(unsafe.Pointer(&d)))) {
		return false
	}
	return d.Flags&dxgiAdapterFlagSoftware != 0 || d.VendorID == vendorMicrosoft
}
