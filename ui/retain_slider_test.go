package ui

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// sliderRows: a list whose even rows hold a Slider and whose odd rows hold a Radio,
// which is built in every frame: the rows kept are the rows of sliders, and only
// those. Each twin keeps its own values; RowVersion does not hold them (the
// slider versions itself, by its *float64).
func sliderRows(n int, values *[2][]float64) func() func(c *Context) {
	twin := 0
	return func() func(c *Context) {
		me := twin
		twin++
		pick := new(int)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d", i)
						if i%2 == 0 {
							Slider(c, &values[me][i], 0, 100).Grow(1)
						} else {
							Radio(c, pick, i, "radio")
						}
					})
				}).Grow(1)
			})
		}
	}
}

// What a row of a slider shows is what it was built with: the app moves a value
// without a word (not in RowVersion), and the twin that keeps the rows draws the
// slider where the twin that builds them all does, in the picture and to
// assistive technology; a value changed and changed back builds nothing.
func TestRetainSliderRowsShowTheirValue(t *testing.T) {
	const n = 30
	var values [2][]float64
	values[0], values[1] = make([]float64, n), make([]float64, n)
	p := newPair(t, 400, 300, sliderRows(n, &values))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; k == 0 || b > k {
		t.Fatalf("idle: %d rows built, %d kept; the rows of sliders are kept at rest", b, k)
	}
	set := func(f func(k int)) {
		for k := range 2 {
			f(k)
		}
	}
	set(func(k int) { values[k][2] = 40 })
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("one slider")
	set(func(k int) {
		for i := 0; i < n; i += 2 {
			values[k][i] = float64(i * 3)
		}
	})
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("all sliders")
	p.idle(2)
	rows := p.kept.rt.rowsBuilt
	set(func(k int) { values[k][4] += 7; values[k][4] -= 7 }) // changed and changed back
	p.both(func(tt *Tester) { tt.Frame() })
	p.all("changed back")
	// Only the Radio rows are built in a frame at rest.
	if got := p.kept.rt.rowsBuilt - rows; got > n/2 {
		t.Errorf("a value changed back before the frame built %d rows", got)
	}
}

// A row of a slider is kept once its knob has let go, and built while the knob
// moves (the pointer pressed it, or let it go and it shrinks back).
func TestRetainSliderRowsAreBuiltWhileHeld(t *testing.T) {
	const n = 10
	var values [2][]float64
	values[0], values[1] = make([]float64, n), make([]float64, n)
	p := newPair(t, 400, 300, sliderRows(n, &values))
	p.idle(4)
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(1)
	var x, y float32
	for _, nd := range p.kept.h.access.Nodes {
		if nd.Role == platform.RoleSlider {
			x, y = float32(nd.Bounds.X+nd.Bounds.W/2), float32(nd.Bounds.Y+nd.Bounds.H/2)
			break
		}
	}
	if x == 0 {
		t.Fatal("no slider in the tree")
	}
	p.both(func(tt *Tester) { tt.Press(x, y) })
	p.both(func(tt *Tester) { tt.Advance(40 * time.Millisecond) })
	p.compareFocus("held")
	p.both(func(tt *Tester) { tt.Release(x, y) })
	p.both(func(tt *Tester) { tt.Advance(40 * time.Millisecond) })
	p.compareFocus("let go, shrinking")
	p.both(func(tt *Tester) { tt.Advance(time.Second) })
	p.idle(3)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("rested")
	if k := p.kept.rt.rowsKept - kept; k == 0 {
		t.Errorf("rested: %d rows built, none kept", p.kept.rt.rowsBuilt-built)
	}
}

// A random walk of the keys that move a slider (the arrows, Page, Home and End), the
// pointer (a click, a drag), assistive technology (increment, decrement), the
// wheel, the window's focus and the system's theme and preferences, and the app
// setting a value, over rows with sliders: the twin that keeps the rows draws and
// describes what the twin that builds them all does, with the clock in small steps
// so that frames fall in the middle of the knob's motion. It counts the rows
// kept that hold a slider, so that it fails when they are not kept.
func TestRetainSliderRowsRandomWalk(t *testing.T) {
	for seed := int64(1); seed <= walkSeeds(); seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			const n = 40
			var values [2][]float64
			values[0], values[1] = make([]float64, n), make([]float64, n)
			p := newPair(t, 500, 400, sliderRows(n, &values))
			p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
			p.idle(3)
			rng := rand.New(rand.NewSource(seed))
			sys := newSystemTheme(uint64(seed))
			keptSliders := 0
			keys := []Key{KeyLeft, KeyRight, KeyUp, KeyDown, KeyPageUp, KeyPageDown, KeyHome, KeyEnd}
			steps := 120
			if raceEnabled {
				steps = 50 // -race is ten times slower, and the walks share a job with the rest
			}
			for step := range steps {
				x, y := float32(rng.Intn(520)-10), float32(rng.Intn(420)-10)
				var what string
				switch k := rng.Intn(14); k {
				case 0, 1, 2:
					d := time.Duration(5+rng.Intn(60)) * time.Millisecond
					what = fmt.Sprintf("advance %v", d)
					p.both(func(tt *Tester) { tt.Advance(d) })
				case 3:
					what = "tab"
					p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
				case 4, 5:
					key := keys[rng.Intn(len(keys))]
					what = fmt.Sprintf("key %v", key)
					p.both(func(tt *Tester) { tt.Key(0, key) })
				case 6:
					what = fmt.Sprintf("click %v,%v", x, y)
					p.both(func(tt *Tester) { tt.ClickAt(x, y) })
				case 7:
					x2 := float32(rng.Intn(520) - 10)
					what = fmt.Sprintf("drag %v,%v to %v", x, y, x2)
					p.both(func(tt *Tester) {
						tt.Press(x, y)
						tt.Frame()
						tt.Move((x+x2)/2, y)
						tt.Frame()
						tt.Move(x2, y)
						tt.Release(x2, y)
					})
				case 8:
					what = fmt.Sprintf("move %v,%v", x, y)
					p.both(func(tt *Tester) { tt.Move(x, y) })
				case 9:
					what = "blur and focus"
					p.both(func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
				case 10:
					i := rng.Intn(n/2) * 2
					what = fmt.Sprintf("app sets %d", i)
					v := float64(rng.Intn(101))
					for k := range 2 {
						values[k][i] = v
					}
				case 11, 12:
					var ids []uint64
					for _, nd := range p.kept.h.access.Nodes {
						if nd.Role == platform.RoleSlider {
							ids = append(ids, nd.ID)
						}
					}
					if len(ids) == 0 {
						continue
					}
					id := ids[rng.Intn(len(ids))]
					act := platform.AccessIncrement
					if rng.Intn(2) == 0 {
						act = platform.AccessDecrement
					}
					what = fmt.Sprintf("assistive technology %v on %d", act, id)
					p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: id, Action: act}) })
				default:
					what = "scroll"
					dy := float32(rng.Intn(120) - 60)
					p.both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
				}
				p.both(func(tt *Tester) { tt.Frame() })
				what += sys.step(p.both)
				p.compareFocus(fmt.Sprintf("%d %s", step, what))
				before := p.kept.rt.rowsKept
				p.frame(fmt.Sprintf("%d %s +1", step, what))
				keptSliders += p.kept.rt.rowsKept - before // the Radio rows are never kept
			}
			if keptSliders == 0 {
				t.Fatalf("no row of a slider kept: the walk tests nothing")
			}
		})
	}
}
