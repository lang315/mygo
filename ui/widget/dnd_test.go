package widget

import (
	"fmt"
	"sync"
	"testing"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// dragTo drags the node from onto the node to with the pointer: press, move
// in steps, release.
func dragTo(tt *Tester, from, to *Node) {
	a, b := from.Bounds(), to.Bounds()
	ax, ay := a.X+a.W/2, a.Y+a.H/2
	bx, by := b.X+b.W/2, b.Y+b.H/2
	tt.Press(ax, ay)
	for i := 1; i <= 4; i++ {
		f := float32(i) / 4
		tt.Move(ax+(bx-ax)*f, ay+(by-ay)*f)
		tt.Frame()
	}
	tt.Release(bx, by)
	tt.Settle()
}

type evlog struct {
	mu sync.Mutex
	s  []string
}

func (l *evlog) add(f string, a ...any) {
	l.mu.Lock()
	l.s = append(l.s, fmt.Sprintf(f, a...))
	l.mu.Unlock()
}
func (l *evlog) get() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.s...)
}

func TestDragSourceDropsOnATargetOfItsTypeInOrder(t *testing.T) {
	l := &evlog{}
	src := Label("src").DragSource("payload")
	dst := Label("dst").Padding(8)
	OnDrop(dst, func(v string) { l.add("drop %s", v) })
	OnDragOver(dst, func(over bool, v string) {
		if over {
			l.add("over true %s", v)
		} else {
			l.add("over false")
		}
	})
	tt := NewTester(t, VBox(40).Add(src, dst), 300, 300)
	dragTo(tt, tt.Find(ByLabel("src")), tt.Find(ByLabel("dst")))
	got := l.get()
	want := []string{"over true payload", "over false", "drop payload"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("events %q, want %q: hover ends before the drop", got, want)
	}
}

func TestDragOfAnotherTypeIsNotDroppableAndFiresNothing(t *testing.T) {
	l := &evlog{}
	src := Label("src").DragSource("a string")
	dst := Label("dst").Padding(8)
	OnDrop(dst, func(v int) { l.add("drop %d", v) })
	OnDragOver(dst, func(over bool, v int) { l.add("over %v", over) })
	tt := NewTester(t, VBox(40).Add(src, dst), 300, 300)
	dragTo(tt, tt.Find(ByLabel("src")), tt.Find(ByLabel("dst")))
	if got := l.get(); len(got) != 0 {
		t.Fatalf("a string drag reached an int target: %q", got)
	}
}

func TestInnerTargetOfTheMatchingTypeWins(t *testing.T) {
	l := &evlog{}
	src := Label("src").DragSource(42)
	inner := Label("inner").Padding(8)
	OnDrop(inner, func(v int) { l.add("inner %d", v) })
	outer := VBox(4).Add(inner).Padding(16)
	OnDrop(outer, func(v string) { l.add("outer %s", v) })
	tt := NewTester(t, VBox(40).Add(src, outer), 300, 400)
	dragTo(tt, tt.Find(ByLabel("src")), tt.Find(ByLabel("inner")))
	if got := l.get(); fmt.Sprint(got) != "[inner 42]" {
		t.Fatalf("events %q", got)
	}
}

func TestEscapeMidDragEndsHoverAndDropsNothing(t *testing.T) {
	l := &evlog{}
	src := Label("src").DragSource("p")
	dst := Label("dst").Padding(8)
	OnDrop(dst, func(v string) { l.add("drop") })
	OnDragOver(dst, func(over bool, v string) { l.add("over %v", over) })
	tt := NewTester(t, VBox(40).Add(src, dst), 300, 300)
	a, b := tt.Find(ByLabel("src")).Bounds(), tt.Find(ByLabel("dst")).Bounds()
	tt.Press(a.X+a.W/2, a.Y+a.H/2)
	for i := 1; i <= 4; i++ {
		tt.Move(a.X+a.W/2, a.Y+a.H/2+float32(i)*(b.Y-a.Y)/4)
		tt.Frame()
	}
	tt.Key(0, ui.KeyEscape)
	tt.Release(b.X+b.W/2, b.Y+b.H/2)
	tt.Settle()
	got := l.get()
	if len(got) == 0 || got[len(got)-1] != "over false" {
		t.Fatalf("events %q: after Escape the last hover must be over=false", got)
	}
	for _, e := range got {
		if e == "drop" {
			t.Fatalf("events %q: a cancelled drag dropped", got)
		}
	}
}

