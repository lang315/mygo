//go:build !js

package mygo

import (
	"os"
	"path/filepath"
	"testing"
)

// explainedTempDir is t.TempDir, and when the directory cannot be removed at
// the end (Windows: a file still open, or being written), it logs what is left
// in it, with the sizes, so that the failure names the file.
func explainedTempDir(t *testing.T) string {
	t.Helper()
	var dir string
	// Registered before t.TempDir's own cleanup, so that it runs after it.
	t.Cleanup(func() {
		if dir == "" {
			return
		}
		if _, err := os.Stat(dir); err == nil {
			logLeft(t, dir, 0)
		}
	})
	dir = t.TempDir()
	return dir
}

// logLeft logs the entries under path, to a few levels.
func logLeft(t *testing.T, path string, depth int) {
	es, _ := os.ReadDir(path)
	for _, e := range es {
		size := int64(-1)
		if info, err := e.Info(); err == nil {
			size = info.Size()
		}
		t.Logf("left behind: %s (%d bytes, dir %v)", filepath.Join(path, e.Name()), size, e.IsDir())
		if e.IsDir() && depth < 4 {
			logLeft(t, filepath.Join(path, e.Name()), depth+1)
		}
	}
}
