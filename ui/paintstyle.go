package ui

import (
	"slices"
	"unsafe"
)

// textPaint is what of a text style paints text besides what lays it out
// (textLayoutSet). Elements inherit it, so the layout cache compares it down
// the tree (sameLayout's inh).
type textPaint struct {
	set                     uint32
	color                   Color
	underline, wavy, strike bool
	decoColor               Color
	decoThick               float32
	background, selection   Color
}

// textPaintSet are the text styles that paint text and are not laid out.
const textPaintSet = setColor | setUnderline | setStrike | setDecoColor | setDecoThick | setBackground | setSelection

// textPaintOf returns what of t paints.
func textPaintOf(t *textStyle) textPaint {
	return textPaint{
		set: t.set & textPaintSet, color: t.color, underline: t.underline, wavy: t.wavy, strike: t.strike,
		decoColor: t.decoColor, decoThick: t.decoThick, background: t.background, selection: t.selection,
	}
}

// same reports whether t paints text as p says: what is not set is not read.
func textPaintSame(p *textPaint, t *textStyle) bool {
	set := t.set & textPaintSet
	if set != p.set {
		return false
	}
	return set == 0 ||
		(set&setColor == 0 || p.color == t.color) &&
			(set&setUnderline == 0 || p.underline == t.underline && p.wavy == t.wavy) &&
			(set&setStrike == 0 || p.strike == t.strike) &&
			(set&setDecoColor == 0 || p.decoColor == t.decoColor) &&
			(set&setDecoThick == 0 || p.decoThick == t.decoThick) &&
			(set&setBackground == 0 || p.background == t.background) &&
			(set&setSelection == 0 || p.selection == t.selection)
}

// paintLook is the scalar fields that paint an element, besides what paintStyle
// compares from elsewhere: kept together in node, so that comparing an element
// with its last layout's reads few cache lines.
//
// It has no padding (TestPaintLookHasNoPadding): equal compares its bytes,
// which is faster than comparing its fields, and stricter only for -0 and 0.
type paintLook struct {
	// rowIndex is the row of an element holding the row of a list (listRow).
	rowIndex int
	radius   [4]float32
	opacity  float32
	bg       Color
	borderC  Color
	// checked is whether a check box, radio or switch is off (1), on (2) or
	// mixed (3).
	checked int8
	// checkmark is the state of a check box (1 off, 2 on, 3 mixed; a radio button's dot: 4 off, 5 on) when its
	// box was built, which the box paints (paintCheckmark), and 0 for any
	// other element.
	checkmark int8
	// buttonLook is how a button paints (buttonPlain, buttonPrimary or
	// buttonFlat), for buttonPaint.
	buttonLook  uint8
	borderStyle BorderStyle
	// styleID is the static function styleFn is, staticNone for a closure
	// (setStyle).
	styleID    styleID
	opacitySet bool
	// noCull: what the element paints may be beyond its ink bounds.
	noCull bool
	// backface hides it while its 3D transform turns its back to the viewer
	// (BackfaceHidden).
	backface bool
	// highlighted is set on the option of a select the pointer or the arrows
	// are on.
	highlighted bool
	listRow     bool
	// collapsed: a toolbar's control with no room, out of its flow.
	collapsed bool
	// invalid marks a value that is not valid (Description, Error).
	invalid bool
	_       [4]byte // the padding, written out
}

// equal reports whether a and b are the same, byte by byte.
func paintLookEqual(a, b *paintLook) bool {
	const n = unsafe.Sizeof(paintLook{})
	return *(*[n]byte)(unsafe.Pointer(a)) == *(*[n]byte)(unsafe.Pointer(b))
}

