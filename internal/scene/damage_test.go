package scene

import (
	"image"
	"testing"
)

// TestDamageOfNestedTransformedClip checks that a clip under a
// transform, nested in a rounded one, changes all the rounded one cuts as
// it comes and goes: the GPU's renderers then cut the outer clip with a
// mask rather than in the shader, at its edges a unit in the last place
// apart.
func TestDamageOfNestedTransformedClip(t *testing.T) {
	outer := Op{Kind: OpPushClip, Rect: Rect{X: 10, Y: 10, W: 150, H: 150}, Radii: Uniform(8)}
	corner := Op{Kind: OpFill, Rect: Rect{X: 10, Y: 10, W: 20, H: 20}, Color: Color{R: 255, A: 255}}
	nested := []Op{
		{Kind: OpPushClip, Rect: Rect{X: 100, Y: 100, W: 40, H: 40}, Transform: Rotation(10)},
		{Kind: OpFill, Rect: Rect{X: 100, Y: 100, W: 20, H: 20}, Color: Color{G: 255, A: 255}},
		{Kind: OpPopClip},
	}
	without := &Scene{Width: 200, Height: 200, Ops: []Op{outer, corner, {Kind: OpPopClip}}}
	with := &Scene{Width: 200, Height: 200, Ops: append(append([]Op{outer, corner}, nested...), Op{Kind: OpPopClip})}
	for _, frames := range [][2]*Scene{{without, with}, {with, without}} {
		d := Differ{Reach: 2}
		d.Diff(frames[0])
		d.Remember(frames[0])
		damage, ok := d.Diff(frames[1])
		if !ok {
			t.Fatal("no damage found")
		}
		// The fill in the outer clip's corner, which the inner clip
		// does not reach.
		at := image.Rect(10, 10, 30, 30)
		hit := false
		for _, r := range damage {
			hit = hit || at.In(r)
		}
		if !hit {
			t.Errorf("%d to %d ops: damage %v misses %v", len(frames[0].Ops), len(frames[1].Ops), damage, at)
		}
	}
}

// TestDamageOfClipOnlyMoved checks that a clip pushed under a translation
// alone is no transform for the clips around it, as the GPU's renderers
// have it (MovedClip): with the only turned op in a rounded clip gone,
// they cut it in the shader rather than with a mask, and all it cuts
// changes at its edge, though the translated clip in it stays.
func TestDamageOfClipOnlyMoved(t *testing.T) {
	outer := Op{Kind: OpPushClip, Rect: Rect{X: 10, Y: 10, W: 150, H: 150}, Radii: Uniform(8)}
	corner := Op{Kind: OpFill, Rect: Rect{X: 10, Y: 10, W: 20, H: 20}, Color: Color{R: 255, A: 255}}
	turned := Op{Kind: OpFill, Rect: Rect{X: 100, Y: 20, W: 20, H: 20}, Transform: Rotation(10), Color: Color{B: 255, A: 255}}
	moved := []Op{
		{Kind: OpPushClip, Rect: Rect{W: 40, H: 40}, Transform: Translation(100, 100)},
		{Kind: OpFill, Rect: Rect{X: 100, Y: 100, W: 20, H: 20}, Color: Color{G: 255, A: 255}},
		{Kind: OpPopClip},
	}
	with := &Scene{Width: 200, Height: 200, Ops: append(append([]Op{outer, corner, turned}, moved...), Op{Kind: OpPopClip})}
	without := &Scene{Width: 200, Height: 200, Ops: append(append([]Op{outer, corner}, moved...), Op{Kind: OpPopClip})}
	for _, frames := range [][2]*Scene{{with, without}, {without, with}} {
		d := Differ{Reach: 2}
		d.Diff(frames[0])
		d.Remember(frames[0])
		damage, ok := d.Diff(frames[1])
		if !ok {
			t.Fatal("no damage found")
		}
		at := image.Rect(10, 10, 30, 30)
		hit := false
		for _, r := range damage {
			hit = hit || at.In(r)
		}
		if !hit {
			t.Errorf("%d to %d ops: damage %v misses %v", len(frames[0].Ops), len(frames[1].Ops), damage, at)
		}
	}
}
