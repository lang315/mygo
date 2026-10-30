package ui

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
)

// A change of the system's appearance or of the desktop's preferences reaches
// the rows kept across frames: they are built again with the new theme.
func TestRetainRowsFollowSystemTheme(t *testing.T) {
	for _, ctrl := range []string{"switch", "checkbox", "button"} {
		for _, pref := range []string{"accent", "hc", "reduce", "textscale", "sysdark"} {
			t.Run(ctrl+"/"+pref, func(t *testing.T) {
				const n = 10
				var on [2][]bool
				on[0], on[1] = make([]bool, n), make([]bool, n)
				for i := range n {
					on[0][i], on[1][i] = i%2 == 0, i%2 == 0
				}
				twin := 0
				p := newPair(t, 300, 300, func() func(c *Context) {
					me := twin
					twin++
					s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
					return func(c *Context) {
						Column(c).Fill().Children(func() {
							List(c, s, n, func(i int) {
								Row(c).Padding(4).Gap(8).Children(func() {
									switch ctrl {
									case "switch":
										Switch(c, &on[me][i])
									case "checkbox":
										Checkbox(c, &on[me][i], "c")
									default:
										Button(c, "b")
									}
								})
							}).Grow(1)
						})
					}
				})
				p.idle(4)
				pr := Preferences{}
				switch pref {
				case "accent":
					pr.Accent = RGB(200, 30, 90)
				case "hc":
					pr.HighContrast = true
				case "textscale":
					pr.TextScale = 1.5
				default:
					pr.ReduceMotion = true
				}
				if pref == "sysdark" {
					p.both(func(tt *Tester) { tt.SetDark(true) })
				} else {
					p.both(func(tt *Tester) { tt.SetPreferences(pr) })
				}
				for range 3 {
					p.both(func(tt *Tester) { tt.Advance(50 * time.Millisecond); tt.Frame() })
				}
				p.compare("system theme, 3 frames later")
			})
		}
	}
}

// A frame of two passes: the first builds row 3 with its switch on (the knob
// starts to move), the second with it off again, as the row was kept. The
// second pass must not keep the row while the knob's channel moves: a kept twin
// would then jump the knob to on, without motion, when the app turns it on later.
func TestRetainRowsTwoPassFlip(t *testing.T) {
	const n = 8
	var on [2][]bool
	var flips [2]int
	on[0], on[1] = make([]bool, n), make([]bool, n)
	twin := 0
	p := newPair(t, 300, 300, func() func(c *Context) {
		me := twin
		twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			if flips[me] > 0 {
				flips[me]--
				on[me][3] = !on[me][3]
				if flips[me] == 1 {
					c.build().rt.consumed = true // another pass, as an event handled while building asks
				}
			}
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Children(func() { Switch(c, &on[me][i]) })
				}).Grow(1)
			})
		}
	})
	p.idle(4)
	flips = [2]int{2, 2}
	p.frame("flip and back in one frame")
	if flips != [2]int{} {
		t.Fatalf("flips left %v: the frame did not run two passes", flips)
	}
	p.both(func(tt *Tester) { tt.Advance(500 * time.Millisecond) })
	p.compare("later")
	for k := range 2 {
		on[k][3] = true
	}
	p.frame("app turns 3 on")
	p.both(func(tt *Tester) { tt.Advance(20 * time.Millisecond) })
	p.compare("20ms on")
}

// systemTheme changes the system's appearance and preferences now and then
// in a random walk, from a stream of its own so the walk's other steps stay as
// they were. step returns what it did, for the walk's message, or "".
type systemTheme struct {
	rng  *rand.Rand
	dark bool
	pr   Preferences
}

func newSystemTheme(seed uint64) *systemTheme {
	return &systemTheme{rng: rand.New(rand.NewPCG(seed, 31))}
}

func (s *systemTheme) step(both func(func(tt *Tester))) string {
	switch s.rng.IntN(24) {
	case 0:
		s.dark = !s.dark
		both(func(tt *Tester) { tt.SetDark(s.dark) })
		return fmt.Sprintf(" + system dark %v", s.dark)
	case 1, 2, 3:
		switch s.rng.IntN(4) {
		case 0:
			s.pr.Accent = Color{R: uint8(s.rng.IntN(256)), G: uint8(s.rng.IntN(256)), B: uint8(s.rng.IntN(256)), A: 255}
		case 1:
			s.pr.HighContrast = !s.pr.HighContrast
		case 2:
			s.pr.ReduceMotion = !s.pr.ReduceMotion
		default:
			s.pr.TextScale = []float32{0, 1, 1.25, 1.5}[s.rng.IntN(4)]
		}
		pr := s.pr
		both(func(tt *Tester) { tt.SetPreferences(pr) })
		return fmt.Sprintf(" + system preferences %+v", pr)
	}
	return ""
}

// The app moves a Switch's knob with Jump (and versions its rows): the channel
// rests at once, without a frame of motion, so the row is built the frame after
// it, as a built twin's, which then moves the knob back.
func TestRetainRowsSwitchKnobJumped(t *testing.T) {
	const n = 8
	var on [2][]bool
	var jump [2]float32
	var ver [2]uint64
	on[0], on[1] = make([]bool, n), make([]bool, n)
	twin := 0
	p := newPair(t, 300, 300, func() func(c *Context) {
		me := twin
		twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return ver[me] }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Children(func() {
						sw := Switch(c, &on[me][i])
						if i == 3 && jump[me] != 0 {
							sw.Jump("knob", jump[me])
						}
					})
				}).Grow(1)
			})
		}
	})
	p.idle(4)
	jump, ver = [2]float32{0.5, 0.5}, [2]uint64{1, 1}
	p.frame("jump")
	jump = [2]float32{}
	p.frame("after the jump")
	p.both(func(tt *Tester) { tt.Advance(500 * time.Millisecond) })
	p.compare("later")
}
