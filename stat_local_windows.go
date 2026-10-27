package mygo

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var procGetDriveType = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDriveTypeW")

// driveRemote is DRIVE_REMOTE of GetDriveType.
const driveRemote = 4

// localVolume reports whether the file at path is on a volume whose file
// identities (volume serial and file index) can be trusted to name a file
// for as long as it exists, which network volumes don't promise: false when
// it is on one, and when the system doesn't say.
func localVolume(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	root, err := syscall.UTF16PtrFromString(filepath.VolumeName(abs) + `\`)
	if err != nil {
		return false
	}
	kind, _, _ := procGetDriveType.Call(uintptr(unsafe.Pointer(root)))
	return kind != driveRemote
}
