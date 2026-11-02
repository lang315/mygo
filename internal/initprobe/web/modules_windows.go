//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// listModules prints the DLLs the process has loaded, to stderr.
func listModules(when string) {
	k := syscall.NewLazyDLL("kernel32.dll")
	enum := k.NewProc("K32EnumProcessModules")
	name := k.NewProc("K32GetModuleFileNameExW")
	self, _ := syscall.GetCurrentProcess()
	var mods [1024]syscall.Handle
	var need uint32
	enum.Call(uintptr(self), uintptr(unsafe.Pointer(&mods[0])), uintptr(len(mods))*unsafe.Sizeof(mods[0]), uintptr(unsafe.Pointer(&need)))
	n := int(need) / int(unsafe.Sizeof(mods[0]))
	var names []string
	for _, m := range mods[:n] {
		var buf [260]uint16
		name.Call(uintptr(self), uintptr(m), uintptr(unsafe.Pointer(&buf[0])), 260)
		p := syscall.UTF16ToString(buf[:])
		if i := strings.LastIndexByte(p, '\\'); i >= 0 {
			p = p[i+1:]
		}
		names = append(names, strings.ToLower(p))
	}
	fmt.Fprintf(os.Stderr, "MODULES %s (%d): %s\n", when, n, strings.Join(names, " "))
}
