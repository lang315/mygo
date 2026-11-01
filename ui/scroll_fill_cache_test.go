package ui

import (
	"fmt"
	"testing"
	"time"
)

// fillCaseState is what the scripted frames change.
type fillCaseState struct {
	lines        int
	fill, nested bool
	pad, border  float32
	minH         float32
	wrap, both   bool
}

func fillCaseView(s *fillCaseState) func(c *context) {
	return func(c *context) {
		coreColumn(c).Fill().Children(func() {
			sc := coreScroll(c)
			if s.both {
				sc = coreScrollBoth(c)
			}
			sc.Grow(1).Padding(s.pad)
			if s.border > 0 {
				sc.Border(s.border, RGB(0, 0, 0))
			}
			if s.fill {
				sc.FillViewport()
			}
			sc.Children(func() {
				coreColumn(c).Fill().Children(func() {
					for i := range s.lines {
						coreText(c, fmt.Sprintf("line %d", i))
					}
					if s.wrap {
						coreText(c, "a long paragraph of words that wraps at a narrow width and so its height depends on the width of the scroller")
					}
					if s.minH > 0 {
						coreBox(c).MinHeight(s.minH)
					}
					if s.nested {
						in := coreScroll(c).Grow(1).FillViewport()
						in.Children(func() {
							coreColumn(c).Fill().Children(func() {
								coreText(c, "inner")
								coreSpacer(c)
								coreButton(c, "Inner")
							})
						})
					}
					coreSpacer(c)
					coreButton(c, "Apply")
				})
			})
		})
	}
}

// The layout cache lays a FillViewport scroller out as laying out anew does,
// frame after frame, as its content, its flag, its padding and the window's
// size change (the random model of TestLayoutCacheMatches never sets it).
func TestFillViewportCacheMatchesFresh(t *testing.T) {
	s := &fillCaseState{lines: 2, fill: true}
	start := time.Unix(1e9, 0)
	cached := lcTester(fillCaseView(s), 300, 300, start)
	fresh := lcTester(fillCaseView(s), 300, 300, start)
	fresh.rt.noLayoutCache = true
	w, h := float32(300), float32(300)
	steps := []struct {
		what string
		do   func()
	}{
		{"unchanged", func() {}},
		{"taller window", func() { h = 500 }},
		{"shorter window", func() { h = 300 }},
		{"long content", func() { s.lines = 60 }},
		{"short content", func() { s.lines = 2 }},
		{"fill off", func() { s.fill = false }},
		{"fill on", func() { s.fill = true }},
		{"padding", func() { s.pad = 20 }},
		{"border", func() { s.border = 3 }},
		{"taller window, padded", func() { h = 600 }},
		{"wrapping text", func() { s.wrap = true }},
		{"narrow window", func() { w = 120 }},
		{"wide window", func() { w = 300 }},
		{"min height child, short", func() { s.minH = 100 }},
		{"min height child, over", func() { s.minH = 900 }},
		{"min height gone", func() { s.minH = 0 }},
		{"nested", func() { s.nested = true }},
		{"nested, shorter window", func() { h = 250 }},
		{"nested, long", func() { s.lines = 40 }},
		{"nested, short", func() { s.lines = 1 }},
		{"unchanged again", func() {}},
		{"scrolls both ways", func() { s.both = true }},
		{"right to left", func() {
			cached.SetDirection(RightToLeft)
			fresh.SetDirection(RightToLeft)
		}},
		{"right to left, long", func() { s.lines = 50 }},
		{"right to left, short", func() { s.lines = 2 }},
		{"right to left, taller", func() { h = 450 }},
		{"vertical only, right to left", func() { s.both = false }},
	}
	for i, st := range steps {
		st.do()
		for _, tt := range []*Tester{cached, fresh} {
			tt.h.w, tt.h.h = w, h
		}
		cached.Frame()
		fresh.Frame()
		if d := sameBoxes(cached.rt.c.root, fresh.rt.c.root, "root"); d != "" {
			t.Fatalf("step %d (%s): %s", i, st.what, d)
		}
		// A short content fills the viewport: Apply ends at its bottom.
		if s.fill && s.lines < 10 && s.minH < 300 {
			b := coreBottomOf(cached, "Apply")
			want := h - s.pad - s.border
			if b < want-1 || b > want+1 {
				t.Errorf("step %d (%s): Apply ends at %v, want %v", i, st.what, b, want)
			}
		}
	}
}

func coreBottomOf(tt *Tester, label string) float32 {
	for _, n := range tt.Nodes() {
		if n.Label == label && n.Role == RoleButton {
			return n.Bounds.Y + n.Bounds.H
		}
	}
	return -1
}

