package ui

import (
	"fmt"
	"testing"
)

// replayRowTwins runs view in two testers, one replaying rows and one painting
// them all (noReplay), and after each step compares their ops and their
// nodes' flags. It reports how many rows replayed in steps after the first.
func replayRowTwins(t *testing.T, view func(c *Context), steps int, step func(i int)) int {
	t.Helper()
	rep := NewTester(view, 300, 400)
	fresh := NewTester(view, 300, 400)
	fresh.rt.noReplay = true
	replayed := 0
	for i := range steps {
		step(i)
		for rr := range 3 {
			rep.Frame()
			fresh.Frame()
			if i > 0 {
				replayed += rep.rt.replayed
			}
			if d := sameScenesAcross(&rep.rt.scene, &fresh.rt.scene); d != "" {
				t.Fatalf("step %d.%d (%d replayed): %s", i, rr, rep.rt.replayed, d)
			}
			got, want := map[uint64]uint32{}, map[uint64]uint32{}
			walkRow(rep.rt.c.root, func(e *node) { got[e.id] |= e.flags })
			walkRow(fresh.rt.c.root, func(e *node) { want[e.id] |= e.flags })
			for id, f := range want {
				if got[id] != f {
					t.Fatalf("step %d.%d: element %x flags %b replaying, %b painting", i, rr, id, got[id], f)
				}
			}
		}
	}
	return replayed
}

// A column's width changes and nothing in the rows does: the cells of the
// other column move while every row keeps its box, ids and flags. Only the
// boxes in the record tell the row from the one recorded.
func TestReplayRowColumnWidthMovesCells(t *testing.T) {
	widths := []float32{80, 140, 60, 140}
	var w float32
	var list ListState
	view := func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A", Width: w}, {Title: "B"}}, 100, func(r, col int) {
			Textf(c, "r%d c%d", r, col)
		}).Grow(1)
	}
	n := replayRowTwins(t, view, len(widths), func(i int) { w = widths[i] })
	t.Logf("%d rows replayed", n)
}

// The same, through the user's widths (ListState.Columns), with a column
// sharing the room the other leaves.
func TestReplayRowSharedColumnMovesCells(t *testing.T) {
	widths := []float32{100, 180, 50}
	var w float32
	var list ListState
	view := func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}, {Title: "B", Width: w}, {Title: "C"}}, 100, func(r, col int) {
			Textf(c, "r%d c%d", r, col)
		}).Grow(1)
	}
	n := replayRowTwins(t, view, len(widths), func(i int) { w = widths[i] })
	t.Logf("%d rows replayed", n)
}

// A row gains and loses an element, of no size, inside a cell.
func TestReplayRowRowGainsAnElement(t *testing.T) {
	var extra bool
	var list ListState
	view := func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}}, 100, func(r, col int) {
			Row(c).Children(func() {
				Textf(c, "row %d", r)
				if extra && r == 3 {
					Box(c)
				}
			})
		}).Grow(1)
	}
	n := replayRowTwins(t, view, 4, func(i int) { extra = i%2 == 1 })
	t.Logf("%d rows replayed", n)
}

// Rows holding a list, whose rows replay inside the outer row (paintRow
// within paintRow): the outer row never records, the inner ones do.
func TestReplayRowRowHoldingAList(t *testing.T) {
	var outer ListState
	inner := make([]ListState, 100)
	label := "x"
	view := func(c *Context) {
		List(c, &outer, 100, func(r int) {
			Column(c).Height(60).Children(func() {
				Textf(c, "outer %d", r)
				List(c, &inner[r], 10, func(k int) { Textf(c, "%s %d.%d", label, r, k) }).Height(40)
			})
		}).Grow(1)
	}
	n := replayRowTwins(t, view, 4, func(i int) {
		if i == 2 {
			label = "y"
		}
	})
	t.Logf("%d rows replayed", n)
	if n == 0 {
		t.Error("no row replayed")
	}
}

// Scrolls by whole pixels shift rows, then a column's width changes while
// shifted.
func TestReplayRowShiftThenColumnWidth(t *testing.T) {
	var w float32 = 80
	var list ListState
	view := func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A", Width: w}, {Title: "B"}}, 100, func(r, col int) {
			Textf(c, "%s", fmt.Sprint("r", r, " c", col))
		}).Grow(1)
	}
	rep := NewTester(view, 300, 400)
	fresh := NewTester(view, 300, 400)
	fresh.rt.noReplay = true
	cmp := func(what string) {
		t.Helper()
		if d := sameScenesAcross(&rep.rt.scene, &fresh.rt.scene); d != "" {
			t.Fatalf("%s (%d replayed): %s", what, rep.rt.replayed, d)
		}
	}
	for range 3 {
		rep.Frame()
		fresh.Frame()
	}
	shifted := 0
	for k := range 4 {
		rep.Scroll(100, 100, 0, 1)
		fresh.Scroll(100, 100, 0, 1)
		rep.Frame()
		fresh.Frame()
		shifted += rep.rt.replayed
		cmp(fmt.Sprint("scroll ", k))
		if k == 1 {
			w = 150
		}
	}
	t.Logf("%d rows replayed", shifted)
}

// A held row whose kept layout leaves an element elsewhere than the record
// (a stale kept record, spoiled here as TestRetainLayoutCheckFindsAStaleRecord
// does) keeps its ids, flags and box: the record's boxes alone keep it from
// replaying where the element no longer is. The frame's ops equal painting
// every row.
func TestReplayRowReplayFollowsTheBoxesTheLayoutLeft(t *testing.T) {
	for _, spoil := range []func(r *rowKeep){
		func(r *rowKeep) { r.nodes[len(r.nodes)-1].x += 3 },
		func(r *rowKeep) { r.nodes[len(r.nodes)-1].y += 3 },
		func(r *rowKeep) { r.nodes[1].x += 3 },
		func(r *rowKeep) { r.nodes[0].w += 3 },
	} {
		replayRowStaleRecord(t, spoil)
	}
}

func replayRowStaleRecord(t *testing.T, spoil func(r *rowKeep)) {
	k := newKeptList(50)
	tt := NewTester(k.mk(), 300, 400)
	for range 4 {
		tt.Frame()
	}
	done := false
	for _, st := range tt.rt.states {
		if r := st.krec; r != nil && r.lay.at != 0 && r.lay.cok && len(r.lay.hits) > 0 && len(r.lay.labels) > 0 {
			spoil(r)
			done = true
			break
		}
	}
	if !done {
		t.Fatal("no row held")
	}
	tt.Frame()
	t.Logf("replayed %d", tt.rt.replayed)
	got := snapScene(&tt.rt.scene)
	w, h, sc := tt.h.size()
	tt.rt.noReplay = true
	tt.rt.paint(tt.rt.c.root, w, h, sc)
	tt.rt.noReplay = false
	if d := sameScenes(&got, &tt.rt.scene); d != "" {
		t.Fatalf("replayed frame differs from painting every row: %s", d)
	}
}
