package scene

import (
	"image"
	"math/rand/v2"
	"testing"
)

func areaOf(rs []image.Rectangle) int {
	n := 0
	for _, r := range rs {
		n += r.Dx() * r.Dy()
	}
	return n
}

// tableScene is the lines of a table: for each of rows, a background, a
// check box and two texts' boxes, wrapped in a clip every fifth.
func tableScene(rows int) *Scene {
	s := &Scene{Width: 1280, Height: 800}
	for r := 0; r < rows; r++ {
		y := float32(40 + r*32)
		if r%5 == 0 {
			s.Ops = append(s.Ops, Op{Kind: OpPushClip, Rect: Rect{X: 9, Y: y, W: 1262, H: 32}})
		}
		s.Ops = append(s.Ops,
			Op{Kind: OpFill, Rect: Rect{X: 9, Y: y, W: 1262, H: 32}, Color: Color{R: uint8(r), A: 255}},
			Op{Kind: OpFill, Rect: Rect{X: 17, Y: y + 8, W: 18, H: 18}, Color: Color{G: 120, A: 255}},
			Op{Kind: OpFill, Rect: Rect{X: 41, Y: y + 4, W: 300, H: 24}, Color: Color{B: uint8(r), A: 255}},
			Op{Kind: OpFill, Rect: Rect{X: 341, Y: y + 4, W: 100, H: 24}, Color: Color{R: 90, G: uint8(r), A: 255}},
		)
		if r%5 == 0 {
			s.Ops = append(s.Ops, Op{Kind: OpPopClip})
		}
	}
	return s
}

func insertOp(s *Scene, at int, op Op) *Scene {
	c := *s
	c.Ops = append(append(append([]Op(nil), s.Ops[:at]...), op), s.Ops[at:]...)
	return &c
}

// TestDamageOfARowPressedAndAScrollbarShown is the press of a check box in a
// table of the ui package: the frame inserts the background of the row pressed
// in the middle of the display list and a scrollbar at its end. The first
// version of Diff matched one start and one end, and so damaged everything
// between the two (824,294 pixels of 1,024,000 on a 1280×800 window, in the
// table of the ui package); it now damages the row and the scrollbar.
func TestDamageOfARowPressedAndAScrollbarShown(t *testing.T) {
	before := tableScene(18)
	row := Rect{X: 9, Y: 40 + 3*32, W: 1262, H: 32}
	// After the background and the clip of the row.
	at := 4*3 + 2 + 1
	pressed := insertOp(before, at, Op{Kind: OpFill, Rect: row, Color: Color{R: 200, G: 200, B: 200, A: 255}})
	bar := Rect{X: 1260, Y: 44, W: 8, H: 26}
	after := insertOp(pressed, len(pressed.Ops), Op{Kind: OpFill, Rect: bar, Color: Color{R: 100, G: 100, B: 100, A: 255}})
	d := Differ{Reach: 1}
	d.Diff(before)
	d.Remember(before)
	damage, ok := d.Diff(after)
	if !ok {
		t.Fatal("no damage found")
	}
	if got, limit := areaOf(damage), 45000; got > limit {
		t.Errorf("damage %v is %d pixels, want the row and the scrollbar: at most %d", damage, got, limit)
	}
	var hitRow, hitBar bool
	for _, r := range damage {
		hitRow = hitRow || row.Bounds().In(r)
		hitBar = hitBar || bar.Bounds().In(r)
	}
	if !hitRow || !hitBar {
		t.Errorf("damage %v misses the row (%v) or the scrollbar (%v)", damage, hitRow, hitBar)
	}
	if !d.multi {
		t.Error("the ops were not matched in several ranges")
	}
}

// Bounds is the box of r in whole pixels.
func (r Rect) Bounds() image.Rectangle {
	return image.Rect(int(r.X), int(r.Y), int(r.X+r.W), int(r.Y+r.H))
}

