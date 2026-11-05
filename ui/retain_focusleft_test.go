package ui

import "testing"

func focusLeftPair(t *testing.T, auto bool) {
	show := false
	var got [2]int
	p := newPair(t, 400, 400, func() func(c *Context) {
		texts := make([]string, 10)
		var other string
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			n := 0
			if c.rt.noRetain {
				n = 1
			}
			Column(c).Fill().Children(func() {
				if show {
					if auto {
						TextInput(c, &other).AutoFocus()
					} else {
						TextInput(c, &other)
					}
				} else {
					Box(c).Size(10, 30)
				}
				List(c, s, 10, func(i int) {
					Row(c).Height(30).Children(func() {
						in := TextInput(c, &texts[i]).Width(150)
						if in.FocusLeft() {
							got[n]++
						}
					})
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	p.both(func(tt *Tester) { tt.ClickAt(50, 45) })
	p.frame("focused row 0")
	p.frame("focused row 0, 2")
	show = true
	if !auto {
		p.frame("other shown")
		p.both(func(tt *Tester) { tt.ClickAt(50, 15) })
	}
	for range 4 {
		p.frame("moved")
	}
	if got[1] == 0 || got[0] != got[1] {
		t.Errorf("FocusLeft true: kept %d, twin %d", got[0], got[1])
	}
}

func TestRetainRowsFocusLeftWhenTheFocusGoesToANewElement(t *testing.T)  { focusLeftPair(t, true) }
func TestRetainRowsFocusLeftWhenTheFocusGoesToAnotherElement(t *testing.T) { focusLeftPair(t, false) }
