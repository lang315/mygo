package ui

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"
)

// sameFrames reports where two Testers' frames differ: the pictures, the
// operations they were drawn with, and what commit recorded of them, the hits,
// the labels, the focus order and the flags of every state.
func sameFrames(kept, built *Tester, pictures bool) string {
	if ia, ib := kept.Image(), built.Image(); pictures && !bytes.Equal(ia.Pix, ib.Pix) {
		at := 0
		for at < len(ia.Pix) && ia.Pix[at] == ib.Pix[at] {
			at++
		}
		px := at / 4
		return fmt.Sprintf("the pictures differ, first at %d,%d (kept %v, built %v); boxes: %q", px%ia.Rect.Dx(), px/ia.Rect.Dx(), ia.Pix[at-at%4:at-at%4+4], ib.Pix[at-at%4:at-at%4+4], sameBoxes(kept.rt.c.root, built.rt.c.root, "root"))
	}
	if d := sameOps(withoutEmptyClips(kept.rt.scene.Ops), withoutEmptyClips(built.rt.scene.Ops)); d != "" {
		return "ops: " + d
	}
	a, b := kept.rt, built.rt
	if len(a.hits) != len(b.hits) {
		return fmt.Sprintf("%d hits, want %d", len(a.hits), len(b.hits))
	}
	for i := range a.hits {
		if a.hits[i].st.id != b.hits[i].st.id || a.hits[i].r != b.hits[i].r || a.hits[i].flags != b.hits[i].flags {
			ctx := ""
			for j := max(0, i-2); j < min(len(a.hits), i+3); j++ {
				ctx += fmt.Sprintf(" [%d kept id %d %v %#x | built id %d %v %#x]", j, a.hits[j].st.id, a.hits[j].r, a.hits[j].flags, b.hits[j].st.id, b.hits[j].r, b.hits[j].flags)
			}
			return fmt.Sprintf("hit %d differs:%s", i, ctx)
		}
	}
	if len(a.labels) != len(b.labels) {
		return fmt.Sprintf("%d labels, want %d", len(a.labels), len(b.labels))
	}
	for i := range a.labels {
		if a.labels[i] != b.labels[i] {
			return fmt.Sprintf("label %d: %+v, want %+v", i, a.labels[i], b.labels[i])
		}
	}
	if len(a.focusOrder) != len(b.focusOrder) {
		return fmt.Sprintf("focus order of %d, want %d", len(a.focusOrder), len(b.focusOrder))
	}
	for i := range a.focusOrder {
		if a.focusOrder[i] != b.focusOrder[i] {
			return fmt.Sprintf("focus order %d: %d, want %d", i, a.focusOrder[i], b.focusOrder[i])
		}
	}
	if len(a.states) != len(b.states) {
		return fmt.Sprintf("%d states, want %d", len(a.states), len(b.states))
	}
	for id, s := range a.states {
		o := b.states[id]
		if o == nil {
			return fmt.Sprintf("state %d is not in the other", id)
		}
		if s.flags != o.flags || s.x != o.x || s.y != o.y || s.w != o.w || s.h != o.h ||
			s.vx != o.vx || s.vy != o.vy || s.vw != o.vw || s.vh != o.vh || s.role != o.role || s.parent != o.parent {
			return fmt.Sprintf("state %d: flags %#x box %v,%v %v×%v, want flags %#x box %v,%v %v×%v", id, s.flags, s.x, s.y, s.w, s.h, o.flags, o.x, o.y, o.w, o.h)
		}
	}
	return ""
}

// Keeping the rows that did not change builds what building them does, as
// random trees of Lists and Tables change frame after frame (mutations of
// their data, which version their rows), and the pointer moves, presses,
// clicks and scrolls, the keys move the focus, and the theme changes. Two
// Testers, one of them building every row, are driven alike: after each frame
// their pictures, operations, hits, labels, focus order and states are the same.
var (
	retainSeeds = flag.Int("retain.seeds", 12, "seeds of TestRetainMatchesFresh")
	retainSeed0 = flag.Int("retain.seed0", 0, "the first seed of TestRetainMatchesFresh")
)

