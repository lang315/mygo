package ui

import (
	"reflect"
	"strings"
	"testing"
)

// Every field of node is placed: layout reads it (the layout cache compares
// it, or it is what layout makes), paintStyle or textPaint compares it, it
// makes an element never paint-same (volatile), it is in the key of a replay
// (geometry, and what the frame's state decides), or paint does not read it.
// A field added to node must be placed here: a field that paints and sits in
// none of these would let a later frame replay a row that looks different
// (retained rows).
func TestPaintInputs(t *testing.T) {
	fields := map[string]string{
		// layout: compared by sameLayout, or made by the layout
		"layout": "layoutStyle cols rows hidden measured tl lay laid",
		// compared by paintStyle, or by textPaint (ts)
		"paint": "flags themeAt paintLook ts shadows styleFn spans",
		// an element with one set is never paint-same
		"volatile": "paintRare ownText paragraph merged frags spansKey grad stripes originX originY originSet content pose poseAnim themed hideScrollbars keepScrollbars barsFrom", // only scroll containers read the scroll bars' fields, and they are volatile; a Themed is volatile (paintVolatile); the origin counts with a transform, in paintRare
		// geometry and what the frame decides: the key of a replay
		"key": "x y w h contentW contentH world poseM ink scrollBase track",
		// not read by paint
		"neither": "epoch key valueInput ignoreInput c id parent first last next nchild depth st cursor inputFn textClient caret " +
			"takesText noIME copyData runes maximizeButton dismissed label widget role expanded vertical tip accValue " +
			"accRange hasRange accFill accStep popover rowsOf focusGroup groupSelects placeOn overflow segment " +
			"activeDescendant nameFrom nameJoin outOfName keepLinks focusTo setPos setSize choosesItems search description " +
			"field form rowMinW followX sort colFit level expandable chooseMany gridFit attach serial fillViewport",
	}
	where := map[string]string{}
	for place, list := range fields {
		for _, f := range strings.Fields(list) {
			if w, dup := where[f]; dup {
				t.Errorf("node.%s is in both %s and %s", f, w, place)
			}
			where[f] = place
		}
	}
	ty := reflect.TypeFor[node]()
	seen := map[string]bool{}
	for i := range ty.NumField() {
		f := ty.Field(i).Name
		seen[f] = true
		if where[f] == "" {
			t.Errorf("node.%s: say whether paint reads it (TestPaintInputs, paintstyle.go)", f)
		}
	}
	// paintLook's fields are all compared, paintRare's make an element volatile:
	// node holds them for that.
	if where["paintLook"] != "paint" || where["paintRare"] != "volatile" {
		t.Error("paintLook is not compared, or paintRare is not volatile")
	}
	for f := range where {
		if !seen[f] {
			t.Errorf("node.%s is placed but is not a field", f)
		}
	}
}

