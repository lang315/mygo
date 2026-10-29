package ui

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// P4b: a row that moved replays its painting shifted by whole device pixels.
// The kept Tester paints every replayed row as well (replayCheck) and counts
// what differs; the pictures and ops of the two Testers are compared with ==.

// shiftPair is a pair whose kept Tester checks its replays.
func shiftPair(t *testing.T, scale float32, mk func() func(c *Context)) *pair {
	t.Helper()
	p := newPair(t, 300, 400, mk)
	p.both(func(tt *Tester) { tt.SetScale(scale) })
	p.kept.rt.replayCheck = true
	p.idle(4)
	return p
}

// scrollBy scrolls by dy and compares, failing if a replay differed from a
// painting.
func (p *pair) scrollBy(dy float32, what string) {
	p.t.Helper()
	p.scrolled(dy, what)
	if n := p.kept.rt.replayMismatch; n != 0 {
		p.t.Fatalf("%s: %d replayed rows differ from painting them", what, n)
	}
}

func TestRetainShiftPaintScroll1px(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		t.Run(fmt.Sprintf("scale %v", scale), func(t *testing.T) {
			p := shiftPair(t, scale, newKeptList(500).mk)
			p.scrollBy(1, "first move")
			p.idle(2)
			total := 0
			for i := range 8 {
				p.scrollBy(float32(1+i%2), fmt.Sprintf("scroll %d", i))
				total += p.kept.rt.rowsShiftReplayed
			}
			if total == 0 {
				t.Fatal("no row replayed its paint shifted")
			}
		})
	}
}

// A shift that is not a whole device pixel is refused and the record stays:
// the next frame, a whole pixel from it, replays.
func TestRetainShiftPaintHalfPixelIsRefused(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	p.scrollBy(1, "first move")
	p.idle(2)
	p.scrollBy(0.5, "half")
	if n := p.kept.rt.rowsShiftReplayed; n != 0 {
		t.Errorf("a shift by half a pixel replayed %d rows", n)
	}
	p.scrollBy(0.5, "half again")
	if n := p.kept.rt.rowsShiftReplayed; n == 0 {
		t.Errorf("a whole pixel from the record replayed no row")
	}
}

// At scale 2 half a DIP is one device pixel: it replays; a quarter is not.
func TestRetainShiftPaintScaleTwo(t *testing.T) {
	p := shiftPair(t, 2, newKeptList(500).mk)
	p.scrollBy(1, "first move")
	p.idle(2)
	p.scrollBy(0.5, "half a DIP")
	if n := p.kept.rt.rowsShiftReplayed; n == 0 {
		t.Errorf("half a DIP at scale 2 (one device pixel) replayed no row")
	}
	p.scrollBy(0.25, "a quarter")
	if n := p.kept.rt.rowsShiftReplayed; n != 0 {
		t.Errorf("a quarter of a DIP at scale 2 replayed %d rows", n)
	}
	p.scrollBy(0.25, "a quarter again")
	p.idle(2)
}

// Rows cut by the clip at the top and the bottom are painted anew: culling
// kept what showed at the place they recorded.
func TestRetainShiftPaintCutRows(t *testing.T) {
	for _, h := range []int{400, 301, 260} {
		t.Run(fmt.Sprint(h), func(t *testing.T) {
			p := newPair(t, 300, h, newKeptList(500).mk)
			p.kept.rt.replayCheck = true
			p.idle(4)
			p.scrollBy(1, "first move")
			p.idle(2)
			for i := range 12 {
				p.scrollBy(float32(1+i%3), fmt.Sprintf("scroll %d", i))
			}
		})
	}
}

