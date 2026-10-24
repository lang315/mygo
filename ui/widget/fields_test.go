package widget

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// bar is a control h DIPs high, red from edge to edge, with text at its top
// left: tt.Find gives where the control starts, and a pixel where it ends.
func bar(text string, h float32) *CustomWidget {
	return Custom(func(c *ui.Context) ui.Element {
		return ui.Column(c).Height(h).Background(ui.RGB(255, 0, 0)).Children(func() { ui.Text(c, text) })
	})
}

// ink returns the first and last row inside r that has a dark pixel: where
// a label's glyphs are, as tt.Find gives the whole box of the label, which
// a cell stretches.
func ink(t *testing.T, tt *ui.Tester, r ui.Rect) (top, bottom float32) {
	t.Helper()
	img, top, bottom := tt.Image(), float32(-1), float32(-1)
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if c := img.RGBAAt(x, y); c.R < 100 && c.G < 100 && c.B < 100 {
				if top < 0 {
					top = float32(y)
				}
				bottom = float32(y + 1)
				break
			}
		}
	}
	if top < 0 {
		t.Fatalf("no text in %v", r)
	}
	return top, bottom
}

func approx(a, b float32) bool { return math.Abs(float64(a-b)) < 1.01 }

func threeRows() *FieldsWidget {
	return Fields().Row("Name", bar("c1", 24)).Row("Email address", bar("c2", 24)).Row("Age", bar("c3", 24))
}

func TestFieldsTwoColumns(t *testing.T) {
	tt := tester(t, threeRows())
	n, e, a := find(t, tt, "Name"), find(t, tt, "Email address"), find(t, tt, "Age")
	if n.X != e.X || e.X != a.X {
		t.Fatalf("labels at x %v %v %v: want one x", n.X, e.X, a.X)
	}
	c1, c2, c3 := find(t, tt, "c1"), find(t, tt, "c2"), find(t, tt, "c3")
	if c1.X != c2.X || c2.X != c3.X {
		t.Fatalf("controls at x %v %v %v: want one x", c1.X, c2.X, c3.X)
	}
	// The label column is as wide as the widest label, and the gap is the
	// theme's 3 units (12 DIPs).
	if want := e.X + e.W + 12; !approx(c1.X, want) {
		t.Fatalf("controls at x %v, want %v: widest label %v wide plus 12", c1.X, want, e.W)
	}
	// Labels sit at the middle of their control's height.
	if top, bottom := ink(t, tt, e); math.Abs(float64((top+bottom)/2-(c2.Y+12))) > 3 {
		t.Fatalf("label text spans y %v to %v, its control's middle is %v", top, bottom, c2.Y+12)
	}
	// The control column takes the rest of the width: red to the right edge.
	if px := tt.Image().RGBAAt(798, int(c1.Y)+20); px.R < 200 || px.G > 50 {
		t.Fatalf("pixel at the right edge %v: the control does not fill the column", px)
	}
}

func TestFieldsStacked(t *testing.T) {
	tt := tester(t, threeRows().Stacked())
	for i, name := range []string{"Name", "Email address", "Age"} {
		l, c := find(t, tt, name), find(t, tt, []string{"c1", "c2", "c3"}[i])
		if l.Y+l.H > c.Y+0.5 || l.X != c.X {
			t.Fatalf("%s at %v, its control at %v: want the label above, at the same x", name, l, c)
		}
	}
	if px := tt.Image().RGBAAt(798, int(find(t, tt, "c1").Y)+20); px.R < 200 || px.G > 50 {
		t.Fatalf("pixel at the right edge %v: the control does not fill the width", px)
	}
	if a, b := find(t, tt, "c1"), find(t, tt, "Email address"); b.Y <= a.Y {
		t.Fatalf("the second row at %v is not below the first control at %v", b, a)
	}
}

// A control much taller than its label: the label sits at its middle, and
// its own VAlign moves it.
func TestFieldsLabelsAreCenteredAgainstTheirControl(t *testing.T) {
	f := Fields().Row("Name", bar("tall", 80)).Row("Top", bar("tall2", 80))
	f.Label(1).VAlign(Start)
	tt := tester(t, f)
	tall := find(t, tt, "tall")
	if top, bottom := ink(t, tt, find(t, tt, "Name")); math.Abs(float64((top+bottom)/2-(tall.Y+40))) > 3 {
		t.Fatalf("label text spans y %v to %v, its control's middle is %v", top, bottom, tall.Y+40)
	}
	if top, _ := ink(t, tt, find(t, tt, "Top")); top > find(t, tt, "tall2").Y+8 {
		t.Fatalf("VAlign(Start) label's text starts at y %v, its control at %v", top, find(t, tt, "tall2").Y)
	}
}