func coreTopOfText(tt *Tester, s string) float32 {
	r, ok := tt.Find(s)
	if !ok || r.H == 0 { // out of view
		return -1
	}
	return r.Y
}

// A FillViewport content scrolled to its end that shrinks below the viewport
// shows from its top again, as a plain scroller does.
func TestFillViewportScrollClampsWhenContentShrinks(t *testing.T) {
	for _, fill := range []bool{false, true} {
		t.Run(fmt.Sprint("fill=", fill), func(t *testing.T) {
			s := &fillCaseState{lines: 60, fill: fill}
			tt := coreNewTester(fillCaseView(s), 300, 300)
			tt.Frame()
			for range 20 {
				tt.Scroll(100, 100, 0, 500)
				tt.Frame()
			}
			if y := coreTopOfText(tt, "line 0"); y >= 0 {
				t.Fatalf("line 0 at y=%v: the wheel did not scroll", y)
			}
			s.lines = 2
			tt.Frame()
			tt.Frame()
			if y := coreTopOfText(tt, "line 0"); y != 0 {
				t.Fatalf("line 0 at y=%v after the content shrank", y)
			}
		})
	}
}

// Nested FillViewport scrollers: the elements laid out in a frame that changes
// one text, against the same tree without the flag. The second pass is the
// documented cost (2x at depth 1); the layout cache keeps nesting from
// compounding it (2^depth): measured 17/8, 28/11, 42/14, 59/17, 79/20 at depths
// 1 to 5, about (depth+1)x. Informational, fails only on super-linear growth.
func TestFillViewportNestedFillViewportWork(t *testing.T) {
	for depth := 1; depth <= 5; depth++ {
		plain, fill := nestedFillCount(depth, false), nestedFillCount(depth, true)
		t.Logf("depth %d: %d laid out with FillViewport, %d without", depth, fill, plain)
		if fill > uint32(depth+2)*plain {
			t.Errorf("depth %d: a changed text lays out %d elements with nested FillViewport scrollers, %d without", depth, fill, plain)
		}
	}
}

func nestedFillCount(depth int, fill bool) uint32 {
	{
		label := "a"
		var nest func(c *context, d int)
		nest = func(c *context, d int) {
			sc := coreScroll(c).Grow(1)
			if fill {
				sc.FillViewport()
			}
			sc.Children(func() {
				coreColumn(c).Fill().Children(func() {
					coreText(c, label)
					for i := range 20 {
						coreText(c, fmt.Sprint("t", d, i))
					}
					if d > 0 {
						nest(c, d-1)
					} else {
						coreSpacer(c)
					}
				})
			})
		}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() { nest(c, depth) })
		}, 300, 2000)
		tt.Frame()
		label = "b"
		before := tt.rt.computed
		tt.Frame()
		return tt.rt.computed - before
	}
}

// A list of a fixed height inside a short FillViewport content (laid out twice
// a frame, at an unlimited height then at the viewport's) keeps its rows and
// lays out and paints them as kept, as the same list in a plain scroller does:
// counted over idle frames and frames that change a text beside it.
func TestFillViewportKeepsListRows(t *testing.T) {
	type counts struct{ built, kept, same int }
	run := func(fill bool) counts {
		var s ListState
		s.RowVersion = func(i int) uint64 { return 1 }
		label := "a"
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				sc := coreScroll(c).Grow(1)
				if fill {
					sc.FillViewport()
				}
				sc.Children(func() {
					coreColumn(c).Fill().Children(func() {
						coreText(c, label)
						coreList(c, &s, 200, func(i int) {
							coreText(c, fmt.Sprint("row ", i))
						}).Height(150)
						coreSpacer(c)
						coreButton(c, "Apply")
					})
				})
			})
		}, 300, 400)
		for range 3 {
			tt.Frame()
		}
		var n counts
		for i := range 6 {
			if i%2 == 1 {
				label = fmt.Sprint("b", i)
			}
			b, k := tt.rt.rowsBuilt, tt.rt.rowsKept
			tt.Frame()
			n.built += tt.rt.rowsBuilt - b
			n.kept += tt.rt.rowsKept - k
			n.same += tt.rt.paintSameRows
		}
		return n
	}
	plain, fill := run(false), run(true)
	t.Logf("plain %+v, fill %+v", plain, fill)
	if fill != plain {
		t.Fatalf("rows built/kept/laid out as before: %+v in a FillViewport scroller, %+v in a plain one", fill, plain)
	}
}
