package ui

import (
	"fmt"
	"testing"
	"time"
)

// The ids actingIDs gathers are the elements the engine is acting on or
// waiting for: a row holding one is built, not linked. Each test puts the
// thing the engine does on an element inside a row of a Table with a
// RowVersion, idles until rows are kept, does it, and compares with the same
// view building every row; a row linked while the engine acts on an element of
// it shows a stale state.

// actingTable is a Table of rows with a RowVersion whose cells are built by
// cell(c, row).
func actingPair(t *testing.T, cell func(c *Context, row int)) *pair {
	t.Helper()
	return actingPairKept(t, true, cell)
}

// actingPairKept is actingPair for cells whose rows are never kept
// (retainable, state.pending): wantKept false asserts that, and the test then
// compares what the engine does there.
func actingPairKept(t *testing.T, wantKept bool, cell func(c *Context, row int)) *pair {
	t.Helper()
	p := newPair(t, 400, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		cols := []TableColumn{{Title: "Name"}, {Title: "Value", Width: 120}}
		return func(c *Context) {
			Table(c, s, cols, 60, func(r, col int) {
				if col == 0 {
					cell(c, r)
				} else {
					Textf(c, "%d", r)
				}
			}).Grow(1)
		}
	})
	p.idle(4)
	if wantKept && p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test shows nothing")
	}
	if !wantKept && p.kept.rt.rowsKept != 0 {
		t.Fatalf("%d rows kept: they were never kept, which the test relies on", p.kept.rt.rowsKept)
	}
	return p
}

// A PopoverMenu on an element of a row opens when the row is clicked with the
// secondary button: the menu drawn belongs to the element, and a row linked
// is not built, so nothing draws the menu and resolveMenu closes it.
func TestRetainActingDrawnMenu(t *testing.T) {
	p := actingPair(t, func(c *Context, row int) {
		Text(c, fmt.Sprintf("acc%d", row)).PopoverMenu(func(m *Menu) {
			m.Item("Open").Chosen()
			m.Item("Delete").Chosen()
		})
	})
	r, ok := p.kept.Find("acc2") // the first cell of row 2
	if !ok {
		t.Fatal("no cell acc2")
	}
	x, y := r.X+r.W/2, r.Y+r.H/2
	p.both(func(tt *Tester) { tt.RightClickAt(x, y) })
	p.compare("menu opened")
	for _, tt := range []*Tester{p.kept, p.built} {
		if !tt.HasText("Open") {
			t.Fatalf("the menu does not show (kept: %v)", tt == p.kept)
		}
	}
	p.idle(3)
	for _, tt := range []*Tester{p.kept, p.built} {
		if !tt.HasText("Open") {
			t.Fatalf("the menu closed by itself (kept: %v)", tt == p.kept)
		}
	}
}

// A ContextMenu of the system: asked on the secondary press, built once by the
// frame, shown by the host, and the item chosen is reported to the element in
// the next frame.
func TestRetainActingContextMenuChosen(t *testing.T) {
	var chosen []string
	p := actingPair(t, func(c *Context, row int) {
		Text(c, fmt.Sprintf("acc%d", row)).ContextMenu(func(m *Menu) {
			if m.Item("Open").Chosen() {
				chosen = append(chosen, fmt.Sprint("open ", row))
			}
		})
	})
	r, _ := p.kept.Find("acc2")
	p.both(func(tt *Tester) { tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2) })
	p.compare("menu asked")
	for _, tt := range []*Tester{p.kept, p.built} {
		if len(tt.Menu()) == 0 {
			t.Fatalf("no system menu (kept: %v)", tt == p.kept)
		}
	}
	p.both(func(tt *Tester) {
		if err := tt.ChooseMenuItem("Open"); err != nil {
			t.Fatal(err)
		}
	})
	p.compare("item chosen")
	p.idle(2)
	if len(chosen) != 2 {
		t.Errorf("the item was reported %d times (once to each Tester), want 2: %v", len(chosen), chosen)
	}
}