func TestRetainMatchesFresh(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)
	seeds, frames := *retainSeeds, 60
	if testing.Short() && *retainSeeds == 12 {
		seeds = 5
	}
	if raceEnabled && *retainSeeds == 12 {
		seeds = 4
	}
	for _, scale := range []float32{1, 2} {
		var kept, built int
		for seed := range uint64(seeds) {
			k, b := retainModelSeed(t, scale, seed+uint64(*retainSeed0), frames)
			kept += k
			built += b
		}
		if kept == 0 {
			t.Errorf("scale %v: no row was kept in %d built", scale, built)
		}
		t.Logf("scale %v: %d rows kept, %d built", scale, kept, built)
	}
}

// retainModelSeed drives the model of the seed in two Testers, one of them
// building every row, for frames frames, failing the test at the first
// difference between them. It returns how many rows the kept one linked and
// built.
func retainModelSeed(t *testing.T, scale float32, seed uint64, frames int) (kept, built int) {
	t.Helper()
	m := newLCModel(seed + 3000)
	m.versions, m.noDup, m.inner, m.deco, m.menus = true, true, true, true, true
	themes := [2]*Theme{LightTheme(), DarkTheme()}
	theme := 0
	view := func(tester int) func(c *context) {
		return func(c *context) {
			c.SetTheme(themes[theme])
			coreBox(c).Fill().Children(func() { m.build(c, tester) })
		}
	}
	start := time.Unix(1e9, 0)
	// The model treats its first Tester as the one it asks about the
	// tree; the two compared are the others. It builds every row, like b.
	first := lcTester(view(0), m.w, m.h, start)
	a, b := lcTester(view(1), m.w, m.h, start), lcTester(view(2), m.w, m.h, start)
	first.rt.noRetain, b.rt.noRetain = true, true
	a.rt.replayCheck = true // rows replayed are painted as well, and compared
	a.rt.layoutCheck = true // rows kept are laid out and committed as well, and compared
	m.testers = [3]*Tester{first, a, b}
	both := func(f func(tt *Tester)) {
		f(first)
		f(a)
		f(b)
	}
	both(func(tt *Tester) { tt.SetScale(scale) })
	var steps []string
	rightClicks := rand.New(rand.NewPCG(seed, 5)) // a stream of its own: the other steps stay as they were
	for f := range frames {
		m.dup = nil
		var what string
		switch v := m.rng.IntN(15); {
		case v < 3:
			for range 1 + m.rng.IntN(2) {
				what = m.mutate()
			}
			both(func(tt *Tester) { tt.h.w, tt.h.h = float32(m.w), float32(m.h); tt.Frame() })
		case v < 5:
			x, y := float32(m.rng.IntN(m.w)), float32(m.rng.IntN(m.h))
			what = fmt.Sprintf("move %v,%v", x, y)
			both(func(tt *Tester) { tt.Move(x, y) })
		case v < 6:
			x, y := float32(m.rng.IntN(m.w)), float32(m.rng.IntN(m.h))
			// TestRetainSeed11 (retainRightClicks) draws from the model's own
			// stream, as the run that found the bug did; the other seeds take
			// theirs from a stream of their own.
			rc := rightClicks
			if retainRightClicks {
				rc = m.rng
			}
			if rc.IntN(2) == 0 {
				// The secondary button over a text of a row, which opens the
				// menu drawn (PopoverMenu) of the row's first text, if the click
				// hits it.
				var at []Point
				var find func(n *node, inRow bool)
				find = func(n *node, inRow bool) {
					inRow = inRow || n.listRow
					first := n.parent != nil && n.parent.listRow && n == n.parent.first
					if (retainRightClicks && inRow || first) && n.kind == kindText && n.w > 0 && n.x >= 0 && n.y >= 0 && n.x+n.w <= float32(m.w) && n.y+n.h <= float32(m.h) {
						at = append(at, Point{n.x + n.w/2, n.y + n.h/2})
					}
					for c := n.first; c != nil; c = c.next {
						find(c, inRow)
					}
				}
				find(first.rt.c.root, false)
				if len(at) > 0 {
					p := at[rc.IntN(len(at))]
					x, y = p.X, p.Y
				}
				what = fmt.Sprintf("right click %v,%v", x, y)
				if retainRightClicks {
					both(func(tt *Tester) { tt.RightClickAt(x, y) })
				} else {
					both(func(tt *Tester) {
						tt.Frame() // the rows are kept when the menu is asked for
						tt.Frame()
						tt.Frame()
						tt.RightClickAt(x, y)
						tt.Frame()
						tt.Frame()
					})
				}
				break
			}
			what = fmt.Sprintf("click %v,%v", x, y)
			both(func(tt *Tester) { tt.ClickAt(x, y) })
		case v < 7:
			what = "tab"
			both(func(tt *Tester) { tt.Key(0, KeyTab) })
		case v < 9:
			x, y := float32(m.rng.IntN(m.w)), float32(m.rng.IntN(m.h))
			dy := []float32{1, -1, 0.5, -0.5, 30, -30, 200, 0.25, 0.37, -1.13, 500, -500}[m.rng.IntN(12)]
			dx := []float32{0, 0, 0, 0, 12, -12, 40}[m.rng.IntN(7)]
			pressed := m.rng.IntN(4) == 0 // the pointer pressed elsewhere: the hover list is frozen
			if pressed {
				dx = []float32{12, -12, 40}[m.rng.IntN(3)]
			}
			if dx != 0 {
				// Over a scroll container of a row, if the view has one in sight.
				var at []Point
				var find func(n *node, inRow bool)
				find = func(n *node, inRow bool) {
					inRow = inRow || n.listRow
					if inRow && n.flags&flagScrollX != 0 && n.w > 0 && n.x >= 0 && n.y >= 0 && n.x+n.w <= float32(m.w) && n.y+n.h <= float32(m.h) {
						at = append(at, Point{n.x + n.w/2, n.y + n.h/2})
					}
					for c := n.first; c != nil; c = c.next {
						find(c, inRow)
					}
				}
				find(first.rt.c.root, false)
				if len(at) > 0 {
					p := at[m.rng.IntN(len(at))]
					x, y = p.X, p.Y
				}
			}
			what = fmt.Sprintf("scroll %v,%v at %v,%v pressed %v", dx, dy, x, y, pressed)
			both(func(tt *Tester) {
				if pressed {
					tt.Frame() // idle frames first: the rows are held when the scroll comes
					tt.Frame()
					tt.Frame()
					tt.Press(1, 1)
				}
				tt.Scroll(x, y, dx, dy)
				if pressed {
					tt.Release(1, 1)
					tt.Frame() // the frames after the release show the offset the row kept
					tt.Frame()
				}
			})
		case v < 10:
			theme = 1 - theme
			what = "theme"
			both(func(tt *Tester) { tt.Frame() })
		case v < 11:
			// How the rows look changes, with their version.
			if l := pickList(m); l != nil {
				l.tint = (l.tint + 1) % 5
				what = "tint"
			}
			both(func(tt *Tester) { tt.Frame() })
		case v < 13:
			// The choice, or the checks, change: in the rows' keys, and not in their version.
			if l := pickList(m); l != nil {
				key := any(m.rng.IntN(l.rows()))
				if l.keyed {
					key = l.at(key.(int))
				}
				sets := &l.chosen
				what = "choose"
				if v == 12 && l.kind == lcTable {
					sets, what = &l.checked, "check"
				}
				for i := range sets {
					sets[i].set(key, !sets[i].has(key))
				}
			}
			both(func(tt *Tester) { tt.Frame() })
		default:
			what = "idle"
			both(func(tt *Tester) { tt.Frame() })
		}
		steps = append(steps, what)
		if a.rt.layoutMismatch != 0 {
			t.Fatalf("seed %d scale %v frame %d (%s): %d rows kept or replayed differ from laying out and committing them", seed, scale, f, what, a.rt.layoutMismatch)
		}
		if a.rt.replayMismatch != 0 {
			t.Fatalf("seed %d scale %v frame %d (%s): %d replayed rows differ from painting them", seed, scale, f, what, a.rt.replayMismatch)
		}
		d := sameNodes(a.rt.c.root, b.rt.c.root, "root")
		if d == "" {
			d = sameFrames(a, b, true)
		}
		if d == "" {
			d = sameStates(a, b)
		}
		if d != "" {
			if len(steps) > 6 {
				steps = steps[len(steps)-6:]
			}
			t.Fatalf("scale %v, seed %d, frame %d, after %s: %s", scale, seed, f, strings.Join(steps, ", "), d)
		}
	}
	return a.rt.rowsKept, a.rt.rowsBuilt
}