func TestFieldsGap(t *testing.T) {
	tt := tester(t, threeRows().Gap(10, 20))
	e := find(t, tt, "Email address")
	if want := e.X + e.W + 20; !approx(find(t, tt, "c1").X, want) {
		t.Fatalf("controls at x %v, want %v: a column gap of 20", find(t, tt, "c1").X, want)
	}
	if step := find(t, tt, "c2").Y - find(t, tt, "c1").Y; !approx(step, 24+10) {
		t.Fatalf("rows %v apart, want the control's 24 plus the row gap 10", step)
	}
}

func TestFieldsHiddenRowTakesNoSpace(t *testing.T) {
	c2 := bar("c2", 24)
	f := Fields().Row("Name", bar("c1", 24)).Row("Email address", c2).Row("Age", bar("c3", 24)).Row("Nothing", nil)
	tt := tester(t, f)
	before := find(t, tt, "c3").Y
	if tt.HasText("Nothing") {
		t.Fatal("a row without a control is shown")
	}
	c2.SetVisible(false)
	tt.Frame()
	if tt.HasText("Email address") || tt.HasText("c2") {
		t.Fatalf("a hidden row shows: %q", tt.Texts())
	}
	// The row and its gap are gone: the third row moved up by both.
	if got := find(t, tt, "c3").Y; !approx(before-got, 24+8) {
		t.Fatalf("third row moved up by %v, want a row and its gap, 32", before-got)
	}
	c2.SetVisible(true)
	tt.Frame()
	if got := find(t, tt, "c3").Y; !approx(got, before) || !tt.HasText("Email address") {
		t.Fatalf("shown again: third row at %v, was %v; texts %q", got, before, tt.Texts())
	}
}

func TestFieldsLabelsStyleAsLabels(t *testing.T) {
	plain := find(t, tester(t, threeRows()), "Email address").W
	f := threeRows()
	f.Label(1).Bold()
	if got := find(t, tester(t, f), "Email address").W; got <= plain {
		t.Fatalf("a bold label is %v wide, plain %v", got, plain)
	}
	if f.Label(-1) != nil || f.Label(3) != nil || f.Label(0) == nil {
		t.Fatal("Label(i) must be nil out of range and a label inside")
	}
}

// A styled label is as wide as its text, not as its column.
func TestFieldsLabelsHugTheirText(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	for _, stacked := range []bool{false, true} {
		f := threeRows()
		if stacked {
			f.Stacked()
		}
		f.Label(2).Background(red) // "Age", narrower than "Email address"
		tt := tester(t, f)
		age, wide := find(t, tt, "Age"), find(t, tt, "Email address")
		y := int(age.Y) + 1
		if !near(tt.Image().RGBAAt(int(age.X)+1, y), red) {
			t.Fatalf("stacked %v: the label's own background is not drawn at its text", stacked)
		}
		x := int(wide.X + wide.W - 2) // the end of the column, or well past the text
		if stacked {
			x = 300
		}
		if near(tt.Image().RGBAAt(x, y), red) {
			t.Fatalf("stacked %v: the label's background reaches x %d, past its text", stacked, x)
		}
	}
}

func TestFieldsLabelAlignmentIsTheLabels(t *testing.T) {
	f := threeRows()
	f.Label(2).Align(End) // "Age", the shortest: right-aligned in the column
	tt := tester(t, f)
	if age, name := find(t, tt, "Age"), find(t, tt, "Name"); age.X <= name.X {
		t.Fatalf("Age at x %v, Name at %v: Align(End) must move it right", age.X, name.X)
	}
}

func TestFieldsHiddenLabelKeepsTheControlInItsColumn(t *testing.T) {
	f := threeRows()
	tt := tester(t, f)
	want, y := find(t, tt, "c1").X, find(t, tt, "c1").Y
	f.Label(0).SetVisible(false) // not the widest: the column keeps its width
	tt.Frame()
	if tt.HasText("Name") {
		t.Fatal("the label is still shown")
	}
	if c1, c2 := find(t, tt, "c1"), find(t, tt, "c2"); c1.X != want || c2.X != want || c1.Y != y {
		t.Fatalf("controls moved: c1 at %v, c2 at %v; want x %v and c1 still at y %v", c1, c2, want, y)
	}
}

