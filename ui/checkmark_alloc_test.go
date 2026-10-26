package ui

import "testing"

// framesAllocating returns what a frame allocates, on average, for 60 check
// boxes all in the state on, beside a box that changes color every frame so that
// the boxes are painted again.
func framesAllocating(on bool) float64 {
	states := make([]bool, 60)
	for i := range states {
		states[i] = on
	}
	n := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreBox(c).Size(10, 10).Background(Color{R: uint8(n), A: 255})
			for i := range states {
				coreCheckbox(c, &states[i], "")
			}
		})
	}, 300, 2400)
	tt.Frame()
	return testing.AllocsPerRun(50, func() { n++; tt.Frame() })
}

// A check mark is painted from a path the engine keeps, not one made for each
// check box in every frame: a frame of 60 checked boxes allocates what a frame
// of 60 unchecked ones does, give or take the boxes' own, not three slices more
// for each.
func TestCheckmarkAllocatesNothingPerBox(t *testing.T) {
	checked, unchecked := framesAllocating(true), framesAllocating(false)
	if extra := checked - unchecked; extra > 20 {
		t.Fatalf("a frame of 60 checked boxes allocates %.0f, unchecked %.0f: %.0f more", checked, unchecked, extra)
	}
}

// checkPath empties the path it is given, and makes the same mark in it each
// time.
func TestCheckPathResetsItsPath(t *testing.T) {
	var p Path
	r := Rect{10, 20, 16, 16}
	checkPath(&p, r)
	first := append([]pathCmd(nil), p.cmds...)
	checkPath(&p, r)
	if len(p.cmds) != 3 || len(first) != 3 {
		t.Fatalf("the mark has %d commands the second time, %d the first, want 3", len(p.cmds), len(first))
	}
	for i := range first {
		if first[i] != p.cmds[i] {
			t.Fatalf("command %d changed: %v then %v", i, first[i], p.cmds[i])
		}
	}
}
