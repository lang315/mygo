package ui

import "testing"

// MeasureHeight is the height a view takes laid out at a width, as a window
// sized to its content would have it.
func TestContentMeasureHeightIsTheHeightOfTheContent(t *testing.T) {
	v := View(func(c *Context) {
		Column(c).Fill().Padding(10).Children(func() {
			Box(c).Height(100)
			Box(c).Height(50)
		})
	})
	if h := v.MeasureHeight(300); h != 170 {
		t.Errorf("height %v, want 100 + 50 + 2*10", h)
	}
}

func TestContentMeasureHeightFollowsTheWidthOfWrappedText(t *testing.T) {
	long := "A long line of text that wraps in a narrow window and takes more lines there than in a wide one."
	v := View(func(c *Context) { Column(c).Fill().Children(func() { Text(c, long) }) })
	narrow, wide := v.MeasureHeight(120), v.MeasureHeight(1200)
	if narrow <= wide || wide <= 0 {
		t.Errorf("narrow %v, wide %v: the narrow one should be taller", narrow, wide)
	}
}

// A list that scrolls counts a few rows, not all of them: a window sized to
// its content is not as tall as a thousand rows.
func TestContentMeasureHeightCountsAFewRowsOfAList(t *testing.T) {
	var s ListState
	v := View(func(c *Context) {
		Column(c).Fill().Children(func() {
			Text(c, "title")
			List(c, &s, 1000, func(i int) { Box(c).Height(20) }).Grow(1)
		})
	})
	h := v.MeasureHeight(300)
	if h > 400 || h < 100 {
		t.Errorf("height %v for a list of 1000 rows of 20: want about 8 rows and the title", h)
	}
}
