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
// for the next window. The renderers of a process share one device
// (shared.go); when the last releases it, the device and what belongs to it,
// not to a window (shaders, input layout, states), go to a pool of one
// (devices.parked), which the next renderer takes instead of making them. The
// pool is for the usual case, a window that stays open and dialogs that open
// and close in turn, and the app whose only window closes and opens again.
//
// A device goes back only when the renderer let go of everything else it made
// on it (ClearState and Flush ran, its textures, buffers, views and swap
// chain are released), it has not been removed, and it is not a debug device
// (MYGO_D3D_DEBUG=1, whose layer should report what a renderer leaked, as it
// does without the pool). The GPU always comes first, as device.New makes it:
// a pooled device of the GPU is taken only while its adapter is still the
// default one (the one device.New would take), and one of WARP only when the
// GPU cannot make a device, so a window that first got WARP (a remote
// session, a driver that was resetting) finds the GPU again as soon as it can
// (a WARP device that is not taken is released then, once). A device that
// finds the pool full is released. The pool holds a device until the app
// ends, as a window of another renderer would; renderers are made and
// released on the main thread, and the prewarm's goroutine puts its device in
// the pool, so the pool is guarded by devices' mutex.

// Replaced by tests: how a device is made, which adapter is the default, and
// whether the default adapter is a software one.
var (
	newDevice   = device.New
	defaultLUID = defaultAdapterLUID
	// softwareDefault reports whether the default adapter is a software one.
	softwareDefault = defaultAdapterIsSoftware
	devicesMade     atomic.Int32 // devices made, for tests
	shadersMade     atomic.Int32 // sets of shaders made, for tests
)

// complete reports whether the device has everything a renderer needs of it
// (ctx1 may be 0: a device without ID3D11DeviceContext1).
func (d *sharedDevice) complete() bool {
	return d.device != 0 && d.ctx != 0 && d.factory != 0 && d.vs != 0 && d.ps != 0 && d.layout != 0 &&
		d.blend != 0 && d.holeBlend != 0 && d.overBlend != 0 && d.overHoleBlend != 0 && d.raster != 0 && d.samp != 0 && d.passVS != 0 && d.downPS != 0 && d.blurPS != 0
}

// park puts a device nobody holds in the pool, or releases it when it cannot
// go there. devices is locked.
func park(d *sharedDevice) {
	if device.Debug() || !d.complete() || deviceRemoved(d) || devices.parked != nil {
		d.free()
		return
	}
	call(d.ctx, ctxClearState)
	devices.parked = d
}

// takeParked returns the pooled device the next renderer takes, or nil when
// it makes one of its own. devices is locked.
func takeParked() *sharedDevice {
	d := devices.parked
	devices.parked = nil
	switch {
	case d == nil:
		return nil
	case deviceRemoved(d):
		d.free()
		return nil
	case d.warp:
		// Only the GPU's device is kept for its adapter, and WARP for no GPU.
		if dev, ctx, err := newDeviceOnGPU(); err == nil {
			devicesMade.Add(1)
			n, err := newSharedDevice(dev, ctx, false)
			d.free()
			if err != nil {
				return nil
			}
			return n
		}
		return d
	default:
		if luid, ok := defaultLUID(); ok && luid == d.luid {
			return d
		}
		d.free()
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
// them in the background while the first window is created and shown (the
// backend calls it as that window is made): a first frame that took 22 ms in the
// window's own time took 4. On a software adapter (Microsoft's Basic Render
// Driver, which VMs, remote sessions and the CI runners have) it does not
// make them: there making a device is as much work for the CPU as the window's,
// and the two only wait for each other. Nor does it for a debug device
// (MYGO_D3D_DEBUG=1), whose layer should see what a renderer made. The renderer
// that finds a prewarm running waits for it, as making a second device would
// double the work, and a hung driver would hang it just the same.
var prewarm struct {
	sync.Mutex
	done chan struct{}
}

// Prewarm starts, in the background, making the device and shaders of the first
// renderer. It returns at once, and does nothing from the second call on; the
// goroutine looks at the default adapter and makes nothing when it is a software
// one.
func Prewarm() {
	prewarm.Lock()
	defer prewarm.Unlock()
	if prewarm.done != nil {
		return
	}
	done := make(chan struct{})
	prewarm.done = done
	go func() {
		defer close(done)
		if device.Debug() || softwareDefault() {
			return
		}
		dev, ctx, warp, err := newDevice()
		if err != nil {
			return
		}
		devicesMade.Add(1)
		d, err := newSharedDevice(dev, ctx, warp)
		if err != nil {
			return
		}
		devices.Lock()
		defer devices.Unlock()
		if devices.current != nil || devices.parked != nil || device.Debug() {
			d.free()
			return
		}
		devices.parked = d
	}()
}

// waitPrewarm returns once the prewarm, if one started, has put its device in the
// pool (or made none).
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

// defaultAdapterDesc1 describes the adapter device.New takes, the first one
// DXGI lists.
func defaultAdapterDesc1() (d adapterDesc1, ok bool) {
	var factory, adapter uintptr
	if hr, _, _ := procCreateDXGIFactory1.Call(uintptr(unsafe.Pointer(&iidIDXGIFactory1)), uintptr(unsafe.Pointer(&factory))); failed(hr) {
		return d, false
	}
	defer free(&factory)
	if failed(call(factory, dxgiFactory1EnumAdapters1, 0, uintptr(unsafe.Pointer(&adapter)))) {
		return d, false
	}
	defer free(&adapter)
	return d, !failed(call(adapter, dxgiAdapter1GetDesc1, uintptr(unsafe.Pointer(&d))))
}

// defaultAdapterIsSoftware reports whether the default adapter is a software
// one: the flag says so, and Microsoft's Basic Render Driver says it only by its
// vendor (it is a driver of its own kind, and a device on it is not WARP's). It
// is false where that cannot be read.
func defaultAdapterIsSoftware() bool {
	d, ok := defaultAdapterDesc1()
	return ok && (d.Flags&dxgiAdapterFlagSoftware != 0 || d.VendorID == vendorMicrosoft)
}