// TestMatchOfSeveralRanges checks Match after a Diff that found the ops
// in several ranges: an op that Match finds is the op of the last scene
// that is the same and draws where it did, or else one that changed whose
// bounds are in the damage; an op it does not find was inserted and its
// bounds are in the damage.
func TestMatchOfSeveralRanges(t *testing.T) {
	before := tableScene(18)
	after := insertOp(insertOp(before, 20, Op{Kind: OpFill, Rect: Rect{X: 9, Y: 100, W: 1262, H: 32}, Color: Color{R: 200, A: 255}}),
		len(before.Ops)+1-3, Op{Kind: OpFill, Rect: Rect{X: 1260, Y: 44, W: 8, H: 26}, Color: Color{A: 255}})
	d := Differ{Reach: 1}
	d.Diff(before)
	d.Remember(before)
	oldBounds := append([]image.Rectangle(nil), d.bounds...)
	damage, ok := d.Diff(after)
	if !ok || !d.multi {
		t.Fatalf("ok %v, multi %v", ok, d.multi)
	}
	inDamage := func(r image.Rectangle) bool {
		for _, c := range damage {
			if r.In(c) {
				return true
			}
		}
		return r.Empty()
	}
	inserted := 0
	for j := range after.Ops {
		i, kept := d.Match(j)
		if !kept {
			inserted++
			if !inDamage(d.next[j]) {
				t.Errorf("op %d was inserted and its bounds %v are not in the damage %v", j, d.next[j], damage)
			}
			continue
		}
		same := d.same(i, after, j, nil, nil) && oldBounds[i] == d.next[j]
		if !same && !(inDamage(oldBounds[i]) && inDamage(d.next[j])) {
			t.Errorf("op %d stands for op %d, which is neither the same nor in the damage", j, i)
		}
		if same && after.Ops[j].Kind != before.Ops[i].Kind {
			t.Errorf("op %d stands for op %d of another kind", j, i)
		}
	}
	if inserted != 2 {
		t.Errorf("%d ops without a match, want the 2 inserted", inserted)
	}
}

// randomScene returns n ops of fills, shadows and clips in pairs, some under
// a transform.
func randomScene(r *rand.Rand, n int) *Scene {
	s := &Scene{Width: 200, Height: 150}
	for len(s.Ops) < n {
		s.Ops = append(s.Ops, randomOps(r)...)
	}
	return s
}

// randomOps returns an op, or a clip with some ops in it.
func randomOps(r *rand.Rand) []Op {
	rect := Rect{X: float32(r.IntN(180)), Y: float32(r.IntN(130)), W: float32(2 + r.IntN(60)), H: float32(2 + r.IntN(40))}
	switch r.IntN(6) {
	case 0:
		return []Op{{Kind: OpShadow, Rect: rect, Blur: float32(r.IntN(8)), Color: Color{A: 90}}}
	case 1:
		push := Op{Kind: OpPushClip, Rect: rect, Radii: Uniform(float32(r.IntN(6)))}
		if r.IntN(3) == 0 {
			push.Transform = Translation(float32(r.IntN(10)), float32(r.IntN(10)))
		}
		ops := []Op{push}
		for range 1 + r.IntN(3) {
			ops = append(ops, randomOps(r)...)
		}
		return append(ops, Op{Kind: OpPopClip})
	case 2:
		return []Op{{Kind: OpHole, Rect: rect}}
	}
	return []Op{{Kind: OpFill, Rect: rect, Color: Color{R: uint8(r.IntN(256)), G: uint8(r.IntN(256)), A: 255}}}
}

