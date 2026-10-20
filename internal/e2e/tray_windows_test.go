//go:build windows

package e2e

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"unsafe"

	"github.com/egoist/mygo"
)

// TestTrayAdded shows tray icons numbered by their IDs once NewTray is
// done, with or without an image, and where they are as soon as asked,
// each again once made again.
func TestTrayAdded(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	img.Set(0, 0, color.Black)
	var icon bytes.Buffer
	if err := png.Encode(&icon, img); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		withIcon, err := mygo.NewTray(mygo.TrayOptions{ID: "e2e.icon", Icon: icon.Bytes(), ToolTip: "e2e icon"})
		if err != nil {
			t.Fatal(err)
		}
		if b := withIcon.Bounds(); b.Width == 0 || b.Height == 0 {
			t.Errorf("an icon made with an image is not in the notification area: %+v", b)
		}
		blank, err := mygo.NewTray(mygo.TrayOptions{ID: "e2e.blank"})
		if err != nil {
			t.Fatal(err)
		}
		eventually(t, "the blank icon added", func() bool { b := blank.Bounds(); return b.Width > 0 && b.Height > 0 })
		withIcon.Destroy()
		blank.Destroy()
	}
}

// TestTrayFollowsTheTaskbar tells a tray icon that the taskbar turned dark
// or light, as Windows does when the user switches its mode.
func TestTrayFollowsTheTaskbar(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	img.Set(8, 8, color.Black)
	var icon bytes.Buffer
	if err := png.Encode(&icon, img); err != nil {
		t.Fatal(err)
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{ID: "e2e.theme", Icon: icon.Bytes(), IconIsTemplate: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tray.Destroy()
	var told atomic.Bool
	tray.OnAppearanceChange(func() { told.Store(true) })
	dark := tray.Dark()

	light, had := taskbarLight(t)
	t.Cleanup(func() {
		if had {
			setTaskbarLight(t, light)
		} else {
			procRegDeleteKeyValueW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(personalizeKey))), uintptr(unsafe.Pointer(u16("SystemUsesLightTheme"))))
		}
		taskbarModeChanged(t)
	})
	if dark {
		setTaskbarLight(t, 1)
	} else {
		setTaskbarLight(t, 0)
	}
	taskbarModeChanged(t)
	eventually(t, "the icon told the taskbar turned", told.Load)
	if tray.Dark() == dark {
		t.Errorf("Dark is still %v after the taskbar turned", dark)
	}
}

const (
	hkeyCurrentUser = 0x80000001
	personalizeKey  = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
)

var (
	advapi32                  = syscall.NewLazyDLL("advapi32.dll")
	procRegGetValueW          = advapi32.NewProc("RegGetValueW")
	procRegSetKeyValueW       = advapi32.NewProc("RegSetKeyValueW")
	procRegDeleteKeyValueW    = advapi32.NewProc("RegDeleteKeyValueW")
	user32                    = syscall.NewLazyDLL("user32.dll")
	procFindWindowExW         = user32.NewProc("FindWindowExW")
	procGetWindowThreadProcID = user32.NewProc("GetWindowThreadProcessId")
	procSendMessageTimeoutW   = user32.NewProc("SendMessageTimeoutW")
)

func u16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		panic(err)
	}
	return p
}

// taskbarLight reads SystemUsesLightTheme, and whether it is set.
func taskbarLight(t *testing.T) (uint32, bool) {
	t.Helper()
	var v uint32
	size := uint32(4)
	r, _, _ := procRegGetValueW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(personalizeKey))), uintptr(unsafe.Pointer(u16("SystemUsesLightTheme"))),
		0x10, 0, uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&size))) // RRF_RT_REG_DWORD
	return v, r == 0
}

func setTaskbarLight(t *testing.T, v uint32) {
	t.Helper()
	if r, _, _ := procRegSetKeyValueW.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(personalizeKey))), uintptr(unsafe.Pointer(u16("SystemUsesLightTheme"))),
		4, uintptr(unsafe.Pointer(&v)), 4); r != 0 { // REG_DWORD
		t.Fatalf("cannot set SystemUsesLightTheme: error %d", r)
	}
}

// taskbarModeChanged sends the app's own window what Windows broadcasts
// when the user switches the mode: a broadcast would turn the taskbar of
// whoever runs the test too.
func taskbarModeChanged(t *testing.T) {
	t.Helper()
	var h uintptr
	for {
		h, _, _ = procFindWindowExW.Call(0, h, uintptr(unsafe.Pointer(u16("MyGoApp"))), 0)
		if h == 0 {
			t.Fatal("no MyGoApp window in this process")
		}
		var pid uint32
		procGetWindowThreadProcID.Call(h, uintptr(unsafe.Pointer(&pid)))
		if int(pid) == os.Getpid() {
			break
		}
	}
	const wmSettingChange, smtoAbortIfHung = 0x001A, 0x0002
	var res uintptr
	procSendMessageTimeoutW.Call(h, wmSettingChange, 0, uintptr(unsafe.Pointer(u16("ImmersiveColorSet"))), smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&res)))
}