// paintStyle tells apart every change of a paint input, and an element with
// a volatile field set is never paint-same.
func TestPaintStyleSeesEachInput(t *testing.T) {
	tt := coreNewTester(func(c *context) { coreBox(c) }, 100, 100)
	base := func() *node {
		e := &node{c: &tt.rt.c}
		e.bg, e.borderC = RGB(1, 2, 3), RGB(4, 5, 6)
		return e
	}
	refE := base()
	ref, ok := paintStyleOf(refE)
	if !ok || !paintStyleSame(&ref, refE) {
		t.Fatal("a plain element is volatile")
	}
	for name, change := range map[string]func(e *node){
		"bg":          func(e *node) { e.bg = RGB(9, 9, 9) },
		"borderC":     func(e *node) { e.borderC = RGB(9, 9, 9) },
		"borderStyle": func(e *node) { e.borderStyle = BorderDashed },
		"radius":      func(e *node) { e.radius[2] = 3 },
		"opacity":     func(e *node) { e.opacity, e.opacitySet = 0.5, true },
		"checked":     func(e *node) { e.checked = 2 },
		"checkmark":   func(e *node) { e.checkmark = 1 },
		"backface":    func(e *node) { e.backface = true },
		"noCull":      func(e *node) { e.noCull = true },
		"highlighted": func(e *node) { e.highlighted = true },
		"invalid":     func(e *node) { e.invalid = true },
		"listRow":     func(e *node) { e.listRow = true },
		"rowIndex":    func(e *node) { e.rowIndex = 7 },
		"collapsed":   func(e *node) { e.collapsed = true },
		"styleID":     func(e *node) { setStyle(e, hoverRow, staticHoverRow) },
		"buttonLook":  func(e *node) { e.buttonLook = buttonPrimary },
		"flags":       func(e *node) { e.flags |= flagDisabled },
		"shadow":      func(e *node) { e.shadows = []shadow{{y: 1, blur: 2}} },
		"theme": func(e *node) {
			setStyle(e, hoverRow, staticHoverRow)
			e.themeAt = 0
			e.c = &context{theme: &Theme{}, rt: e.c.rt}
		},
	} {
		e := base()
		change(e)
		if _, ok := paintStyleOf(e); !ok || paintStyleSame(&ref, e) {
			t.Errorf("%s: paintStyle did not change", name)
		}
	}
	// The layout reads these flags: layout compares them, not paintStyle.
	e := base()
	e.flags |= flagAbsolute
	if _, ok := paintStyleOf(e); !ok || !paintStyleSame(&ref, e) {
		t.Error("a flag layout reads changed paintStyle")
	}
	for name, set := range map[string]func(e *node){
		"paintFn":   func(e *node) { e.paintFn = func(*Painter, Rect) {} },
		"closure":   func(e *node) { e.styleFn = func(*node) {} },
		"gradient":  func(e *node) { e.fill = fillGradient },
		"transform": func(e *node) { e.Translate(1, 2) },
		"image":     func(e *node) { e.image = &Bitmap{} },
		"input":     func(e *node) { e.kind = kindInput },
		"scroll":    func(e *node) { e.flags |= flagScrollY },
		"leaving":   func(e *node) { e.leaving = 1 },
	} {
		e := base()
		set(e)
		if _, ok := paintStyleOf(e); ok {
			t.Errorf("%s: an element with it is not volatile", name)
		}
	}
}

// textPaintOf tells apart every text style that paints and is not laid out.
func TestTextPaintSeesEachInput(t *testing.T) {
	ref := textPaintOf(&textStyle{})
	for name, change := range map[string]func(s *textStyle){
		"color":      func(s *textStyle) { s.color, s.set = RGB(9, 9, 9), s.set|setColor },
		"underline":  func(s *textStyle) { s.underline, s.set = true, s.set|setUnderline },
		"wavy":       func(s *textStyle) { s.wavy, s.set = true, s.set|setUnderline },
		"strike":     func(s *textStyle) { s.strike, s.set = true, s.set|setStrike },
		"decoColor":  func(s *textStyle) { s.decoColor, s.set = RGB(9, 9, 9), s.set|setDecoColor },
		"decoThick":  func(s *textStyle) { s.decoThick, s.set = 3, s.set|setDecoThick },
		"background": func(s *textStyle) { s.background, s.set = RGB(9, 9, 9), s.set|setBackground },
		"selection":  func(s *textStyle) { s.selection, s.set = RGB(9, 9, 9), s.set|setSelection },
	} {
		var s textStyle
		change(&s)
		if textPaintOf(&s) == ref || textPaintSame(&ref, &s) {
			t.Errorf("%s: textPaintOf did not change", name)
		}
		if p := textPaintOf(&s); !textPaintSame(&p, &s) {
			t.Errorf("%s: a text paint is not the same as its style", name)
		}
	}
	// A text style that lays out does not paint.
	if size := (textStyle{size: 20, set: setSize}); textPaintOf(&size) != ref || !textPaintSame(&ref, &size) {
		t.Error("a size changed textPaintOf")
	}
}

// paintLook has no padding, whose bytes would differ with nothing in them
// differing: equal compares bytes.
func TestPaintLookHasNoPadding(t *testing.T) {
	ty := reflect.TypeFor[paintLook]()
	var sum uintptr
	for i := range ty.NumField() {
		sum += ty.Field(i).Type.Size()
	}
	if sum != ty.Size() {
		t.Errorf("paintLook has %d bytes of padding: reorder its fields or write the padding out", ty.Size()-sum)
	}
}
