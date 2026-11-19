package tsgen

import (
	"os"
	"path/filepath"
	"testing"
)

// -trimpath records "<import path>/<file>", which is no path on disk.
func TestSourcesTrimmedPath(t *testing.T) {
	s := newSources()
	trimmed := "github.com/egoist/mygo/internal/tsgen/source.go"
	s.note(trimmed)
	s.note("example.invalid/nowhere/x.go")
	s.resolvePackages(nil)

	got := s.onDisk(trimmed)
	if got == "" {
		t.Fatal("a trimmed path of this module did not resolve")
	}
	if want, _ := filepath.Abs("source.go"); got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Error(err)
	}
	if got := s.onDisk("example.invalid/nowhere/x.go"); got != "" {
		t.Errorf("a path of no known module resolved to %q", got)
	}
	if fd := s.funcDecl("example.invalid/nowhere/x.go", 1, "F"); fd != nil {
		t.Error("a path of no known module found a declaration")
	}
}