func TestReleaseOutsideAnyTargetEndsHover(t *testing.T) {
	l := &evlog{}
	src := Label("src").DragSource("p")
	dst := Label("dst").Padding(8)
	OnDragOver(dst, func(over bool, v string) { l.add("over %v", over) })
	OnDrop(dst, func(v string) { l.add("drop") })
	tt := NewTester(t, VBox(40).Add(src, dst), 300, 300)
	a, b := tt.Find(ByLabel("src")).Bounds(), tt.Find(ByLabel("dst")).Bounds()
	tt.Press(a.X+a.W/2, a.Y+a.H/2)
	for i := 1; i <= 3; i++ {
		tt.Move(a.X+a.W/2, a.Y+a.H/2+float32(i)*(b.Y-a.Y)/3)
		tt.Frame()
	}
	tt.Move(290, 290) // off the target
	tt.Frame()
	tt.Release(290, 290)
	tt.Settle()
	got := l.get()
	if len(got) == 0 || got[len(got)-1] != "over false" {
		t.Fatalf("events %q", got)
	}
}

func TestAPlainPressStillClicksAButtonThatIsADragSourceAndADragDoesNot(t *testing.T) {
	clicks := 0
	btn := Button("go").OnClick(func() { clicks++ }).DragSource("p")
	dst := Label("dst").Padding(8)
	OnDrop(dst, func(string) {})
	tt := NewTester(t, VBox(40).Add(btn, dst), 300, 300)
	tt.Find(ByLabel("go")).Click()
	if clicks != 1 {
		t.Fatalf("a plain press clicked %d times", clicks)
	}
	dragTo(tt, tt.Find(ByLabel("go")), tt.Find(ByLabel("dst")))
	if clicks != 1 {
		t.Fatalf("a drag clicked the button: %d clicks", clicks)
	}
}

func TestDragDataSnapshotsOnTheMainThreadOffTheLock(t *testing.T) {
	name := TextBox().SetText("Ada")
	called := make(chan string, 4)
	src := Label("src").DragData(func() transfer.Data {
		called <- name.Text() // a getter locks mu: the provider must run off it
		return transfer.TextData(name.Text())
	})
	dst := Label("dst").Padding(8)
	dst.OnDropData([]transfer.Format{transfer.Text}, func(d transfer.Data) {})
	tt := NewTester(t, VBox(40).Add(src, name, dst), 300, 300)
	dragTo(tt, tt.Find(ByLabel("src")), tt.Find(ByLabel("dst")))
	select {
	case got := <-called:
		if got != "Ada" {
			t.Fatalf("snapshot saw %q", got)
		}
	default:
		t.Fatal("the data provider never ran")
	}
}

func TestOnDropFilesGetsTheFilesDroppedFromTheOS(t *testing.T) {
	got := make(chan []string, 2)
	zone := Label("zone").Padding(20)
	zone.OnDropFiles(func(paths []string) { got <- paths })
	tt := NewTester(t, zone, 300, 200)
	b := tt.Find(ByLabel("zone")).Bounds()
	tt.Tester.DropFiles(b.X+b.W/2, b.Y+b.H/2, "/tmp/a.txt", "/tmp/b.txt")
	tt.Settle()
	paths := received(t, got)
	if len(paths) != 2 || paths[0] != "/tmp/a.txt" {
		t.Fatalf("paths %q", paths)
	}
}

func TestDropTargetsCanBeRemoved(t *testing.T) {
	l := &evlog{}
	src := Label("src").DragSource("p")
	dst := Label("dst").Padding(8)
	OnDrop(dst, func(v string) { l.add("drop") })
	tt := NewTester(t, VBox(40).Add(src, dst), 300, 300)
	OnDrop[string](dst, nil)
	tt.Settle()
	dragTo(tt, tt.Find(ByLabel("src")), tt.Find(ByLabel("dst")))
	if got := l.get(); len(got) != 0 {
		t.Fatalf("a removed target dropped: %q", got)
	}
}