// Rows of every kind of painting scroll alike: fills with gradients, shadows,
// borders, rounded corners, rich text and icons.
func TestRetainShiftPaintRowKinds(t *testing.T) {
	kinds := map[string]func(c *Context, e Element){
		"linear gradient": func(c *Context, e Element) { e.Gradient(RGB(250, 240, 200), RGB(200, 220, 250), 90) },
		"conic gradient": func(c *Context, e Element) {
			e.ConicGradient(ConicGradient{Stops: []GradientStop{{RGB(200, 0, 0), 0}, {RGB(0, 0, 200), 1}}})
		},
		"shadow":        func(c *Context, e Element) { e.Shadow(0, 2, 6, 0, RGB(120, 120, 120)) },
		"dashed border": func(c *Context, e Element) { e.Border(1, RGB(90, 90, 200)).BorderStyle(BorderDashed).Radius(4) },
		"rounded fill":  func(c *Context, e Element) { e.Background(RGB(220, 230, 240)).Radius(7) },
		"padding 0.25":  func(c *Context, e Element) { e.Padding(0.25, 3) },
		"rich text": func(c *Context, e Element) {
			RichText(c, Span{Text: "rich ", Weight: 700}, Span{Text: "text", Italic: true, Underline: true})
		},
		"icon":  func(c *Context, e Element) { Icon(c, nil) },
		"plain": func(c *Context, e Element) {},
	}
	for name, deco := range kinds {
		for _, scale := range []float32{1, 2} {
			t.Run(fmt.Sprintf("%s/scale %v", name, scale), func(t *testing.T) {
				if name == "icon" {
					t.Skip("an icon needs an SVG: covered by the model")
				}
				p := shiftPair(t, scale, func() func(c *Context) {
					s := auditState()
					return func(c *Context) {
						List(c, s, 120, func(i int) {
							r := Row(c)
							deco(c, r)
							r.Children(func() { Textf(c, "row %d", i); Text(c, "·") })
						}).Grow(1)
					}
				})
				p.scrollBy(1, "first move")
				p.idle(2)
				replayed := 0
				for i, dy := range []float32{1, 2, 0.5, 0.5, 1, 0.25, 0.25, 3} {
					p.scrollBy(dy, fmt.Sprintf("scroll %d", i))
					replayed += p.kept.rt.rowsShiftReplayed
				}
				t.Logf("rows replayed shifted: %d", replayed)
				switch name {
				case "plain", "shadow", "dashed border":
					if replayed == 0 {
						t.Errorf("%s rows replayed none shifted", name)
					}
				case "linear gradient", "conic gradient":
					// A gradient is volatile (paintVolatile): never recorded.
					if replayed != 0 {
						t.Errorf("%s rows replayed %d shifted", name, replayed)
					}
				}
			})
		}
	}
}

// Text whose box sits at a fractional y: its baseline is snapped, so it
// shifts by whole pixels alike.
func TestRetainShiftPaintGlyphAtOddY(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		t.Run(fmt.Sprintf("scale %v", scale), func(t *testing.T) {
			p := shiftPair(t, scale, func() func(c *Context) {
				s := auditState()
				return func(c *Context) {
					List(c, s, 120, func(i int) {
						Row(c).Padding(0.25, 3).Children(func() { Textf(c, "row %d", i) })
					}).Grow(1)
				}
			})
			p.scrollBy(1, "first move")
			p.idle(2)
			replayed := 0
			for i := range 6 {
				p.scrollBy(float32(1+i%2), fmt.Sprintf("scroll %d", i))
				replayed += p.kept.rt.rowsShiftReplayed
				for _, g := range p.kept.rt.scene.Glyphs {
					if !integral(g.Y) {
						t.Fatalf("a glyph at y %v: baselines are snapped to device pixels", g.Y)
					}
				}
			}
			if replayed == 0 {
				t.Error("no row replayed shifted")
			}
		})
	}
}

// Rows record at their first painting, not at the second while they stood
// still: a list that scrolls from the start replays within a few frames.
func TestRetainShiftPaintRecordsAtTheFirstPainting(t *testing.T) {
	p := newPair(t, 300, 400, newKeptList(500).mk)
	p.kept.rt.replayCheck = true
	for i := range 6 {
		p.scrollBy(float32(1+i%2), fmt.Sprintf("scroll %d", i))
	}
	if n := p.kept.rt.rowsShiftReplayed; n == 0 {
		t.Errorf("rows that never stood still replayed none shifted")
	}
}

