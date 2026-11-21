// Package noloader holds the test that no Go file of the repository loads a
// native library (purego.Dlopen, syscall's and x/sys/windows' LoadLibrary,
// LoadDLL, NewLazyDLL) outside the directories loadAllowed names. The
// system's libraries are loaded where they are used; a library of an app or
// a plugin is named in its mygo-plugin.json and loaded with package ffi,
// which the CLI builds and ships it for (AGENTS.md). The test reads the
// source whatever its build constraints, so it runs on any host.
package noloader
