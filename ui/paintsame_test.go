package ui

import "testing"

// In an idle frame of a table, every row in view paints as before; a row
// whose look changed does not, and the others still do.
func TestRowsThatPaintAsBefore(t *testing.T) {
	var s ListState
	color := RGB(10, 10, 10)
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, []TableColumn{{Title: "A"}}, 20, func(row, col int) {
			e := coreText(c, "row")
			if row == 3 {
				e.TextColor(color)
			}
		}).Grow(1)
	}, 300, 400)
	tt.Frame()
	tt.Frame()
	inView := tt.rt.paintSameRows
	if inView == 0 {
		t.Fatal("no row paints as before in an idle frame")
	}
	color = RGB(200, 0, 0)
	tt.Frame()
	if got := tt.rt.paintSameRows; got != inView-1 {
		t.Fatalf("after one row's text color changed, %d rows paint as before, want %d", got, inView-1)
	}
	tt.Frame()
	if got := tt.rt.paintSameRows; got != inView {
		t.Fatalf("a frame later, %d rows paint as before, want %d", got, inView)
	}
}

// Two elements with one key keep nothing (sameLayout): neither paints as
// before.
func TestTwoElementsOfOneKeyDoNotPaintAsBefore(t *testing.T) {
	var a, b *node
	tt := coreNewTester(func(c *context) {
		c.rt.strict = false // duplicate keys panic otherwise
		coreBox(c).Children(func() {
			a = coreBox(c).Key("same")
			b = coreBox(c).Key("same")
		})
	}, 100, 100)
	tt.rt.paintAll = true
	tt.Frame()
	tt.Frame()
	if a.lay&layPaint != 0 || b.lay&layPaint != 0 {
		t.Fatal("an element sharing its key with another paints as before")
	}
}

// The theme an element was built with is compared, not its index, where its
// painting reads the theme: after the themes start over, an index may name
// another theme.
func TestAnotherThemeAtTheSameIndexIsAChange(t *testing.T) {
	t1, t2 := LightTheme(), LightTheme()
	use := t1
	var e *node
	tt := coreNewTester(func(c *context) {
		e = coreBox(c)
		e.setStyle(staticHoverSurface)
		c.rt.themes = append(c.rt.themes[:0], use)
		e.themeAt = 1
	}, 100, 100)
	tt.rt.paintAll = true
	tt.Frame()
	tt.Frame()
	if e.lay&layPaint == 0 {
		t.Fatal("an unchanged element does not paint as before")
	}
	use = t2
	tt.Frame()
	if e.lay&layPaint != 0 {
		t.Fatal("an element built with another theme paints as before")
	}
}

// A text color is inherited: the child of an element whose color changed
// paints anew although its own styles are the same.
func TestInheritedTextColorIsAChange(t *testing.T) {
	color := RGB(10, 10, 10)
	var child *node
	tt := coreNewTester(func(c *context) {
		coreBox(c).TextColor(color).Children(func() { child = coreText(c, "x") })
	}, 100, 100)
	tt.rt.paintAll = true
	tt.Frame()
	tt.Frame()
	if child.lay&layPaint == 0 {
		t.Fatal("an unchanged text does not paint as before")
	}
	color = RGB(200, 0, 0)
	tt.Frame()
	if child.lay&layPaint != 0 {
		t.Fatal("a text whose inherited color changed paints as before")
	}
}

// A text color, underline and background each make a row paint anew.
func TestTextPaintStylesAreChanges(t *testing.T) {
	for name, style := range map[string]func(e *node){
		"color":      func(e *node) { e.TextColor(RGB(1, 2, 3)) },
		"underline":  func(e *node) { e.Underline() },
		"strike":     func(e *node) { e.Strikethrough() },
		"background": func(e *node) { e.TextBackground(RGB(1, 2, 3)) },
	} {
		on := false
		var e *node
		tt := coreNewTester(func(c *context) {
			e = coreText(c, "x")
			if on {
				style(e)
			}
		}, 100, 100)
		tt.rt.paintAll = true
		tt.Frame()
		tt.Frame()
		if e.lay&layPaint == 0 {
			t.Fatalf("%s: an unchanged text does not paint as before", name)
		}
		on = true
		tt.Frame()
		if e.lay&layPaint != 0 {
			t.Errorf("%s: a text whose style changed paints as before", name)
		}
	}
}

