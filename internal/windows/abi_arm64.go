//go:build windows

package windows

import "strconv"

// On ARM64, structs of up to 16 bytes are passed in registers.
func putBounds(controller uintptr, r rect) {
	comCall(controller, ctlPutBounds,
		uintptr(uint32(r.Left))|uintptr(uint32(r.Top))<<32,
		uintptr(uint32(r.Right))|uintptr(uint32(r.Bottom))<<32)
}

// Go's Windows ARM64 calls do not set the floating point registers, which
// ICoreWebView2Controller::put_ZoomFactor reads: zoom with CSS instead. The
// page is told the zoom (__mygoCssZoom), so that the bridge measures the
// content in the CSS pixels Go multiplies by it: a zoomed root measures in
// zoomed pixels.
func putZoom(w *window, f float64) {
	z := strconv.FormatFloat(f, 'f', -1, 64)
	w.Eval("document.documentElement.style.zoom=" + z + ";window.__mygoCssZoom=" + z)
}

const zoomByScript = true
