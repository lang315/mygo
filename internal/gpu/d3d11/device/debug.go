//go:build windows

package device

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// Debug reports whether MYGO_D3D_DEBUG=1: devices then ask for the Direct3D
// debug layer (when the "Graphics Tools" feature of Windows is installed),
// and Report logs what the layer says. Direct3D 11 has no DRED, which is
// Direct3D 12's; Report names the reason a removed device gives instead.
func Debug() bool { return os.Getenv("MYGO_D3D_DEBUG") == "1" }

const (
	createDeviceDebug = 0x2        // D3D11_CREATE_DEVICE_DEBUG
	sdkComponentMiss  = 0x887A002D // DXGI_ERROR_SDK_COMPONENT_MISSING
	infoQueueGetNum   = 8          // ID3D11InfoQueue::GetNumStoredMessages
	infoQueueGetMsg   = 5          // ::GetMessage
	infoQueueClear    = 4          // ::ClearStoredMessages
	deviceRemovedAt   = 39         // ID3D11Device::GetDeviceRemovedReason
)

// IID_ID3D11InfoQueue
var iidInfoQueue = [16]byte{0xb6, 0xdb, 0x43, 0x65, 0x48, 0x1b, 0xf5, 0x42, 0xab, 0x82, 0xe9, 0x7e, 0xc7, 0x43, 0x26, 0xf6}

var missingOnce sync.Once

// noteMissing says, once, that the debug layer could not be had.
func noteMissing() {
	missingOnce.Do(func() {
		log.Print("mygo: MYGO_D3D_DEBUG=1, but the Direct3D debug layer is not installed (add the \"Graphics Tools\" optional feature of Windows): going on without it")
	})
}

// com calls method i of a COM object.
func com(obj uintptr, i int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(*(*unsafe.Pointer)(unsafe.Pointer(&obj)))
	fn := *(*uintptr)(unsafe.Add(*(*unsafe.Pointer)(unsafe.Pointer(&vtbl)), i*int(unsafe.Sizeof(uintptr(0)))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

// severity names D3D11_MESSAGE_SEVERITY.
var severity = [...]string{"corruption", "error", "warning", "info", "message"}

// Report logs the messages the debug layer stored for the device since it
// last did, and what the device says of why it was removed, if it was. It
// does nothing without MYGO_D3D_DEBUG=1 or when the device has no debug
// layer. Call it after Present and when a call failed; it must run on the
// thread that uses the device.
func Report(dev uintptr) {
	if dev == 0 || !Debug() {
		return
	}
	if hr := com(dev, deviceRemovedAt); int32(uint32(hr)) < 0 {
		log.Printf("mygo: d3d11 debug: the device was removed: %#x", uint32(hr))
	}
	var q uintptr
	if int32(uint32(com(dev, 0, uintptr(unsafe.Pointer(&iidInfoQueue)), uintptr(unsafe.Pointer(&q))))) < 0 || q == 0 {
		return
	}
	defer com(q, 2)
	n := uint64(com(q, infoQueueGetNum))
	for i := uint64(0); i < n && i < 50; i++ { // a storm of one message is not worth more
		var size uintptr
		if int32(uint32(com(q, infoQueueGetMsg, uintptr(i), 0, uintptr(unsafe.Pointer(&size))))) < 0 || size == 0 {
			break
		}
		buf := make([]byte, size)
		if int32(uint32(com(q, infoQueueGetMsg, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))))) < 0 {
			break
		}
		log.Print(formatMessage(buf))
	}
	if n > 50 {
		log.Printf("mygo: d3d11 debug: %d more messages left out", n-50)
	}
	com(q, infoQueueClear)
}

// formatMessage reads a D3D11_MESSAGE: Category, Severity, ID (int32 each),
// the description's address and its length (with its NUL); the text follows
// the structure in the same buffer, which the address points into.
func formatMessage(b []byte) string {
	if len(b) < 32 {
		return "mygo: d3d11 debug: a short message"
	}
	sev := *(*int32)(unsafe.Pointer(&b[4]))
	id := *(*int32)(unsafe.Pointer(&b[8]))
	at := *(*uintptr)(unsafe.Pointer(&b[16]))
	length := *(*uintptr)(unsafe.Pointer(&b[24]))
	text := ""
	base := uintptr(unsafe.Pointer(&b[0]))
	if at >= base+32 && at+length <= base+uintptr(len(b)) && length > 0 {
		off := at - base
		text = string(b[off : off+length-1])
	}
	name := "?"
	if sev >= 0 && int(sev) < len(severity) {
		name = severity[sev]
	}
	return fmt.Sprintf("mygo: d3d11 debug: %s %d: %s", name, id, text)
}
