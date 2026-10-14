package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// waitFake waits for the open fake window titled title.
func waitFake(t *testing.T, title string) *fake.Window {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		for _, fw := range fb.Windows() {
			if fw.Title() == title && !fw.IsClosed() {
				return fw
			}
		}
	}
	t.Fatalf("no open window titled %q", title)
	return nil
}

// press presses the element labeled label in fw's view, as a screen
// reader does, and draws the frame after.
func press(t *testing.T, fw *fake.Window, label string) {
	t.Helper()
	var labels []string
	found := false
	mygo.RunOnMain(func() {
		s := fw.FakeSurface()
		s.Send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
		s.Frame()
		tree, _ := s.Accessibility()
		if tree == nil {
			return
		}
		for _, n := range tree.Nodes {
			if n.Label == label {
				s.Send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
				s.Frame()
				found = true
				return
			}
			labels = append(labels, n.Label)
		}
	})
	if !found {
		t.Fatalf("no %q among %q", label, labels)
	}
}

// typeText types s into the element with the focus in fw's view.
func typeText(fw *fake.Window, s string) {
	mygo.RunOnMain(func() {
		fw.FakeSurface().Frame() // a first frame applies Focus
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
