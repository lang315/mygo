package ui

import (
	"bytes"
	"fmt"
	"io"
	"log"
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
func TestRetainMatchesFresh(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)
	seeds, frames := 12, 60
	if testing.Short() {
		seeds = 5
	}
	if raceEnabled {
		seeds = 4
	}
	for _, scale := range []float32{1, 2} {
		var kept, built int
		for seed := range uint64(seeds) {
			m := newLCModel(seed + 3000)
			m.versions, m.noDup = true, true
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
			m.testers = [3]*Tester{first, a, b}
			both := func(f func(tt *Tester)) {
				f(first)
				f(a)
				f(b)
			}
			both(func(tt *Tester) { tt.SetScale(scale) })
			var steps []string
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
					what = fmt.Sprintf("click %v,%v", x, y)
					both(func(tt *Tester) { tt.ClickAt(x, y) })
				case v < 7:
					what = "tab"
					both(func(tt *Tester) { tt.Key(0, KeyTab) })
				case v < 9:
					x, y := float32(m.rng.IntN(m.w)), float32(m.rng.IntN(m.h))
					dy := []float32{1, -1, 0.5, -0.5, 30, -30, 200}[m.rng.IntN(7)]
					what = fmt.Sprintf("scroll %v at %v,%v", dy, x, y)
					both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
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
				if d := sameFrames(a, b, true); d != "" {
					if len(steps) > 6 {
						steps = steps[len(steps)-6:]
					}
					t.Fatalf("scale %v, seed %d, frame %d, after %s: %s", scale, seed, f, strings.Join(steps, ", "), d)
				}
			}
			kept += a.rt.rowsKept
			built += a.rt.rowsBuilt
		}
		if kept == 0 {
			t.Errorf("scale %v: no row was kept in %d built", scale, built)
		}
		t.Logf("scale %v: %d rows kept, %d built", scale, kept, built)
	}
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
