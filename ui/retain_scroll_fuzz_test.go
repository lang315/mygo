package ui

import (
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

type matchOuter struct{}

var scrollFuzzSeeds = flag.Int("retain.scrollseeds", 40, "seeds of TestRetainScrollFuzz")

// A view built to be scrolled (the random trees of the model rarely hold a list
// that scrolls): a List whose rows hold every kind of painting, a scroll
// container and versions, put through random scrolls of every size, hover,
// clicks, version bumps, resizes, themes and scales. A kept Tester that checks
// its layouts and replays and one that builds every row draw alike after each
// step, with ==.
func TestRetainScrollFuzz(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)
	seeds := *scrollFuzzSeeds
	if raceEnabled && seeds == 40 {
		seeds = 4
	}
	for _, scale := range []float32{1, 2} {
		var shifted, replayed int
		for seed := range uint64(seeds) {
			s, r := scrollFuzzSeed(t, scale, seed+uint64(*retainSeed0), 50)
			shifted += s
			replayed += r
		}
		if shifted == 0 || replayed == 0 {
			t.Errorf("scale %v: %d rows moved with their layout kept and %d replayed their paint shifted: the fuzz no longer scrolls them", scale, shifted, replayed)
		}
		t.Logf("scale %v: %d rows moved by layout, %d replayed shifted", scale, shifted, replayed)
	}
}

