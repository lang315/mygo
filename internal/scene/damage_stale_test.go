package scene

import "testing"

// When Diff finds that everything may have changed (another size), Match does
// not answer with the ranges of the frame before.
func TestMatchAfterADiffOfEverything(t *testing.T) {
	before := tableScene(18)
	after := insertOp(insertOp(before, 20, Op{Kind: OpFill, Rect: Rect{X: 9, Y: 100, W: 1262, H: 32}, Color: Color{R: 200, A: 255}}),
		len(before.Ops)+1-3, Op{Kind: OpFill, Rect: Rect{X: 1260, Y: 44, W: 8, H: 26}, Color: Color{A: 255}})
	d := Differ{Reach: 1}
	d.Diff(before)
	d.Remember(before)
	if _, ok := d.Diff(after); !ok || !d.multi {
		t.Fatalf("setup: ok %v, multi %v", ok, d.multi)
	}
	d.Remember(after)
	resized := *after
	resized.Width++
	if _, ok := d.Diff(&resized); ok {
		t.Fatal("a scene of another size was diffed")
	}
	for j := range resized.Ops {
		if i, kept := d.Match(j); kept {
			t.Fatalf("Match(%d) = %d after a Diff that found everything changed", j, i)
		}
	}
}