// Every field of an operation or a glyph is either shifted by translate or
// known not to be a window coordinate: a field added to scene.Op or
// scene.Glyph fails here until it is one or the other.
func TestRetainTranslateCoversEveryCoordinateField(t *testing.T) {
	shifted := map[string]bool{"Rect": true, "Cast": true, "Y": true}
	notCoordinates := map[string]bool{
		// Op
		"Kind": true, "Transform": true, "Radii": true, "Continuous": true, "Color": true, "Paint": true, "Color2": true,
		"Wide": true, "Border": true, "BorderColor": true, "Dashed": true, "BorderPaint": true, "Inset": true, "Blur": true,
		"CastRadii": true, "Start": true, "End": true, "Stops": true, "Image": true, "Src": true, "Grayscale": true,
		"Pixelated": true, "Opacity": true,
		// Gradient points: a fill with one is not shiftable (only a solid fill is).
		"Gradient": true,
		// Glyph
		"X": true, "W": true, "H": true, "U": true, "V": true, "UW": true, "VH": true, "Colored": true, "Subpixel": true, "Thin": true,
	}
	for _, ty := range []reflect.Type{reflect.TypeOf(scene.Op{}), reflect.TypeOf(scene.Glyph{})} {
		for i := range ty.NumField() {
			name := ty.Field(i).Name
			if !shifted[name] && !notCoordinates[name] {
				t.Errorf("%s.%s is neither shifted by translate nor listed as not a coordinate", ty.Name(), name)
			}
		}
	}
	// What translate shifts it shifts: a recognisable value in each.
	ops := []scene.Op{
		{Kind: scene.OpFill, Rect: scene.Rect{Y: 10}, Paint: scene.PaintSolid},
		{Kind: scene.OpShadow, Rect: scene.Rect{Y: 10}, Cast: scene.Rect{Y: 20}},
		{Kind: scene.OpPushClip, Rect: scene.Rect{Y: 10}},
	}
	glyphs := []scene.Glyph{{Y: 5}}
	if !shiftable(ops, glyphs) {
		t.Fatal("integer coordinates are not shiftable")
	}
	translate(ops, glyphs, 7)
	if ops[0].Rect.Y != 17 || ops[1].Rect.Y != 17 || ops[1].Cast.Y != 27 || ops[2].Rect.Y != 17 || glyphs[0].Y != 12 {
		t.Errorf("translate: %+v %+v", ops, glyphs)
	}
}

func TestRetainTranslateRefusesWhatItDoesNotKnow(t *testing.T) {
	for name, ops := range map[string][]scene.Op{
		"unknown kind":      {{Kind: 250}},
		"image":             {{Kind: scene.OpImage}},
		"fractional rect":   {{Kind: scene.OpFill, Rect: scene.Rect{Y: 10.5}}},
		"linear gradient":   {{Kind: scene.OpFill, Paint: scene.PaintLinear, Gradient: [4]float32{0, 1, 0, 2}}},
		"conic gradient":    {{Kind: scene.OpFill, Paint: scene.PaintConic, Gradient: [4]float32{0, 1, 0, 2}}},
		"stripes":           {{Kind: scene.OpFill, Paint: scene.PaintStripes}},
		"unknown paint":     {{Kind: scene.OpFill, Paint: 200}},
		"fractional shadow": {{Kind: scene.OpShadow, Cast: scene.Rect{Y: 0.25}}},
	} {
		if shiftable(ops, nil) {
			t.Errorf("%s: shiftable", name)
		}
	}
	if shiftable(nil, []scene.Glyph{{Y: 3.5}}) {
		t.Error("a glyph at a fractional y is shiftable")
	}
}

// The flags a record gets when it is made decide what a move may do with it:
// a record that is not shiftable, or whose row was cut by the clip, is not
// replayed shifted. (The record is spoiled behind the engine's back; what it
// stands for is tested by the tests of shiftable and of the cut rows.)
func TestRetainShiftPaintRecordFlags(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spoil func(r *rowPaint)
	}{
		{"not shiftable", func(r *rowPaint) { r.shiftable = false }},
		{"cut at the recording", func(r *rowPaint) { r.inside = false }},
		// Text on an opaque background is drawn another way (subpixel) than
		// on a transparent one: a row whose box is no longer in the same
		// state of cover is painted, not replayed. (Tester does not draw
		// subpixel text, so the flag is spoiled, as the others.)
		{"cover changed", func(r *rowPaint) { r.covered = !r.covered }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := shiftPair(t, 1, newKeptList(500).mk)
			p.scrollBy(1, "first move")
			p.idle(2)
			n := 0
			for _, st := range p.kept.rt.states {
				if st.replay != nil && st.replay.ok {
					tc.spoil(st.replay)
					n++
				}
			}
			if n == 0 {
				t.Fatal("no record")
			}
			p.scrollBy(1, "scroll")
			if r := p.kept.rt.rowsShiftReplayed; r != 0 {
				t.Errorf("%d rows replayed shifted with a record that %s", r, tc.name)
			}
		})
	}
}

// A row that moved sideways is painted anew: only a shift in Y is replayed.
func TestRetainShiftPaintSidewaysIsRefused(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			List(c, s, 100, func(i int) {
				Row(c).Width(600).Children(func() { Textf(c, "row %d", i); Spacer(c); Text(c, "end") })
			}).Width(600).Grow(1)
		}
	})
	p.kept.rt.replayCheck = true
	p.idle(4)
	for i := range 6 {
		p.both(func(tt *Tester) { tt.Scroll(150, 200, float32(1+i%2), 0) })
		p.compare(fmt.Sprintf("sideways %d", i))
		if n := p.kept.rt.replayMismatch; n != 0 {
			t.Fatalf("sideways %d: %d replayed rows differ from painting them", i, n)
		}
		if n := p.kept.rt.rowsShiftReplayed; n != 0 {
			t.Errorf("sideways %d: %d rows replayed shifted", i, n)
		}
	}
}

