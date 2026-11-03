package update

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The installer writes the InstallLocation as the user gave it with /D=, and
// a short name (C:\Users\RUNNER~1\...) is how a runner's temporary folder
// reads; the running executable's folder is in long names. They are one
// folder, and an update must still record its version for it (TestUpdaterSmoke
// failed on a runner: Settings > Apps kept the old version).
func TestSameInstallDirShortName(t *testing.T) {
	long := filepath.Join(t.TempDir(), "A Long Installation Folder")
	other := filepath.Join(t.TempDir(), "Another Long Installation Folder")
	for _, d := range []string{long, other} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	short := shortPath(t, long)
	if short == long {
		t.Skip("the volume makes no short names")
	}
	if !SameInstallDir(short, long) || !SameInstallDir(long, short) {
		t.Errorf("SameInstallDir(%q, %q) = false, want true: one folder", short, long)
	}
	if SameInstallDir(short, other) {
		t.Errorf("SameInstallDir(%q, %q) = true, want false: two folders", short, other)
	}
}

func shortPath(t *testing.T, p string) string {
	t.Helper()
	in, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]uint16, syscall.MAX_PATH)
	n, err := syscall.GetShortPathName(in, &buf[0], uint32(len(buf)))
	if err != nil {
		t.Fatal(err)
	}
	return syscall.UTF16ToString(buf[:n])
}