// A child replaced by another kind with equal fields is not the same.
func TestAnotherKindIsNotPaintSame(t *testing.T) {
	icon := false
	var parent *node
	tt := coreNewTester(func(c *context) {
		parent = coreBox(c).Children(func() {
			if icon {
				coreBox(c)
			} else {
				coreText(c, "")
			}
		})
	}, 100, 100)
	tt.rt.paintAll = true
	tt.Frame()
	tt.Frame()
	if parent.lay&layPaint == 0 {
		t.Fatal("an unchanged parent does not paint as before")
	}
	icon = true
	tt.Frame()
	if parent.lay&layPaint != 0 {
		t.Fatal("a parent whose child is of another kind paints as before")
	}
}

// A change of what paints an element and not of its layout makes it paint
// anew: its background, border, radius and opacity.
func TestPaintStylesAreChanges(t *testing.T) {
	for name, style := range map[string]func(e *node){
		"background": func(e *node) { e.Background(RGB(1, 2, 3)) },
		"border":     func(e *node) { e.BorderColor(RGB(1, 2, 3)) },
		"radius":     func(e *node) { e.Radius(4) },
		"opacity":    func(e *node) { e.Opacity(0.5) },
		"shadow":     func(e *node) { e.Shadow(0, 1, 2, 0, RGB(1, 2, 3)) },
	} {
		on := false
		var e *node
		tt := coreNewTester(func(c *context) {
			e = coreBox(c).Size(20, 20)
			if on {
				style(e)
			}
		}, 100, 100)
		tt.rt.paintAll = true
		tt.Frame()
		tt.Frame()
		if e.lay&layPaint == 0 {
			t.Fatalf("%s: an unchanged element does not paint as before", name)
		}
		on = true
		tt.Frame()
		if e.lay&layPaint != 0 {
			t.Errorf("%s: an element whose paint style changed paints as before", name)
		}
	}
}

// A text style that paints, set on the List or an element around it, is
// inherited by the rows: they paint anew when it changes.
func TestTextPaintAboveTheListIsAChange(t *testing.T) {
	for name, wrap := range map[string]func(c *context, color Color, body func()){
		"around":      func(c *context, color Color, body func()) { coreBox(c).TextColor(color).Children(body) },
		"on the list": func(c *context, color Color, body func()) { body() },
	} {
		color := RGB(10, 10, 10)
		var s ListState
		tt := coreNewTester(func(c *context) {
			wrap(c, color, func() {
				l := coreTable(c, &s, []TableColumn{{Title: "A"}}, 20, func(row, col int) { coreText(c, "row") }).Grow(1)
				if name == "on the list" {
					l.TextColor(color)
				}
			})
		}, 300, 400)
		tt.Frame()
		tt.Frame()
		inView := tt.rt.paintSameRows
		if inView == 0 {
			t.Fatalf("%s: no row paints as before", name)
		}
		color = RGB(200, 0, 0)
		tt.Frame()
		if got := tt.rt.paintSameRows; got != 0 {
			t.Errorf("%s: %d rows paint as before after the inherited color changed, want 0", name, got)
		}
		tt.Frame()
		if got := tt.rt.paintSameRows; got != inView {
			t.Errorf("%s: a frame later, %d rows paint as before, want %d", name, got, inView)
		}
	}
}

// Selectable text paints its selection, which the node does not hold.
func TestSelectableRowsDoNotPaintAsBefore(t *testing.T) {
	var s ListState
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, []TableColumn{{Title: "A"}}, 20, func(row, col int) { coreText(c, "row").Selectable() }).Grow(1)
	}, 300, 400)
	tt.Frame()
	tt.Frame()
	if got := tt.rt.paintSameRows; got != 0 {
		t.Fatalf("%d rows of selectable text paint as before", got)
	}
}
