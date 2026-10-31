package fsretry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// A file another program holds open, as an antivirus scanning it does,
// renames once it lets go.
func TestRenameWaitsForAFileInUse(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows keeps open files from being renamed")
	}
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a.dll"), filepath.Join(dir, "b.dll")
	if err := os.WriteFile(from, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(from) // without FILE_SHARE_DELETE
	if err != nil {
		t.Fatal(err)
	}
	// The file is let go when Rename first waits, once it has found the
	// file in use: no time passes in the test.
	inUse := 0
	sleep = func(time.Duration) {
		if inUse++; inUse == 1 {
			f.Close()
		}
	}
	t.Cleanup(func() { sleep = time.Sleep })
	if err := Rename(from, to); err != nil {
		t.Fatal(err)
	}
	if inUse == 0 {
		t.Fatal("Rename found the file free: the test checks nothing")
	}
	if _, err := os.Stat(to); err != nil {
		t.Fatal(err)
	}
}
