package ui

import "testing"

// Text styles that paint but do not lay out are inherited by the rows of a
// list and compared down the tree (textPaintSet): a kept row whose
// ancestor changes only the stroke or the offset of the underlines paints
// anew, as the twin that builds every row does.
func TestRetainRowsFollowInheritedStrokeAndUnderlineOffset(t *testing.T) {
	for _, tc := range []struct {
		name  string
		style func(e Element, v float32) Element
	}{
		{"stroke color", func(e Element, v float32) Element { return e.TextStroke(2, RGB(uint8(v), 0, 0)) }},
		{"underline offset", func(e Element, v float32) Element { return e.Underline().UnderlineOffset(v) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := float32(10)
			p := newPair(t, 300, 200, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					tc.style(Column(c).Fill(), v).Children(func() {
						List(c, s, 10, func(i int) { Textf(c, "row %d", i).FontSize(20) }).Grow(1)
					})
				}
			})
			p.idle(3)
			v = 200
			if tc.name == "underline offset" {
				v = 6
			}
			p.both(func(tt *Tester) { tt.Frame() })
			p.frame("the style changed")
			p.frame("settled")
		})
	}
}