func TestFieldsIsAControl(t *testing.T) {
	plain := find(t, tester(t, threeRows()), "Name").X
	styled := find(t, tester(t, threeRows().Apply(NewStyle().Padding(3))), "Name").X
	if styled < plain+11 {
		t.Fatalf("Apply(Padding(3)) moved the first label from %v to %v, want 12 right", plain, styled)
	}
	var _ Control = Fields()
	tt := tester(t, VBox(0).Add(threeRows().Align(Start), Label("after")))
	if !tt.HasText("after") || !tt.HasText("Name") {
		t.Fatalf("not shown in a box: %q", tt.Texts())
	}
}

func TestFieldsRowsAddedWhileFramesBuild(t *testing.T) {
	f := Fields()
	tt := tester(t, f)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			f.Row("L", Label("x"))
		}
	}()
	for range 20 {
		tt.Frame()
	}
	<-done
	tt.Frame()
	if !tt.HasText("L") {
		t.Fatalf("not shown: %q", tt.Texts())
	}
}

// A row hidden when its window first draws, as an error row is until there
// is an error, still tells the window when it shows: Fields holds what it
// does not build, as a box does.
func TestShowingAFieldsRowHiddenFromTheStartRedrawsItsWindow(t *testing.T) {
	for _, stacked := range []bool{false, true} {
		for _, what := range []string{"control", "label"} {
			ctl := Label("error").SetVisible(what != "control")
			f := Fields().Row("Name", Label("shown")).Row("Err", ctl)
			if stacked {
				f.Stacked()
			}
			if what == "label" {
				f.Label(1).SetVisible(false)
			}
			_, w := showFrame(t, VBox(0).Add(f))
			redrawn := recordInvalidations(t)
			if what == "control" {
				ctl.SetVisible(true)
			} else {
				f.Label(1).SetVisible(true)
			}
			t.Logf("stacked %v, hidden %s", stacked, what)
			wantRedrawn(t, redrawn(), w)
		}
	}
}

// A stacked form lays its children out down a column, whatever the box
// around it: a control's Align(End) is at the end of the form's width.
func TestFieldsStackedInAnHBoxAlignsAcrossItsColumn(t *testing.T) {
	const long = "A much longer label here"
	tt := tester(t, HBox(0).Add(Fields().Stacked().Row(long, Label("x").Align(End))))
	l, x := find(t, tt, long), find(t, tt, "x")
	if x.X < l.X+l.W/2 {
		t.Fatalf("the control is at x %v, the label spans %v to %v: Align(End) must put it at the end", x.X, l.X, l.X+l.W)
	}
}

// In two columns a control's VAlign and expansion are along its row.
func TestFieldsGridControlsAlignDownTheirRow(t *testing.T) {
	f := Fields().Row("Tall", Label("v").VAlign(End))
	f.Label(0).TextSize(XXXL)
	tt := tester(t, f)
	l, v := find(t, tt, "Tall"), find(t, tt, "v")
	// The ink, not v's box: a control that is not placed is stretched over
	// the row, and its box reaches the bottom too.
	if top, _ := ink(t, tt, v); top < l.Y+l.H/2 {
		t.Fatalf("the control's text starts at y %v in a row from %v to %v: VAlign(End) must put it at the bottom", top, l.Y, l.Y+l.H)
	}
}

func TestFieldsStackedLabelIsOneUnitAboveItsControl(t *testing.T) {
	tt := tester(t, Fields().Stacked().Row("Name", bar("c1", 24)))
	if gap := find(t, tt, "c1").Y - (find(t, tt, "Name").Y + find(t, tt, "Name").H); !approx(gap, 4) {
		t.Fatalf("a stacked label is %v above its control, want the theme's unit, 4", gap)
	}
}

func TestFieldsRowWithAControlTheFormHasIsNotAdded(t *testing.T) {
	c := bar("c1", 24)
	f := Fields().Row("First", c).Row("Second", c)
	if f.Label(1) != nil {
		t.Fatal("a control in two rows")
	}
	if tt := tester(t, f); tt.HasText("Second") || !tt.HasText("First") {
		t.Fatalf("texts %q", tt.Texts())
	}
}
