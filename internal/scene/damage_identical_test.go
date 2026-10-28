package scene

import (
	"image"
	"slices"
	"testing"
)

// identicalScene is a scene with every kind of thing Diff compares: fills, a
// clip around mask glyphs, a gradient with stops, an image and a wide
// color.
func identicalScene() (*Scene, *Image) {
	img := NewImageRGBA(2, 2, make([]byte, 16))
	s := &Scene{Width: 400, Height: 300, MaskAtlas: NewAtlas(1, 64, 64)}
	s.MaskAtlas.Put(0, 0, 8, 8, make([]byte, 64), 8)
	s.Glyphs = []Glyph{
		{X: 12, Y: 12, W: 8, H: 8, U: 0, V: 0, UW: 8, VH: 8, Color: Color{A: 255}},
		{X: 22, Y: 12, W: 8, H: 8, U: 0, V: 0, UW: 8, VH: 8, Color: Color{A: 255}},
		{X: 32, Y: 12, W: 8, H: 8, U: 0, V: 0, UW: 8, VH: 8, Color: Color{A: 255}},
	}
	s.Stops = []Stop{{Color: Color{R: 255, A: 255}, At: 0}, {Color: Color{B: 255, A: 255}, At: 1}}
	s.Wide = []WideColors{{}}
	s.Ops = []Op{
		{Kind: OpFill, Rect: Rect{X: 0, Y: 0, W: 400, H: 300}, Color: Color{R: 240, G: 240, B: 240, A: 255}},
		{Kind: OpPushClip, Rect: Rect{X: 10, Y: 10, W: 100, H: 20}},
		{Kind: OpGlyphs, Start: 0, End: 3},
		{Kind: OpPopClip},
		{Kind: OpFill, Rect: Rect{X: 10, Y: 40, W: 100, H: 20}, Paint: PaintLinearStops, Stops: [2]int32{0, 2}, Gradient: [4]float32{10, 40, 110, 40}},
		{Kind: OpImage, Rect: Rect{X: 10, Y: 70, W: 20, H: 20}, Image: img, Src: Rect{W: 2, H: 2}},
		{Kind: OpFill, Rect: Rect{X: 10, Y: 100, W: 30, H: 30}, Color: Color{R: 200, A: 255}, Wide: 1},
	}
	return s, img
}

// copyOf returns a scene of its own tables, to change.
func copyOf(s *Scene) *Scene {
	c := *s
	c.Ops = slices.Clone(s.Ops)
	c.Glyphs = slices.Clone(s.Glyphs)
	c.Stops = slices.Clone(s.Stops)
	c.Wide = slices.Clone(s.Wide)
	return &c
}

// diffs are what a Differ answers for a scene: all of it that a renderer
// reads.
type diffs struct {
	damage         []image.Rectangle
	ok             bool
	bounds         []image.Rectangle
	match          [][2]int
	pre, suf, n, m int
	pairs, multi   bool
}

func answer(d *Differ, s *Scene) diffs {
	damage, ok := d.Diff(s)
	a := diffs{damage: slices.Clone(damage), ok: ok, bounds: slices.Clone(d.Bounds()),
		pre: d.pre, suf: d.suf, n: d.n, m: d.m, pairs: d.pairs, multi: d.multi}
	for j := range s.Ops {
		i, kept := d.Match(j)
		if !kept {
			i = -1
		}
		a.match = append(a.match, [2]int{j, i})
	}
	return a
}

func (a diffs) same(b diffs) bool {
	return slices.Equal(a.damage, b.damage) && a.ok == b.ok && slices.Equal(a.bounds, b.bounds) &&
		slices.Equal(a.match, b.match) && a.pre == b.pre && a.suf == b.suf && a.n == b.n && a.m == b.m &&
		a.pairs == b.pairs && a.multi == b.multi
}

