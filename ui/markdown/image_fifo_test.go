//go:build unix

package markdown

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// TestFileImagesRefusesPipe checks that a named pipe in the folder is
// refused rather than opened, which blocks until a writer comes.
func TestFileImagesRefusesPipe(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "doc")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pipe := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(pipe, 0o644); err != nil {
		t.Fatal(err)
	}
	// A load blocked opening the pipe goes on once a writer opens it.
	t.Cleanup(func() {
		if f, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			f.Close()
		}
	})
	done := make(chan error, 1)
	go func() {
		_, err := FileImages(dir)("pipe")
		done <- err
	}()
	if err := wait.For(t, "FileImages of a pipe", done); err == nil {
		t.Error("a pipe loaded")
	}
}
