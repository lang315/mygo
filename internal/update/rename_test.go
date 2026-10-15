package update

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
	go func() { time.Sleep(300 * time.Millisecond); f.Close() }()
	if err := Rename(from, to); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(to); err != nil {
		t.Fatal(err)
	}
}