// A scene identical to the last one in everything Diff reads takes the fast
// path (counted), and every change leaves it: the answers are the same as
// the full comparison gives, in the damage, the bounds and what Match says.
func TestDiffIdenticalScene(t *testing.T) {
	for _, tc := range []struct {
		name string
		fast int // fast paths taken by the second Diff
		do   func(s *Scene, img *Image) *Scene
		set  func(d *Differ)
	}{
		{"identical", 1, func(s *Scene, _ *Image) *Scene { return copyOf(s) }, nil},
		{"one op changed", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Ops[6].Color.R++
			return c
		}, nil},
		{"one op moved", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Ops[6].Rect.X += 50
			return c
		}, nil},
		{"glyph table only", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Glyphs[1].Color = Color{R: 255, A: 255}
			return c
		}, nil},
		{"glyph position only", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Glyphs[2].X += 3
			return c
		}, nil},
		{"glyph moved to another op's range", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Ops[2].End = 2
			return c
		}, nil},
		{"stops only", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Stops[1].Color = Color{G: 255, A: 255}
			return c
		}, nil},
		{"wide only", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Wide[0].Color = [4]float32{1.5, 0, 0, 1}
			return c
		}, func(d *Differ) { d.Wide = true }},
		{"atlas pixels under a glyph", 0, func(s *Scene, _ *Image) *Scene {
			s.MaskAtlas.Put(2, 2, 2, 2, []byte{255, 255, 255, 255}, 2)
			return copyOf(s)
		}, nil},
		{"atlas pixels elsewhere", 0, func(s *Scene, _ *Image) *Scene {
			s.MaskAtlas.Put(40, 40, 2, 2, []byte{255, 255, 255, 255}, 2)
			return copyOf(s)
		}, nil},
		{"atlas reset", 0, func(s *Scene, _ *Image) *Scene {
			s.MaskAtlas.Reset()
			return copyOf(s)
		}, nil},
		{"another atlas", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.MaskAtlas = NewAtlas(1, 64, 64)
			return c
		}, nil},
		{"clip rect", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Ops[1].Rect.W = 20
			return c
		}, nil},
		{"clip transform", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Ops[1].Transform = Rotation(5)
			return c
		}, nil},
		{"image pixels", 0, func(s *Scene, img *Image) *Scene {
			img.Changed()
			return copyOf(s)
		}, nil},
		{"reach", 0, func(s *Scene, _ *Image) *Scene { return copyOf(s) }, func(d *Differ) { d.Reach = 4 }},
		{"size", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Width++
			return c
		}, nil},
		{"clear color", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Clear = Color{R: 1, A: 255}
			return c
		}, nil},
		{"an op appended", 0, func(s *Scene, _ *Image) *Scene {
			c := copyOf(s)
			c.Ops = append(c.Ops, Op{Kind: OpFill, Rect: Rect{X: 300, Y: 200, W: 10, H: 10}, Color: Color{A: 255}})
			return c
		}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, img := identicalScene()
			fast, full := Differ{Reach: 1}, Differ{Reach: 1, noFast: true}
			for _, d := range []*Differ{&fast, &full} {
				if tc.set != nil && tc.name == "wide only" {
					tc.set(d)
				}
				d.Diff(s)
				d.Remember(s)
			}
			next := tc.do(s, img)
			for _, d := range []*Differ{&fast, &full} {
				if tc.set != nil && tc.name != "wide only" {
					tc.set(d)
				}
			}
			got, want := answer(&fast, next), answer(&full, next)
			if !got.same(want) {
				t.Errorf("with the fast path: %+v\nwithout it:            %+v", got, want)
			}
			if fast.fast != tc.fast {
				t.Errorf("%d scenes took the fast path, want %d", fast.fast, tc.fast)
			}
			if full.fast != 0 {
				t.Errorf("a Differ with the fast path off took it %d times", full.fast)
			}
			// What follows the answer follows it: the next frame, changed in
			// one op, gives the same either way.
			fast.Remember(next)
			full.Remember(next)
			after := copyOf(next)
			if len(after.Ops) > 6 {
				after.Ops[6].Color.G += 9
			}
			if got, want := answer(&fast, after), answer(&full, after); !got.same(want) {
				t.Errorf("a frame later, with the fast path: %+v\nwithout it:            %+v", got, want)
			}
		})
	}
}

// Diffing an identical scene allocates nothing.
func TestDiffIdenticalSceneAllocatesNothing(t *testing.T) {
	s, _ := identicalScene()
	d := Differ{Reach: 1}
	d.Diff(s)
	d.Remember(s)
	same := copyOf(s)
	d.Diff(same) // sizes the buffers
	d.Remember(same)
	n := testing.AllocsPerRun(100, func() {
		d.Diff(same)
		d.Remember(same)
	})
	if n != 0 {
		t.Fatalf("Diff and Remember of an identical scene allocate %.0f times", n)
	}
	if d.fast < 100 {
		t.Fatalf("%d fast paths", d.fast)
	}
}