// pickList returns a List or a Table of the model, nil if it has none.
func pickList(m *lcModel) *lcNode {
	nodes, _ := m.all()
	var lists []*lcNode
	for _, n := range nodes {
		if n.kind == lcList || n.kind == lcTable {
			lists = append(lists, n)
		}
	}
	if len(lists) == 0 {
		return nil
	}
	return lists[m.rng.IntN(len(lists))]
}

// sameNodes compares the elements of two trees: what layout and commit left in
// them.
func sameNodes(a, b *node, path string) string {
	if a.id != b.id {
		return fmt.Sprintf("%s: id %d, want %d", path, a.id, b.id)
	}
	if a.x != b.x || a.y != b.y || a.w != b.w || a.h != b.h || a.contentW != b.contentW || a.contentH != b.contentH {
		return fmt.Sprintf("%s: box %v,%v %v×%v content %v×%v, want %v,%v %v×%v content %v×%v", path, a.x, a.y, a.w, a.h, a.contentW, a.contentH, b.x, b.y, b.w, b.h, b.contentW, b.contentH)
	}
	if a.world != b.world || a.ink != b.ink || a.noCull != b.noCull {
		return fmt.Sprintf("%s: world %v ink %v noCull %v, want world %v ink %v noCull %v", path, a.world, a.ink, a.noCull, b.world, b.ink, b.noCull)
	}
	if a.nchild != b.nchild {
		return fmt.Sprintf("%s: %d children, want %d", path, a.nchild, b.nchild)
	}
	for ca, cb, i := a.first, b.first, 0; ca != nil && cb != nil; ca, cb, i = ca.next, cb.next, i+1 {
		if d := sameNodes(ca, cb, fmt.Sprintf("%s/%d", path, i)); d != "" {
			return d
		}
	}
	return ""
}

