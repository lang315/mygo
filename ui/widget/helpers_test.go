package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

// waitFake waits for the open fake window titled title, once the form that
// made it is shown: the fake lists a window as soon as NewWindow makes it,
// before show registers OnClose on it, and an Escape or a close request in
// that gap closes a window that should ask or stay open.
func waitFake(t *testing.T, title string) *fake.Window {
	t.Helper()
	var open *fake.Window
	wait.Until(t, "an open window titled "+title, func() bool {
		if !formShown(title) {
			return false
		}
		for _, fw := range fb.Windows() {
			if fw.Title() == title && !fw.IsClosed() {
				open = fw
				return true
			}
		}
		return false
	})
	return open
}

// formShown reports whether a shown form is titled title: show records it
// after it registered the window's handlers.
func formShown(title string) bool {
	mu.Lock()
	defer mu.Unlock()
	for _, f := range shown {
		if f.title == title {
			return true
		}
	}
	return false
}

// press presses the element labeled label in fw's view, as a screen
// reader does, and draws the frame after.
func press(t *testing.T, fw *fake.Window, label string) {
	t.Helper()
	var labels []string
	for _, n := range accessNodes(fw) {
		if n.Label == label {
			mygo.RunOnMain(func() {
				s := fw.FakeSurface()
				s.Send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
				s.Frame()
			})
			return
		}
		labels = append(labels, n.Label)
	}
	t.Fatalf("no %q among %q", label, labels)
}

// labels returns the labels of fw's accessibility tree, in tree order.
func labels(fw *fake.Window) []string {
	var out []string
	for _, n := range accessNodes(fw) {
		out = append(out, n.Label)
	}
	return out
}

// accessNodes turns accessibility on in fw's view, draws a frame and
// returns its accessibility tree's nodes.
func accessNodes(fw *fake.Window) (nodes []platform.AccessNode) {
	mygo.RunOnMain(func() {
		s := fw.FakeSurface()
		s.Send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
		s.Frame()
		if tree, _ := s.Accessibility(); tree != nil {
			nodes = tree.Nodes
		}
	})
	return nodes
}

func frame(fw *fake.Window) { mygo.RunOnMain(func() { fw.FakeSurface().Frame() }) }

// typeText types s into the element with the focus in fw's view.
func typeText(fw *fake.Window, s string) {
	frame(fw)
	mygo.RunOnMain(func() {
		fw.FakeSurface().Send(platform.SurfaceEvent{Kind: platform.TextInput, Text: s})
		fw.FakeSurface().Frame()
	})
}

// key presses k in fw's view.
func key(fw *fake.Window, k platform.Key) {
	mygo.RunOnMain(func() {
		fw.FakeSurface().Send(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: k})
		fw.FakeSurface().Send(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: k})
		fw.FakeSurface().Frame()
	})
}

// answer is what an input dialog returned.
type answer[T any] struct {
	v  T
	ok bool
}

// runAsync runs fn on a goroutine of its own, as a worker does.
func runAsync[T any](fn func() T) <-chan T {
	ch := make(chan T, 1)
	go func() { ch <- fn() }()
	return ch
}

// testPNG returns a 4x4 PNG of one color.
func testPNG(t *testing.T, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
