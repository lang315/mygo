package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
)

// onOpen answers the fake's open dialogs with fn until the test ends.
func onOpen(t *testing.T, fn func(o *platform.OpenDialogOptions) []string) {
	mygo.RunOnMain(func() { fb.OnOpenDialog = fn })
	t.Cleanup(func() { mygo.RunOnMain(func() { fb.OnOpenDialog = nil }) })
}

// onSave answers the fake's save dialogs with fn until the test ends.
func onSave(t *testing.T, fn func(o *platform.SaveDialogOptions) string) {
	mygo.RunOnMain(func() { fb.OnSaveDialog = fn })
	t.Cleanup(func() { mygo.RunOnMain(func() { fb.OnSaveDialog = nil }) })
}

func TestOpenFileAndFiles(t *testing.T) {
	var seen platform.OpenDialogOptions
	onOpen(t, func(o *platform.OpenDialogOptions) []string { seen = *o; return []string{"/a.png", "/b.png"} })
	if got := OpenFile(nil, "Pick", Filter{Name: "Images", Extensions: []string{"png"}}); got != "/a.png" {
		t.Fatalf("OpenFile %q", got)
	}
	if seen.Title != "Pick" || seen.Multiple || seen.OpenDirectories || len(seen.Filters) != 1 {
		t.Fatalf("options %+v", seen)
	}
	if got := OpenFiles(nil, "Pick"); !slices.Equal(got, []string{"/a.png", "/b.png"}) || !seen.Multiple {
		t.Fatalf("OpenFiles %q, Multiple %v", got, seen.Multiple)
	}
}

func TestOpenFileCancel(t *testing.T) {
	onOpen(t, func(*platform.OpenDialogOptions) []string { return nil })
	if got := OpenFile(nil, "Pick"); got != "" {
		t.Fatalf("cancel returned %q", got)
	}
}

func TestSelectFolder(t *testing.T) {
	var seen platform.OpenDialogOptions
	onOpen(t, func(o *platform.OpenDialogOptions) []string { seen = *o; return []string{"/tmp"} })
	if got := SelectFolder(nil, "Where", "/home"); got != "/tmp" || !seen.OpenDirectories {
		t.Fatalf("got %q, Directory %v", got, seen.OpenDirectories)
	}
}

func TestSaveFile(t *testing.T) {
	var seen platform.SaveDialogOptions
	onSave(t, func(o *platform.SaveDialogOptions) string { seen = *o; return "/out.csv" })
	if got := SaveFile(nil, "Export", "accounts.csv"); got != "/out.csv" {
		t.Fatalf("SaveFile %q", got)
	}
	if seen.Title != "Export" {
		t.Fatalf("options %+v", seen)
	}
}

// A file dialog in a handler runs the queued handlers.
func TestFileDialogInAHandlerPumps(t *testing.T) {
	pumped := make(chan struct{})
	onOpen(t, func(*platform.OpenDialogOptions) []string {
		// The fake answers on the main thread; it waits for a handler
		// that runs only if the dialog's caller pumps the queue.
		Post(func() { close(pumped) })
		<-pumped
		return []string{"/x"}
	})
	var got string
	Update(func() { got = OpenFile(nil, "Pick") })
	if got != "/x" {
		t.Fatalf("got %q", got)
	}
}