// sameStates compares every field commit writes of the states two Testers
// share.
func sameStates(a, b *Tester) string {
	for id, s := range a.rt.states {
		o := b.rt.states[id]
		if o == nil {
			continue // sameFrames compares the sets
		}
		if s.x != o.x || s.y != o.y || s.w != o.w || s.h != o.h || s.vx != o.vx || s.vy != o.vy || s.vw != o.vw || s.vh != o.vh ||
			s.cx != o.cx || s.cy != o.cy || s.cw != o.cw || s.ch != o.ch || s.flags != o.flags || s.world != o.world || s.clips != o.clips ||
			s.clipRect != o.clipRect || s.clipRadii != o.clipRadii || s.rtl != o.rtl || s.page != o.page || s.scope != o.scope ||
			s.anchor != o.anchor || s.cursor != o.cursor || s.tip != o.tip || s.contentW != o.contentW || s.contentH != o.contentH {
			return fmt.Sprintf("state %d: box %v,%v %v×%v visible %v,%v %v×%v content %v,%v %v×%v flags %#x, want box %v,%v %v×%v visible %v,%v %v×%v content %v,%v %v×%v flags %#x (other fields may differ)",
				id, s.x, s.y, s.w, s.h, s.vx, s.vy, s.vw, s.vh, s.cx, s.cy, s.cw, s.ch, uint32(s.flags),
				o.x, o.y, o.w, o.h, o.vx, o.vy, o.vw, o.vh, o.cx, o.cy, o.cw, o.ch, uint32(o.flags))
		}
	}
	return ""
}

// retainRightClicks makes the click step of the model right-click, half the
// time, a text of a row; the extra numbers it draws give every step after it
// another value. It found the rows kept differing from the rows built that
// TestRetainSeed11 pins.
var retainRightClicks bool

func TestRetainSeed11(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)
	retainRightClicks = true
	defer func() { retainRightClicks = false }()
	retainModelSeed(t, 1, 11, 60)
}
