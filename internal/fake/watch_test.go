package fake

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A test polling for the watch of a document that opens in the background
// calls LastFileWatch before any watch was created.
func TestLastFileWatchBeforeAnyWatch(t *testing.T) {
	b := New()
	if w := b.LastFileWatch(); w != nil {
		t.Fatalf("LastFileWatch of a new backend = %v, want nil", w)
	}
	first, err := b.NewFileWatch(func(platform.FileWatchNotice) {})
	if err != nil {
		t.Fatal(err)
	}
	if b.LastFileWatch() != first {
		t.Error("LastFileWatch is not the watch created")
	}
}