// A list that moves sideways (a horizontal scroller around it scrolls) moves
// its rows by dx only, and their clip is the same (the scroller's edge cuts
// it at the same place): their paint is drawn anew, not replayed at the old x.
func TestRetainShiftPaintListMovedSidewaysIsRefused(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := auditState()
		return func(c *Context) {
			ScrollHorizontal(c).Children(func() {
				List(c, s, 100, func(i int) {
					Row(c).Width(600).Children(func() { Textf(c, "row %d", i); Spacer(c); Text(c, "end") })
				}).Width(600)
			})
		}
	})
	p.kept.rt.replayCheck = true
	p.idle(4)
	for i := range 6 {
		p.both(func(tt *Tester) { tt.Scroll(150, 200, float32(1+8*(i%2)), 0) })
		p.compare(fmt.Sprintf("sideways %d", i))
		if n := p.kept.rt.replayMismatch; n != 0 {
			t.Fatalf("sideways %d: %d replayed rows differ from painting them", i, n)
		}
		if n := p.kept.rt.rowsShiftReplayed; n != 0 {
			t.Errorf("sideways %d: %d rows replayed shifted", i, n)
		}
	}
}

// The paint side: a replayed shifted row meets what the frame around it
// changes. Pictures, ops and hits are compared with == after each step
// (sameFrames: the ops of the frame are what the damage is made from), and
// every test says a row did replay shifted where it should.
func shiftedSome(t *testing.T, p *pair, steps int, what string) {
	t.Helper()
	n := 0
	for i := range steps {
		p.scrollBy(float32(1+i%2), fmt.Sprintf("%s %d", what, i))
		n += p.kept.rt.rowsShiftReplayed
	}
	if n == 0 {
		t.Fatalf("%s: no row replayed shifted", what)
	}
}