func scrollFuzzSeed(t *testing.T, scale float32, seed uint64, steps int) (shifted, replayed int) {
	t.Helper()
	rng := rand.New(rand.NewPCG(seed, 77))
	type app struct {
		vers  []uint64
		dark  bool
		count int
		boxH  float32 // the height of the element around the list, which a Match transition animates
		show  bool    // false: another element, without the list, holds the Match
	}
	var apps []*app
	n := 40 + rng.IntN(300)
	p := newPair(t, 300, 400, func() func(c *Context) {
		a := &app{vers: make([]uint64, n), count: n, boxH: 250, show: true}
		apps = append(apps, a)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return a.vers[i] }}
		return func(c *Context) {
			if a.dark {
				c.SetTheme(DarkTheme())
			}
			listView := func() {
				List(c, s, a.count, func(i int) {
					r := Row(c).Gap(4)
					switch (i + int(a.vers[i])) % 8 {
					case 0:
						r.Gradient(RGB(250, 240, 200), RGB(200, 220, 250), 90)
					case 1:
						r.Shadow(0, 2, 6, 0, RGB(120, 120, 120))
					case 2:
						r.Border(1, RGB(90, 90, 200)).BorderStyle(BorderDashed).Radius(4)
					case 3:
						r.Background(RGB(220, 230, 240)).Radius(7)
					case 4:
						r.Padding(0.25, 3)
					}
					r.Children(func() {
						Textf(c, "row %d v%d", i, a.vers[i])
						if i%5 == 0 {
							RichText(c, Span{Text: "rich ", Weight: 700}, Span{Text: "text", Italic: true, Underline: true})
						}
						if i%7 == 0 {
							ScrollHorizontal(c).Width(50).Height(18).Children(func() {
								Row(c).Width(200).Children(func() { Text(c, "wide"); Spacer(c); Text(c, "content") })
							})
						}
						Text(c, "·")
					})
				}).Grow(1)
			}
			// The list in an element of a Transition's Match: showAt lays it out
			// and places it a second time in a layout, or, when the element keeps
			// its size and moves, shifts it. In the second kind of seeds two Matches
			// are nested: the inner one keeps its size and moves, the outer one
			// changes size at the same place (the shift, then the place again).
			to := ElementTransition{Duration: 300 * time.Millisecond, Ease: Linear, Match: matchOuter{}}
			ti := ElementTransition{Duration: 300 * time.Millisecond, Ease: Linear, Match: matchKey{}}
			if seed%3 != 0 {
				ti = ElementTransition{Animation: &Animation{Duration: 200 * time.Millisecond, Ease: Linear, Delay: time.Duration(seed%3) * 50 * time.Millisecond}, Match: matchKey{}}
			}
			Column(c).Fill().Children(func() {
				if !a.show {
					Column(c).Key("oa").Height(200).Children(func() {
						Column(c).Key("ia").Height(a.boxH).Transition(ti).Children(func() { Text(c, "a") })
					}).Transition(to)
					return
				}
				Column(c).Key("ob").Height(400).Children(func() {
					Text(c, "spacer").Height(60)
					Column(c).Key("ib").Height(a.boxH).Transition(ti).Children(listView)
				}).Transition(to)
			})
		}
	})
	p.both(func(tt *Tester) { tt.SetScale(scale) })
	p.kept.rt.replayCheck, p.kept.rt.layoutCheck = true, true
	p.idle(3)
	var log []string
	check := func(what string) {
		t.Helper()
		log = append(log, what)
		if len(log) > 6 {
			log = log[len(log)-6:]
		}
		if d := sameFrames(p.kept, p.built, true); d != "" {
			t.Fatalf("scale %v seed %d, after %s: %s", scale, seed, strings.Join(log, ", "), d)
		}
		if d := sameNodes(p.kept.rt.c.root, p.built.rt.c.root, "root"); d != "" {
			t.Fatalf("scale %v seed %d, after %s: %s", scale, seed, strings.Join(log, ", "), d)
		}
		if p.kept.rt.layoutMismatch != 0 || p.kept.rt.replayMismatch != 0 {
			t.Fatalf("scale %v seed %d, after %s: %d layouts and %d replays differ from doing the work", scale, seed, strings.Join(log, ", "), p.kept.rt.layoutMismatch, p.kept.rt.replayMismatch)
		}
		shifted += p.kept.rt.rowsShifted
		replayed += p.kept.rt.rowsShiftReplayed
	}
	dys := []float32{1, 2, 3, 0.5, 0.25, 1.13, -1, -2, -0.5, 5, -5, 9}
	for range steps {
		var what string
		switch v := rng.IntN(20); {
		case v < 8: // scrolls of every size
			d := dys[rng.IntN(len(dys))]
			what = fmt.Sprintf("scroll %v", d)
			p.both(func(tt *Tester) { tt.Scroll(150, 200, 0, d) })
		case v < 10: // the element around the list changes height and the other element takes over the Match
			h := float32(300 + rng.IntN(120))
			if rng.IntN(2) == 0 {
				h = apps[0].boxH // the inner element keeps its size and moves; the outer one changes size
			}
			what = fmt.Sprintf("match %v", h)
			for _, a := range apps {
				a.boxH = h
				a.show = !a.show
			}
			p.both(func(tt *Tester) { tt.Frame() })
			for range rng.IntN(4) {
				if rng.IntN(2) == 0 {
					p.both(func(tt *Tester) { tt.Frame() }) // the clock stands still
				} else {
					p.both(func(tt *Tester) { tt.Advance(16 * time.Millisecond) })
				}
				check("match step")
			}
		case v < 12:
			what = "idle"
			p.both(func(tt *Tester) { tt.Frame() })
		case v < 14:
			y := float32(rng.IntN(380))
			what = fmt.Sprintf("move %v", y)
			p.both(func(tt *Tester) { tt.Move(100, y) })
		case v < 15:
			y := float32(rng.IntN(380))
			what = fmt.Sprintf("click %v", y)
			p.both(func(tt *Tester) { tt.ClickAt(60, y) })
		case v < 17:
			i := rng.IntN(n)
			what = fmt.Sprintf("version %d", i)
			for _, a := range apps {
				a.vers[i]++
			}
			p.both(func(tt *Tester) { tt.Frame() })
		case v < 18:
			w := float32(240 + rng.IntN(120))
			what = fmt.Sprintf("width %v", w)
			p.both(func(tt *Tester) { tt.h.w = w; tt.Frame() })
		case v < 19:
			for _, a := range apps {
				a.dark = !a.dark
			}
			what = "theme"
			p.both(func(tt *Tester) { tt.Frame() })
		case v < 20 && rng.IntN(2) == 0: // the element around the list changes height, and the clock moves or stands still
			h := float32(300 + rng.IntN(120))
			what = fmt.Sprintf("box %v", h)
			for _, a := range apps {
				a.boxH = h
				if int(h)%2 == 0 {
					a.show = !a.show // the other element takes over the Match: showAt
				}
			}
			if rng.IntN(2) == 0 {
				p.both(func(tt *Tester) { tt.Frame() })
			} else {
				p.both(func(tt *Tester) { tt.Advance(16 * time.Millisecond) })
			}
		default:
			h := float32(250 + rng.IntN(250))
			what = fmt.Sprintf("height %v", h)
			p.both(func(tt *Tester) { tt.h.h = h; tt.Frame() })
		}
		check(what)
	}
	return shifted, replayed
}
