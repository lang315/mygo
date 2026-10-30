package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/transfer"
)

func dragLinked() bool { return dragHooks.Load() != nil }

// dragFnsSet reports whether linkDrag published all the functions.
func dragFnsSet() bool {
	f := dragHooks.Load()
	return f != nil && f.move != nil && f.pressDragger != nil && f.end != nil && f.cancel != nil &&
		f.paint != nil && f.dataTargetAt != nil && f.dropData != nil
}

func clearDragFns() {
	dragHooks.Store(nil)
	dragOnce = sync.Once{} // linked again by the test
}

// An app that makes no source and no target of a drag does not link the
// code that follows a drag: the engine reaches it through functions that
// the API of drags sets. The test starts the test binary again, running
// nothing but this test, which presses, moves, releases, presses Escape and
// is sent a native transfer over a view with a button, and asks whether they
// were set at the start and at the end, and that none of it was taken.
func TestNoDragIsLinkedBeforeADragAPI(t *testing.T) {
	if os.Getenv("MYGO_DRAGFNS_CHILD") == "1" {
		start := dragLinked()
		clicks := 0
		tt := coreNewTester(func(c *context) {
			if coreButton(c, "a button").Clicked() {
				clicks++
			}
		}, 300, 200)
		tt.Frame()
		b, _ := tt.Find("a button")
		tt.Press(b.X+5, b.Y+5)
		tt.Move(b.X+30, b.Y+20)
		tt.Release(b.X+30, b.Y+20)
		tt.Key(0, KeyEscape)
		d := &platform.DataDragEvent{Offer: transfer.Offer{Formats: []transfer.Format{transfer.Text}, Operations: transfer.Copy, Suggested: transfer.Copy}}
		over := tt.send2(platform.SurfaceEvent{Kind: platform.DataDragOver, X: 10, Y: 10, Drag: d})
		dropped := tt.send2(platform.SurfaceEvent{Kind: platform.DataDrop, X: 10, Y: 10, Drag: d})
		tt.Image()
		fmt.Printf("drag linked at start: %v, after pointer, Escape and native transfer: %v (clicks %d, taken %v %v)\n", start, dragLinked(), clicks, over, dropped)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNoDragIsLinkedBeforeADragAPI$", "-test.count=1")
	cmd.Env = append(os.Environ(), "MYGO_DRAGFNS_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the test binary failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "drag linked at start: false, after pointer, Escape and native transfer: false (clicks 1, taken false false)\n") {
		t.Errorf("the drag is linked into an app that makes no source and no target, or the events were taken:\n%s", out)
	}
}

// Each API that makes a source or a target of a drag links the engine's
// side of it.
func TestDragAPIsLinkTheDrag(t *testing.T) {
	data := transfer.New(transfer.NewItem(transfer.Bytes(transfer.Text, []byte("x"))))
	opts := transfer.DropOptions{Formats: []transfer.Format{transfer.Text}, Operations: transfer.Copy}
	for name, make := range map[string]func(c *context){
		"Drag":         func(c *context) { coreBox(c).Size(50, 50).Drag("v") },
		"dragFrom":     func(c *context) { coreBox(c).Size(50, 50).dragFrom(func() any { return "v" }) },
		"DragData":     func(c *context) { coreBox(c).Size(50, 50).DragData(data) },
		"DragDataFrom": func(c *context) { coreBox(c).Size(50, 50).DragDataFrom(func() transfer.Data { return data }) },
		"Drop":         func(c *context) { coreDrop[string](coreBox(c).Size(50, 50)) },
		"DragOver":     func(c *context) { coreDragOver[string](coreBox(c).Size(50, 50)) },
		"DragHover":    func(c *context) { coreDragHover[string](coreBox(c).Size(50, 50)) },
		"DropData":     func(c *context) { coreDropData(coreBox(c).Size(50, 50), opts) },
		"DataDragOver": func(c *context) { coreDataDragOver(coreBox(c).Size(50, 50), opts) },
		"list reorder": func(c *context) {
			s := &ListState{Reorder: func(from []int, to int) {}}
			coreList(c, s, 3, func(i int) { coreText(c, fmt.Sprint("row ", i)) })
		},
	} {
		t.Run(name, func(t *testing.T) {
			clearDragFns()
			tt := coreNewTester(func(c *context) { make(c) }, 300, 200)
			tt.Frame()
			if !dragFnsSet() {
				t.Error("the API did not link the drag")
			}
		})
	}
}

// With no function set, the engine does what it does when nothing is
// dragged, and nothing panics.
func TestUnlinkedDragWrappersDoNothing(t *testing.T) {
	clearDragFns()
	tt := coreNewTester(func(c *context) { coreText(c, "plain") }, 300, 200)
	tt.Frame()
	rt := tt.rt
	d := &platform.DataDragEvent{Operation: transfer.Copy, Formats: []transfer.Format{transfer.Text}}
	rt.dragMoveLinked(1, 1)
	rt.paintDragLinked(nil, 300, 200)
	if rt.pressDraggerLinked(nil, nil) != 0 || rt.dragEndLinked() || rt.dragCancelLinked() || rt.dropDataLinked(nil, 1, 1, d) {
		t.Error("a wrapper with no function set did something")
	}
	if rt.dataTargetAtLinked(1, 1, d) != nil || d.Operation != transfer.None || d.Formats != nil {
		t.Errorf("no target is made, yet one took the transfer or the offer was not refused: operation %v, formats %v", d.Operation, d.Formats)
	}
	if dragLinked() {
		t.Error("calling the wrappers linked the drag")
	}
}

// An app with no drag API whose scroll view holds file drop zones
// (DroppedFiles) scrolls it while files are dragged from another app to its
// edge, so that a zone out of view can be reached, linked or not: the engine
// follows a native transfer over its window and scrolls under it whether or
// not a target exists. Dropping, or leaving, ends it.
func TestFileDragEdgeScrollsWithAndWithoutDragAPI(t *testing.T) {
	for _, linked := range []bool{false, true} {
		name := "unlinked"
		if linked {
			name = "linked"
		}
		t.Run(name, func(t *testing.T) {
			clearDragFns()
			if linked {
				linkDrag()
			}
			var scroll ScrollState
			tt := coreNewTester(func(c *context) {
				coreScroll(c).Fill().TrackScroll(&scroll).Children(func() {
					for range 20 {
						coreBox(c).Size(200, 40).DroppedFiles()
					}
				})
			}, 200, 200)
			tt.Frame()
			d := &platform.DataDragEvent{HasFiles: true, Offer: transfer.Offer{Formats: []transfer.Format{transfer.FileList, transfer.URIList}, Operations: transfer.Copy}}
			tt.rt.event(platform.SurfaceEvent{Kind: platform.DataDragOver, X: 50, Y: 199, Drag: d})
			tt.rt.event(platform.SurfaceEvent{Kind: platform.FileDragOver, X: 50, Y: 199})
			before := scroll.Y
			for range 5 {
				tt.rt.runFrame()
			}
			if scroll.Y <= before {
				t.Fatalf("files dragged to the bottom edge did not scroll: %v -> %v", before, scroll.Y)
			}
			// The drop ends the transfer: the view stops.
			tt.rt.event(platform.SurfaceEvent{Kind: platform.DataDrop, X: 50, Y: 199, Drag: d})
			if tt.rt.incoming != nil {
				t.Fatal("the transfer is still followed after the drop")
			}
			after := scroll.Y
			for range 5 {
				tt.rt.runFrame()
			}
			if scroll.Y != after {
				t.Errorf("the view went on scrolling after the drop: %v -> %v", after, scroll.Y)
			}
		})
	}
}
