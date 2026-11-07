package tw

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

const longText = "A very long line of text that cannot fit in a narrow window, and goes on and on and on and on and on"

// boundsOf shows c in a window w wide and returns the bounds of the text
// named name: no font metrics, only how it compares with other layouts.
func boundsOf(t *testing.T, c widget.Control, w int, name string) ui.Rect {
	t.Helper()
	return widget.NewTester(t, c, w, 400).Find(widget.ByLabel(name)).Bounds()
}

// oneLine is the height of a label of one short line.
func oneLine(t *testing.T) float32 {
	return boundsOf(t, widget.VBox(0).Add(widget.Label("Hi")), 120, "Hi").H
}

// truncate keeps a label to one line ending in an ellipsis, in any width.
func TestTruncateOneLine(t *testing.T) {
	single := oneLine(t)
	box := func(l *widget.LabelWidget) widget.Control { return widget.VBox(0).Add(l) }

	wrapped := boundsOf(t, box(widget.Label(longText)), 120, longText)
	if wrapped.H <= single {
		t.Fatalf("setup: the label wraps to %v high, one line is %v", wrapped.H, single)
	}
	if got := boundsOf(t, box(widget.Label(longText).Apply(S("truncate"))), 120, longText); got.H != single {
		t.Errorf("truncate: %v high at 120 wide, want one line, %v", got.H, single)
	}
	// Without the class, or with others, the label is as it was.
	if got := boundsOf(t, box(widget.Label(longText).Apply(S("italic"))), 120, longText); got.H != wrapped.H {
		t.Errorf("without truncate: %v high, want %v", got.H, wrapped.H)
	}
}

// The text is still all there for assistive technology, and a wider window
// shows more of it.
func TestTruncateKeepsNameAndShowsMoreWhenWider(t *testing.T) {
	box := func() widget.Control { return widget.VBox(0).Add(widget.Label(longText).Apply(S("truncate"))) }
	narrow := boundsOf(t, box(), 120, longText)
	wide := boundsOf(t, box(), 600, longText)
	if narrow.W > 120 {
		t.Errorf("truncated label is %v wide in a window of 120", narrow.W)
	}
	if wide.W <= narrow.W {
		t.Errorf("wider window shows no more: %v wide, narrow %v", wide.W, narrow.W)
	}
	tt := widget.NewTester(t, box(), 120, 400)
	if n := len(tt.FindAll(widget.ByRole(ui.RoleText).Label(longText))); n != 1 {
		t.Errorf("%d texts are named by the whole text, want 1", n)
	}
}

// A truncated label shrinks below its text beside another control, in a
// row and in a form, so the control stays in view.
func TestTruncateShrinksBesideAControl(t *testing.T) {
	single := oneLine(t)
	truncated := func() *widget.LabelWidget { return widget.Label(longText).Apply(S("truncate")) }

	row := widget.NewTester(t, widget.HBox(4).Add(truncated(), widget.Button("OK")), 240, 400)
	lbl, ok := row.Find(widget.ByLabel(longText)), row.Find(widget.ByLabel("OK"))
	if lbl.Bounds().H != single {
		t.Errorf("in a row: %v high, want one line, %v", lbl.Bounds().H, single)
	}
	if r := ok.Bounds().X + ok.Bounds().W; r > 240 {
		t.Errorf("in a row the button ends at %v, past the window of 240", r)
	}

	form := widget.NewTester(t, widget.Fields().Row("Note", truncated()), 240, 400)
	if b := form.Find(widget.ByLabel(longText)).Bounds(); b.H != single || b.X+b.W > 240 {
		t.Errorf("in a form: %+v, want one line of %v inside the window of 240", b, single)
	}
}

// truncate parses, with no error, to the same Style as the others make.
func TestTruncateParses(t *testing.T) {
	if _, err := Parse("truncate"); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("truncate italic"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(longText, "truncate") {
		t.Fatal("setup")
	}
}

// A label of styled spans is cut the same way.
func TestTruncateRichLabel(t *testing.T) {
	single := oneLine(t)
	l := widget.Label("").Rich(ui.Span{Text: longText}).Apply(S("truncate"))
	if got := boundsOf(t, widget.VBox(0).Add(l), 120, longText); got.H != single {
		t.Errorf("rich label: %v high at 120 wide, want one line, %v", got.H, single)
	}
}