// A tooltip shown for an element of a row.
func TestRetainActingTooltip(t *testing.T) {
	p := actingPairKept(t, false, func(c *Context, row int) {
		Text(c, fmt.Sprintf("acc%d", row)).Tooltip(fmt.Sprintf("tip %d", row))
	})
	r, _ := p.kept.Find("acc3")
	p.both(func(tt *Tester) { tt.Move(r.X+r.W/2, r.Y+r.H/2) })
	p.compare("hovered")
	for i := range 20 {
		p.both(func(tt *Tester) { tt.Advance(100 * time.Millisecond) })
		p.compare(fmt.Sprintf("tooltip wait %d", i))
	}
	for _, tt := range []*Tester{p.kept, p.built} {
		if !tt.HasText("tip 3") {
			t.Fatalf("no tooltip (kept: %v)", tt == p.kept)
		}
	}
	p.both(func(tt *Tester) { tt.Move(-5, -5) })
	p.compare("away")
	p.idle(2)
}

// A value dragged over an element of a row that takes it, then dropped.
func TestRetainActingDragOverARowElement(t *testing.T) {
	dropped := 0
	p := newPair(t, 400, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				Box(c).Size(100, 30).Label("Source").Drag("hello")
				List(c, s, 60, func(i int) {
					t := Text(c, fmt.Sprintf("acc%d", i)).Padding(4)
					if _, over := DragOver[string](t); over {
						t.Background(RGB(200, 255, 200))
					}
					if _, ok := Drop[string](t); ok {
						dropped++
					}
				}).Grow(1)
			})
		}
	})
	p.idle(4)
	src, _ := p.kept.Find("Source")
	tgt, _ := p.kept.Find("acc4")
	p.both(func(tt *Tester) {
		tt.Press(src.X+10, src.Y+10)
		tt.Move(src.X+30, src.Y+30)
	})
	p.compare("dragging")
	p.both(func(tt *Tester) { tt.Move(tgt.X+10, tgt.Y+5) })
	p.compare("over a row element")
	p.idle(2)
	p.both(func(tt *Tester) { tt.Release(tgt.X+10, tgt.Y+5) })
	p.compare("dropped")
	if dropped != 2 {
		t.Errorf("dropped %d times (once in each Tester), want 2", dropped)
	}
	p.idle(2)
}

// A text input of a row composing text (IME): the input has the focus.
func TestRetainActingCompositionInARow(t *testing.T) {
	vals := make([]string, 60)
	p := actingPairKept(t, false, func(c *Context, row int) {
		TextInput(c, &vals[row]).Width(100)
	})
	r := Rect{}
	for _, n := range p.kept.Nodes() {
		if n.Role == RoleTextField && n.Bounds.Y > 40 {
			r = n.Bounds
			break
		}
	}
	if r.W == 0 {
		t.Fatal("no text input")
	}
	p.both(func(tt *Tester) { tt.ClickAt(r.X+r.W/2, r.Y+r.H/2) })
	p.compare("focused")
	p.both(func(tt *Tester) { tt.Compose("ni", 2) })
	p.compare("composing")
	p.idle(3)
	p.both(func(tt *Tester) { tt.Compose("你", 1) })
	p.compare("composing again")
}

// A popover opened from an element of a row (a Select's list).
func TestRetainActingPopoverFromARow(t *testing.T) {
	sel := make([]string, 60)
	p := actingPairKept(t, false, func(c *Context, row int) {
		Select(c, &sel[row], []string{"one", "two", "three"}).Width(100)
	})
	var r Rect
	for _, n := range p.kept.Nodes() {
		if n.Bounds.Y > 40 && n.Bounds.W > 0 && n.Role != RoleText {
			r = n.Bounds
			break
		}
	}
	p.both(func(tt *Tester) { tt.ClickAt(r.X+r.W/2, r.Y+r.H/2) })
	p.compare("opened")
	p.idle(3)
	p.both(func(tt *Tester) { tt.Key(0, KeyEscape) })
	p.compare("closed")
	p.idle(2)
}
