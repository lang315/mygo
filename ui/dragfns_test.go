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
		f.scroll != nil && f.paint != nil && f.dataDragOver != nil && f.dataDrop != nil
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
	d := &platform.DataDragEvent{}
	rt.dragMoveLinked(1, 1)
	rt.dragScrollLinked()
	rt.paintDragLinked(nil, 300, 200)
	if rt.pressDraggerLinked(nil, nil) != 0 || rt.dragEndLinked() || rt.dragCancelLinked() || rt.dataDragOverLinked(1, 1, d) || rt.dataDropLinked(1, 1, d) {
		t.Error("a wrapper with no function set did something")
	}
	if dragLinked() {
		t.Error("calling the wrappers linked the drag")
	}
}
