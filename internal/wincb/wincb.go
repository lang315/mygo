//go:build windows

// Package wincb makes the callbacks that native code calls on Windows.
//
// syscall.NewCallback's entry on Windows ARM64 (runtime·callbackasm1) saves
// the integer registers a C function must keep (R19-R28) but not the
// floating point ones (F8-F15, callee-saved too), while the Go code of the
// callback is free to use them. Native code that keeps a double in one of
// them across a call to the callback, as UI Automation's core does with the
// rectangles it clips by, finds it changed. On Windows ARM64 New therefore
// returns an entry that keeps F8-F15; elsewhere it is syscall.NewCallback.
package wincb