// TestDamageOfSeveralChangesIsInsideTheOldGap checks, over random pairs of
// scenes that differ in one to three places (ops inserted, deleted and changed,
// a clip put around ops or taken off them), that when Diff finds the ops in
// several ranges its damage lies inside the single range it found before, the
// bounds of everything between the shared start and end, and that Diff
// still finds damage whenever the scenes differ. (That the pixels are right is
// the property test of the raster package, whose seeds are the same ones.)
func TestDamageOfSeveralChangesIsInsideTheOldGap(t *testing.T) {
	multi, runs := 0, 0
	for seed := range uint64(2000) {
		r := rand.New(rand.NewPCG(seed, 11))
		before := randomScene(r, 20+r.IntN(30))
		after := &Scene{Width: before.Width, Height: before.Height, Ops: append([]Op(nil), before.Ops...)}
		for range 1 + r.IntN(3) {
			at := r.IntN(len(after.Ops) + 1)
			switch r.IntN(3) {
			case 0:
				after.Ops = append(after.Ops[:at], append(randomOps(r), after.Ops[at:]...)...)
			case 1:
				if at < len(after.Ops) && after.Ops[at].Kind != OpPushClip && after.Ops[at].Kind != OpPopClip {
					after.Ops = append(after.Ops[:at], after.Ops[at+1:]...)
				}
			default:
				if at < len(after.Ops) && after.Ops[at].Kind != OpPushClip && after.Ops[at].Kind != OpPopClip {
					after.Ops[at].Color = Color{R: uint8(r.IntN(256)), A: 255}
					after.Ops[at].Rect.X += 3
				}
			}
		}
		d := Differ{Reach: 1}
		d.Diff(before)
		d.Remember(before)
		damage, ok := d.Diff(after)
		if !ok {
			t.Fatalf("seed %d: no damage found", seed)
		}
		runs++
		if !d.multi {
			continue
		}
		multi++
		// What a single range damages: the bounds of the ops between the
		// shared start and end of the scenes, in either.
		var gap image.Rectangle
		for i := d.pre; i < d.n-d.suf; i++ {
			gap = gap.Union(d.bounds[i])
		}
		for j := d.pre; j < d.m-d.suf; j++ {
			gap = gap.Union(d.next[j])
		}
		for _, c := range damage {
			if !c.In(gap) {
				t.Fatalf("seed %d: damage %v is not inside the old gap %v", seed, c, gap)
			}
		}
	}
	t.Logf("%d of %d random pairs of scenes (seeds 0..1999) were matched in several ranges", multi, runs)
	if multi < 100 {
		t.Errorf("only %d pairs were matched in several ranges: the generator does not reach the code", multi)
	}
}

// A clip that cannot be matched in a few tries, or ops that do not hold their
// clips in pairs, leave the damage as the single range had it.
func TestDamageOfUnbalancedChangesIsTheOldGap(t *testing.T) {
	before := tableScene(10)
	// Remove the clip of the first row (its push and its pop) and add an
	// overlay at the end.
	after := &Scene{Width: before.Width, Height: before.Height}
	for i, op := range before.Ops {
		if i == 0 || i == 5 {
			continue // the PushClip and the PopClip of row 0
		}
		after.Ops = append(after.Ops, op)
	}
	after.Ops = append(after.Ops, Op{Kind: OpFill, Rect: Rect{X: 1260, Y: 44, W: 8, H: 26}, Color: Color{A: 255}})
	d := Differ{Reach: 1}
	d.Diff(before)
	d.Remember(before)
	damage, ok := d.Diff(after)
	if !ok || len(damage) == 0 {
		t.Fatalf("ok %v, damage %v", ok, damage)
	}
	// The row's fills draw as before unless the clip cut them: they are
	// inside the clip rect, so none moved; the overlay is new.
	hit := false
	for _, c := range damage {
		hit = hit || image.Rect(1260, 44, 1268, 70).In(c)
	}
	if !hit {
		t.Errorf("damage %v misses the overlay", damage)
	}
}

// Ops that are the same draw where they did only in the same clips: a clip
// pushed and never popped, taken off, leaves the ops it cut drawn whole, and
// where they now draw is damaged (the bounds of an op count within its clips).
func TestDamageWhereAClipWasTakenOff(t *testing.T) {
	cut := Rect{X: 10, Y: 10, W: 30, H: 30}
	a := Op{Kind: OpFill, Rect: Rect{X: 20, Y: 20, W: 100, H: 10}, Color: Color{R: 255, A: 255}}
	b := Op{Kind: OpFill, Rect: Rect{X: 20, Y: 60, W: 100, H: 10}, Color: Color{G: 255, A: 255}}
	before := &Scene{Width: 200, Height: 150, Ops: []Op{{Kind: OpPushClip, Rect: cut}, a, b}}
	after := &Scene{Width: 200, Height: 150, Ops: []Op{a, b}}
	d := Differ{Reach: 1}
	d.Diff(before)
	d.Remember(before)
	damage, ok := d.Diff(after)
	if !ok {
		t.Fatal("no damage found")
	}
	for _, want := range []image.Rectangle{image.Rect(20, 20, 120, 30), image.Rect(20, 60, 120, 70)} {
		hit := false
		for _, c := range damage {
			hit = hit || want.In(c)
		}
		if !hit {
			t.Errorf("damage %v misses %v, drawn now that the clip is gone", damage, want)
		}
	}
}
