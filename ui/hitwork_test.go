package ui

import (
	"math/rand/v2"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

func hitWorkView() func(c *Context) {
	s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
	return func(c *Context) {
		Column(c).Fill().Children(func() {
			List(c, s, 200, func(i int) { Row(c).Children(func() { Textf(c, "row %d", i); Text(c, "·") }) }).Grow(1)
		})
	}
}

// A frame with the pointer outside the window tests no hit past the window
// (rehover walked the whole list of hits, an inverse each), and one with the
// pointer inside tests some.
func TestContainsTestsNothingForAPointerOutsideTheWindow(t *testing.T) {
	tt := NewTester(hitWorkView(), 300, 400)
	for range 4 {
		tt.Frame()
	}
	tt.Move(-50, -50)
	tt.Frame()
	before := tt.rt.hitWork
	for range 5 {
		tt.Frame()
	}
	if n := tt.rt.hitWork - before; n != 0 {
		t.Errorf("the pointer outside the window: %d hits tested in 5 frames, want 0", n)
	}
	tt.Move(150, 200)
	before = tt.rt.hitWork
	tt.Frame()
	if tt.rt.hitWork == before {
		t.Error("the pointer inside the window tested no hit")
	}
}

// inversePoint is the inverse's Point for every map, with == on both
// coordinates (a zero of the other sign is equal), and takes the shortcut for the
// maps that only move.
func TestInversePointMatchesTheInverse(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	val := func() float32 { return float32(rng.IntN(2001)-1000) / float32(1+rng.IntN(7)) }
	for i := range 20000 {
		var m scene.Affine
		switch i % 4 {
		case 0: // identity
		case 1: // a move
			m = scene.Matrix(1, 0, 0, 1, val(), val())
		case 2: // a scale and a move
			m = scene.Matrix(1+val()/100, 0, 0, 1+val()/100, val(), val())
		default: // a rotation
			m = scene.Matrix(0.8, 0.6, -0.6, 0.8, val(), val())
		}
		x, y := val(), val()
		inv, ok := m.Inverse()
		gx, gy, gok := inversePoint(m, x, y)
		if gok != ok {
			t.Fatalf("%+v: ok %v, want %v", m, gok, ok)
		}
		if ok {
			wx, wy := inv.Point(x, y)
			if gx != wx || gy != wy {
				t.Fatalf("%+v at %v,%v: %v,%v, want %v,%v", m, x, y, gx, gy, wx, wy)
			}
		}
	}
}

// forgetInput visits the states the pass built, not every state: with a
// thousand states that nothing built it visits none of them, and it visits all
// that are stamped as seen.
func TestForgetInputVisitsOnlyWhatThePassBuilt(t *testing.T) {
	tt := NewTester(hitWorkView(), 300, 400)
	for range 4 {
		tt.Frame()
	}
	rt := tt.rt
	built := len(rt.states)
	for i := range 1000 {
		id := uint64(1<<60 + i)
		rt.states[id] = &state{id: id, seen: 1}
	}
	before := rt.swept
	tt.Frame()
	visited := rt.swept - before
	// A call of forgetInput a pass (three at most) and one a late row: the
	// states of the view, a few times, and not the thousand more.
	if visited > 4*built {
		t.Errorf("visited %d states in a frame, the view has %d and a thousand more were not built", visited, built)
	}
	if visited == 0 {
		t.Error("visited no state")
	}
	listed := map[*state]bool{}
	for _, s := range rt.built {
		listed[s] = true
	}
	for _, s := range rt.states {
		if s.seen == rt.frame && s.pass == rt.pass && !listed[s] {
			t.Errorf("state %d is stamped as seen by the last pass and not listed", s.id)
		}
	}
}