// paintRare holds the fields of node that few elements set, together: an
// element with any of them set is never paint-same (paintVolatile), which
// sameLayout finds by comparing them all with zero at once.
type paintRare struct {
	// barInset moves a scroll container's scroll bars in from its edges
	// (ScrollbarInsets): top, right, bottom, left. blur is the standard
	// deviation of the element's blur, in DIPs. rotate is that of an Icon, in
	// degrees; fit and gray are those of an image.
	barInset     [4]float32
	blur, rotate float32
	fit          Fit
	gray         bool
	transform    Transform
	material     Material
	effect       Effect
	// cache keeps the element's layer while what it paints is the same
	// (Cache).
	cache layerKeeper
	// grads holds a conic fill and a border's gradient, which few elements
	// have.
	grads *gradients
	// paintFn draws the element (Draw), objectFn the inline objects of a text
	// (DrawObjects) and paintAfterFn goes over its children.
	paintFn      func(p *Painter, r Rect)
	objectFn     func(p *Painter, at int, r Rect)
	paintAfterFn func(p *Painter, r Rect)
	webView      NativeWebView
	image        *Bitmap
	svg          *SVG // of an Icon, or an Image in its own colors
	// list is the List the element is, while it builds and lays out.
	list *listFrame
	// poseFade is how far a pose fades the element (pose, AnimatePose).
	poseFade float32
	fill     fillKind
	// contentSet: its content changes (content, ContentTransition). poseSet:
	// AnimatePose moves it, and reposed tells, in a frame that only paints,
	// that its pose moved. effecting: the effect is running.
	contentSet, poseSet, reposed, effecting bool
	// leaving marks a copy of an element gone, which its exit transition
	// animates (1 if it was in its parent's flow, 2 if absolute), and ghosts an
	// element holding such copies.
	leaving uint8
	ghosts  bool
}

// paintStyle is what paints an element besides its layout (layoutStyle) and
// its text styles (textPaint): sameLayout keeps it with the layout's record,
// so that a row whose elements all paint as before may replay what it painted
// (retained rows). TestPaintInputs places every field of node.
type paintStyle struct {
	// flags are those layout does not read (layoutFlags).
	flags uint32
	// theme is the theme painting read (readsTheme), or nil.
	theme    *Theme
	look     paintLook
	nshadows uint8
	shadows  [paintShadows]shadow
	// spans are those of a RichText, whose colors, lines and backgrounds
	// paint (the layout compares their styles).
	spans []Span
}

// paintShadows is how many shadows paintStyle holds, in the record itself so
// that keeping them allocates nothing: an element with more is never
// paint-same.
const paintShadows = 2

// paintVolatile reports whether e paints with something a later frame
// cannot compare: a closure, an effect, a layer cache, a gradient, a
// transform, a picture, an input, a scroll container, text made of inline
// elements, a color transition, or an element leaving.
func paintVolatile(e *node) bool {
	const n = unsafe.Sizeof(paintRare{})
	return *(*[n]byte)(unsafe.Pointer(&e.paintRare)) != zeroRare ||
		e.styleFn != nil && e.styleID == staticNone || len(e.shadows) > paintShadows ||
		e.kind == kindInput || e.kind == kindImage || e.kind == kindIcon || e.kind == kindText && e.first != nil ||
		e.flags&(flagScrollX|flagScrollY|flagDividers|flagDebug|flagSelectable) != 0 || e.isInline() ||
		e.st != nil && e.st.trec != nil
}

// zeroRare is a paintRare with nothing set, byte by byte.
var zeroRare [unsafe.Sizeof(paintRare{})]byte

// paintStyle returns what paints e besides its layout and its text styles,
// and false when e is volatile (paintVolatile).
func paintStyleOf(e *node) (ps paintStyle, ok bool) {
	if paintVolatile(e) {
		return paintStyle{}, false
	}
	setPaintStyle(&ps, e)
	return ps, true
}

// set makes ps what paints e, which is not volatile.
func setPaintStyle(ps *paintStyle, e *node) {
	ps.flags, ps.look = e.flags&^layoutFlags, e.paintLook
	ps.theme = nil
	if readsTheme(e) {
		ps.theme = e.builtTheme()
	}
	ps.nshadows = uint8(copy(ps.shadows[:], e.shadows))
	ps.spans = append(ps.spans[:0], e.spans...)
}

// same reports whether e, not volatile, paints as ps says. It compares the
// node itself, which costs less than making a paintStyle of it to compare.
func paintStyleSame(ps *paintStyle, e *node) bool {
	return paintLookEqual(&ps.look, &e.paintLook) && ps.flags == e.flags&^layoutFlags && slices.Equal(ps.shadows[:ps.nshadows], e.shadows) && (len(e.spans) == 0 && len(ps.spans) == 0 || slices.Equal(ps.spans, e.spans)) &&
		(!readsTheme(e) || ps.theme == e.builtTheme())
}

// readsTheme reports whether painting e reads the theme it was built with
// (builtWith): the static styles and the check mark do.
func readsTheme(e *node) bool {
	return e.c != nil && (e.styleID != staticNone || e.checkmark != 0)
}
