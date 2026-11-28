//go:build !js

package mygo

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// TestRelaunchArgumentsReplaceTheApps: Relaunch's arguments replace those
// the app started with, and reach mygo dev (the rest is package updates').
func TestRelaunchArgumentsReplaceTheApps(t *testing.T) {
	args := []string{"--reason", "it's an update", "a b", `$HOME "quoted" \ back`, "line\nbreak", ""}
	onMain(func() {
		if got := relaunchArguments(); !slices.Equal(got, os.Args[1:]) {
			t.Errorf("without arguments: %q, want the app's %q", got, os.Args[1:])
		}
		App.relaunchArgs = args
		defer func() { App.relaunchArgs = nil }()
		if got := relaunchArguments(); !slices.Equal(got, args) {
			t.Errorf("with arguments: %q", got)
		}
	})
	dev := strings.Split(string(devRelaunchData(args)), "\x00")
	if !slices.Equal(dev[:len(dev)-1], args) {
		t.Errorf("for mygo dev: %q", dev)
	}
}

func TestIsUpdateLeftover(t *testing.T) {
	for name, want := range map[string]bool{
		".app.exe.old-1234":  true,
		".lib.so.old-7":      true,
		".app.exe.old-42":    false, // the running process's
		".notes.old-backup":  false,
		".notes.old-12a":     false,
		".notes.old-":        false,
		".notes.old":         false,
		"notes.old-1234":     false,
		"..old-1234":         false,
		".old-1234":          false,
		".x.old-12345678901": false,
	} {
		if got := isUpdateLeftover(name, 42); got != want {
			t.Errorf("isUpdateLeftover(%q) = %v, want %v", name, got, want)
		}
	}
}
