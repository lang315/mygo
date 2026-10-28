package ui

import (
	"strings"
	"testing"
	"time"
)

// replayExtras change what paints an element and not what lays it out, which
// the layout cache's styles all do; those of TestPaintCullMatches mostly keep a
// row from replaying (transforms, Draw, blur).
var replayExtras = []func(e *node){
	nil,
	func(e *node) { e.Background(RGB(250, 200, 200)) },
	func(e *node) { e.Background(RGB(200, 200, 250)) },
	func(e *node) { e.TextColor(RGB(200, 0, 0)) },
	func(e *node) { e.TextColor(RGB(0, 0, 200)) },
	func(e *node) { e.Opacity(0.6) },
	func(e *node) { e.BorderColor(RGB(0, 150, 0)) },
	func(e *node) { e.Radius(4).Background(RGB(230, 230, 190)) },
}

// Replaying rows paints what painting them does, as random trees with lists
// and tables change, scroll, hover and take focus frame after frame, at scale
// 1 and 2: each replayed row is painted as well (replayCheck), and a frame's
// operations are compared with a painting that replays nothing.
func TestReplayMatches(t *testing.T) {
	seeds, frames := 16, 60
	if testing.Short() || raceEnabled {
		seeds = 4
	}
	var replayed int
	for _, scale := range []float32{1, 2} {
		for seed := range uint64(seeds) {
			m := newLCModel(seed + 2000)
			extras := map[*lcNode]int{}
			m.extra = func(n *lcNode, e *node) {
				if f := replayExtras[extras[n]]; f != nil {
					f(e)
				}
			}
			restyle := func() {
				nodes, _ := m.all()
				for _, n := range nodes {
					if m.rng.IntN(4) == 0 {
						extras[n] = m.rng.IntN(len(replayExtras))
					}
				}
			}
			restyle()
			invisible := false
			tt := lcTester(func(c *context) {
				b := coreBox(c).Fill()
				if invisible {
					b.Invisible()
				}
				b.Children(func() { m.build(c, 0) })
			}, m.w, m.h, time.Unix(1e9, 0))
			tt.h.scale = scale
			tt.rt.replayCheck = true
			var log []string
			for f := range frames {
				m.dup = nil
				switch k := m.rng.IntN(9); {
				case k < 2:
					for range 1 + m.rng.IntN(2) {
						log = append(log, m.mutate())
					}
					restyle()
				case k == 2:
					tt.Move(float32(m.rng.IntN(m.w)), float32(m.rng.IntN(m.h)))
					log = append(log, "move")
				case k == 3:
					tt.Scroll(float32(m.rng.IntN(m.w)), float32(m.rng.IntN(m.h)), 0, float32(m.rng.IntN(7)-3))
					log = append(log, "scroll")
				case k == 4:
					tt.Key(0, KeyTab)
					log = append(log, "tab")
				case k == 5:
					// What paints changes, what lays out does not.
					restyle()
					log = append(log, "restyle")
				case k == 6:
					invisible = !invisible
					log = append(log, "invisible")
				default:
					log = append(log, "unchanged")
				}
				tt.h.w, tt.h.h = float32(m.w), float32(m.h)
				// The frame, then frames with nothing changed: a row that changed
				// stands still again, and must not replay what it was before.
				for rep := range 1 + m.rng.IntN(4) {
					tt.Frame()
					replayed += tt.rt.replayed
					w, h, sc := tt.h.size()
					tt.rt.paint(tt.rt.c.root, w, h, sc)
					replayed += tt.rt.replayed
					got := withoutEmptyClips(tt.rt.scene.Ops)
					tt.rt.noReplay = true
					tt.rt.paint(tt.rt.c.root, w, h, sc)
					tt.rt.noReplay = false
					if len(log) > 6 {
						log = log[len(log)-6:]
					}
					if tt.rt.replayMismatch != 0 {
						t.Fatalf("scale %v, seed %d, frame %d.%d, after %s: %d replayed rows differ from painting them",
							scale, seed, f, rep, strings.Join(log, ", "), tt.rt.replayMismatch)
					}
					if d := sameOps(got, withoutEmptyClips(tt.rt.scene.Ops)); d != "" {
						t.Fatalf("scale %v, seed %d, frame %d.%d, after %s: %s", scale, seed, f, rep, strings.Join(log, ", "), d)
					}
				}
			}
		}
	}
	t.Logf("%d rows replayed", replayed)
	if replayed == 0 {
		t.Error("no row replayed")
	}
}
