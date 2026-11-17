//go:build unix || windows

package mygo

import (
	"io"
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
	if err := os.WriteFile(from, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(from, 0o640); err != nil { // the umask may have cleared bits
		t.Fatal(err)
	}
	if err := copyFile(from, to); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(to)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Errorf("mode = %v, want 0640", got)
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

func TestFileOwnedByOther(t *testing.T) {
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

func TestWriteFileSafelyRoutesAnotherUsersFileInPlace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.SameFile does not tell a rename from a write in place there")
	}
	for _, other := range []bool{true, false} {
		old := ownedByOther
		t.Cleanup(func() { ownedByOther = old })
		ownedByOther = func(os.FileInfo) bool { return other }
		path := filepath.Join(t.TempDir(), "f")
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		before, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = writeFileSafely(path, func(w io.Writer) error { _, err := io.WriteString(w, "new"); return err })
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(path); string(got) != "new" {
			t.Errorf("other=%v: content = %q", other, got)
		}
		if same := os.SameFile(before, after); same != other {
			t.Errorf("other=%v: same file = %v, want in place only for another user's file", other, same)
		}
	}
}