func TestRetainShiftPaintTextCache(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	p.scrollBy(1, "first move")
	p.idle(2)
	// Far more scrolling than the glyph cache keeps unused: a replay that did
	// not touch its glyphs would draw them from an atlas that dropped them.
	shiftedSome(t, p, 60, "scroll")
	// Every row replayed in the last frame marked its glyphs used in it (the
	// counter has moved on since: one frame ago).
	rt, checked := p.kept.rt, 0
	var walk func(n *node)
	walk = func(n *node) {
		if r := rowRecord(n); r != nil && r.at == rt.layouts {
			checked++
			if !rt.text.UsedAgo(&r.uses, 1) {
				t.Errorf("row %v replayed without marking its glyphs used", n.id)
			}
		}
		for c := n.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(rt.c.root)
	if checked == 0 {
		t.Fatal("no row replayed in the last frame")
	}
}

func rowRecord(n *node) *rowPaint {
	if n.listRow && n.st != nil && n.st.replay != nil && n.st.replay.ok {
		return n.st.replay
	}
	return nil
}

// Dark rows seldom replay at all, in P2 as well (what paints them differs from
// frame to frame in the dark theme, which is not P4b's); the pictures are equal
// in both themes and the light one replays shifted again after the round trip.
func TestRetainShiftPaintAfterThemeChange(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	shiftedSome(t, p, 4, "light")
	p.both(func(tt *Tester) { tt.SetDark(true) })
	p.frame("dark")
	if n := p.kept.rt.rowsShiftReplayed; n != 0 {
		t.Errorf("the theme changed: %d rows replayed", n)
	}
	for i := range 4 {
		p.scrollBy(float32(1+i%2), fmt.Sprintf("dark %d", i))
	}
	p.both(func(tt *Tester) { tt.SetDark(false) })
	p.idle(2)
	shiftedSome(t, p, 4, "light again")
}

func TestRetainShiftPaintAfterWidthChange(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	shiftedSome(t, p, 4, "wide")
	p.both(func(tt *Tester) { tt.SetSize(240, 400) })
	p.frame("narrower")
	if n := p.kept.rt.rowsShiftReplayed; n != 0 {
		t.Errorf("the width changed: %d rows replayed", n)
	}
	shiftedSome(t, p, 4, "narrow")
}

// The row under the pointer is hovered: it is painted, not replayed, and the
// pictures are equal while it moves under a still pointer.
func TestRetainShiftPaintWhenHovered(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	shiftedSome(t, p, 2, "before")
	p.both(func(tt *Tester) { tt.Move(150, 200) })
	p.frame("hover")
	shiftedSome(t, p, 6, "hovered")
}

// With the inspector open nothing is recorded or replayed.
func TestRetainShiftPaintInspectorOpen(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	p.both(func(tt *Tester) { tt.rt.insp.open = true })
	p.frame("inspector")
	n := 0
	for i := range 6 {
		p.both(func(tt *Tester) { tt.Scroll(20, 200, 0, float32(1+i%2)) })
		p.compare(fmt.Sprintf("scroll %d", i))
		n += p.kept.rt.rowsShiftReplayed
	}
	if n != 0 {
		t.Errorf("%d rows replayed shifted with the inspector open", n)
	}
}

// A row left at a fraction of a pixel from its record stands still: it records
// where it stands and replays, as a row does that stood still after any other
// move (without this it painted anew in every frame, idle ones too).
func TestRetainShiftPaintFractionalStandingStillRecords(t *testing.T) {
	p := shiftPair(t, 1, newKeptList(500).mk)
	p.scrollBy(1, "first move")
	p.idle(3)
	p.scrollBy(0.5, "half")
	n := 0
	for i := range 5 {
		p.frame(fmt.Sprintf("idle %d", i))
		n += p.kept.rt.replayed
	}
	if n == 0 {
		t.Fatalf("no row replayed in 5 idle frames after a scroll by half a pixel")
	}
	p.both(func(tt *Tester) { tt.Move(150, 380) })
	p.frame("hover")
	if p.kept.rt.replayed == 0 {
		t.Error("no row replayed in the hover frame")
	}
}

// Rows with padding of fractions of a pixel, shadows and text scrolled by
// fractional, whole and big steps and idle frames, at scales 1, 1.5 and 2: a
// record shifted must keep whole-pixel coordinates (replayCheck counts a
// record that does not), and every picture is equal to painting the rows. (At
// scale 1.5 a product dy*scale added in place was fused with the add on arm64
// and left 119.00001 for 119 in the record.)
func TestRetainShiftPaintMixedScales(t *testing.T) {
	// Under the race detector (which makes it 30 times slower, 537 s of the 1036 s
	// of the package, 45 s with two seeds) a few seeds and steps: the same shapes, fewer of them.
	seeds, steps := uint64(12), 200
	if raceEnabled {
		seeds, steps = 1, 100
	}
	for _, scale := range []float32{1, 1.5, 2} {
		for seed := range seeds {
			t.Run(fmt.Sprintf("scale %v seed %d", scale, seed), func(t *testing.T) {
				rng := rand.New(rand.NewPCG(seed, 9))
				p := newPair(t, 300, 900, func() func(c *Context) {
					s := auditState()
					return func(c *Context) {
						List(c, s, 3000, func(i int) {
							Row(c).Padding(float32(i%3)/3, 3).Background(RGB(220, 230, 240)).Shadow(0, 1, 2, 0, RGB(100, 100, 100)).Children(func() { Textf(c, "row %d", i) })
						}).Grow(1)
					}
				})
				p.both(func(tt *Tester) { tt.SetScale(scale) })
				p.kept.rt.replayCheck = true
				p.idle(3)
				p.scrolled(2000, "down")
				shifted := 0
				fracs := []float32{0.5, 1.0 / 3, 0.25, 0.1, 0.7, 2.0 / 3, 0.5 - 1.0/65536}
				for i := range steps {
					switch rng.IntN(6) {
					case 0, 1:
						dy := float32(rng.IntN(5)-2) * fracs[rng.IntN(len(fracs))]
						if dy == 0 {
							dy = 0.5
						}
						p.scrollBy(dy, fmt.Sprintf("fractional %d %v", i, dy))
					case 2, 3:
						dy := float32(rng.IntN(9) - 4)
						if dy == 0 {
							dy = 1
						}
						p.scrollBy(dy/scale, fmt.Sprintf("whole %d %v", i, dy/scale))
					case 4:
						dy := float32(rng.IntN(400) - 200)
						if dy == 0 {
							dy = 150
						}
						p.scrollBy(dy, fmt.Sprintf("big %d %v", i, dy))
					default:
						p.frame(fmt.Sprintf("idle %d", i))
					}
					shifted += p.kept.rt.rowsShiftReplayed
				}
				if shifted == 0 {
					t.Error("no row replayed shifted")
				}
			})
		}
	}
}
