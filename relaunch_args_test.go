//go:build !js

package mygo

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/updatehook"
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

func TestCleanUpdateLeftoversKeepsLookAlikesOfABuildWithoutUpdates(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("darwin's updater leaves no entries of this shape to clean")
	}
	dir := t.TempDir()
	foreign := filepath.Join(dir, ".notes.old-backup")
	if err := os.WriteFile(foreign, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(dir, ".folder.old-1234", "nested")
	if err := os.MkdirAll(folder, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "sentinel.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	updates, exe := updatehook.Updates.Load(), startExe
	defer func() { updatehook.Updates.Store(updates); startExe = exe }()
	updatehook.Updates.Store(false)
	startExe = filepath.Join(dir, "app")
	cleanUpdateLeftovers()
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("the unrelated file beside the executable was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(folder, "sentinel.txt")); err != nil {
		t.Fatalf("the unrelated folder beside the executable was removed: %v", err)
	}
}

func TestCleanUpdateLeftoversRemovesTheLeftoversOfABuildWithUpdates(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("darwin's updater leaves no entries of this shape to clean")
	}
	dir := t.TempDir()
	write := func(name string) string {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	gone := []string{
		write(".notes.old-1234"),
		filepath.Dir(write(filepath.Join(".lib.old-99", "nested", "sentinel.txt"))),
	}
	kept := []string{
		write(".notes.old-backup"),
		write(".notes.old-" + strconv.Itoa(os.Getpid())), // this process's own, mid-update
		write("notes.old-1234"),
		write(".notes.old-"),
	}
	updates, exe := updatehook.Updates.Load(), startExe
	defer func() { updatehook.Updates.Store(updates); startExe = exe }()
	updatehook.Updates.Store(true)
	startExe = filepath.Join(dir, "app")
	cleanUpdateLeftovers()
	for _, p := range gone {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("the update leftover %s stayed: %v", filepath.Base(p), err)
		}
	}
	for _, p := range kept {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s was removed: %v", filepath.Base(p), err)
		}
	}
}
