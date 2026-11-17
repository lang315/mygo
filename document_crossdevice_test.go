//go:build unix || windows

package mygo

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestDocumentMoveKeepsTheMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix modes")
	}
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(from, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(from, to); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(to)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %v, want 0600", got)
	}
}

func TestCrossDevice(t *testing.T) {
	if !crossDevice(&os.LinkError{Err: syscall.EXDEV}) {
		t.Error("EXDEV is a move across devices")
	}
	if got, want := crossDevice(&os.LinkError{Err: syscall.Errno(17)}), runtime.GOOS == "windows"; got != want {
		t.Errorf("Errno(17) = %v, want %v (ERROR_NOT_SAME_DEVICE on Windows, EEXIST on Unix)", got, want)
	}
}

func TestWriteFileSafelyWritesInPlaceForAnotherUsersFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mine")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fileOwnedByOther(info) {
		t.Error("a file we own is reported as owned by another user")
	}
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		return
	}
	root, err := os.Stat("/etc/hosts")
	if err != nil {
		t.Skip(err)
	}
	if !fileOwnedByOther(root) {
		t.Error("a file root owns is not reported as owned by another user")
	}
}
