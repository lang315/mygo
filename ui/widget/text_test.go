package widget

import (
	"testing"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// received returns what the handler sent.
func received[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	return wait.For(t, "the handler to run", ch)
}

// focus moves the focus to the nth input (from 1) with Tab, as inputs
// show no label the tester finds them by.
func focus(tt *ui.Tester, nth int) {
	for range nth {
		tt.Key(0, ui.KeyTab)
	}
	tt.Frame()
}

// typeInto types s at the end of the nth input.
func typeInto(t *testing.T, tt *ui.Tester, nth int, s string) {
	t.Helper()
	focus(tt, nth)
	tt.Key(0, ui.KeyEnd)
	tt.Type(s)
	tt.Frame()
}

func TestEntryTypingReachesTextAndOnChange(t *testing.T) {
	got := make(chan string, 8)
	var e *TextBoxWidget
	TextBox().SetText("ab").AssignTo(&e).OnChange(func() { got <- e.Text() })
	tt := tester(t, e)
	typeInto(t, tt, 1, "c")
	if s := received(t, got); s != "abc" {
		t.Fatalf("OnChange saw %q, want abc", s)
	}
	if e.Text() != "abc" {
		t.Fatalf("Text() %q", e.Text())
	}
}

func TestEntrySetTextShows(t *testing.T) {
	e := TextBox().SetText("one")
	tt := tester(t, e)
	done := make(chan struct{})
	go func() { e.SetText("two"); close(done) }()
	<-done
	tt.Frame()
	if e.text.shown != "two" { // what ui shows; the tester lists no input values
		t.Fatalf("shows %q", e.text.shown)
	}
}

func TestEntryOnEnter(t *testing.T) {
	entered := make(chan string, 1)
	var e *TextBoxWidget
	TextBox().SetText("go").AssignTo(&e).OnEnter(func() { entered <- e.Text() })
	tt := tester(t, e)
	typeInto(t, tt, 1, "!")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if s := received(t, entered); s != "go!" {
		t.Fatalf("OnEnter saw %q", s)
	}
}

func TestEntryOnFocusOut(t *testing.T) {
	left := make(chan struct{}, 1)
	a := TextBox().SetText("first")
	a.OnFocusOut(func() { left <- struct{}{} })
	b := TextBox().SetText("second")
	tt := tester(t, VBox(0).Add(a, b))
	focus(tt, 1) // first
	focus(tt, 1) // second
	received(t, left)
}

// U1: the caret follows the widget when the box reorders.
func TestEntryStateFollowsTheWidget(t *testing.T) {
	a, b := TextBox().SetText("aaa"), TextBox().SetText("bbb")
	box := VBox(0).Add(a, b)
	tt := tester(t, box)
	focus(tt, 1) // aaa
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	box.Clear().Add(b, a)
	tt.Frame()
	tt.Type("X")
	tt.Frame()
	if a.Text() != "aaaX" || b.Text() != "bbb" {
		t.Fatalf("a %q, b %q: typing went to the wrong entry", a.Text(), b.Text())
	}
}

// Review Focus 4: hiding keeps the value.
func TestHiddenEntryKeepsItsText(t *testing.T) {
	e := TextBox().SetText("kept")
	tt := tester(t, e)
	e.SetVisible(false)
	tt.Frame()
	e.SetVisible(true)
	tt.Frame()
	if e.text.shown != "kept" || e.Text() != "kept" {
		t.Fatalf("shows %q, Text %q", e.text.shown, e.Text())
	}
}

func TestTextViewTyping(t *testing.T) {
	v := MultilineTextBox().SetText("line")
	tt := tester(t, v)
	typeInto(t, tt, 1, "s")
	if v.Text() != "lines" {
		t.Fatalf("Text() %q", v.Text())
	}
}

func TestReadOnlyEntryIgnoresTyping(t *testing.T) {
	e := TextBox().SetText("fixed").SetEditable(false)
	tt := tester(t, e)
	typeInto(t, tt, 1, "x")
	if e.Text() != "fixed" {
		t.Fatalf("Text() %q", e.Text())
	}
}

// A widget shown in two views: focus in one is not lost in the other.
func TestOnFocusOutIgnoresAnotherView(t *testing.T) {
	left := make(chan struct{}, 4)
	e := TextBox().SetText("x")
	e.OnFocusOut(func() { left <- struct{}{} })
	tt1, tt2 := tester(t, e), tester(t, e)
	focus(tt1, 1)
	tt2.Frame()
	tt1.Frame()
	Update(func() {})
	select {
	case <-left:
		t.Fatal("OnFocusOut ran while the entry kept the focus in its view")
	default:
	}
}

func TestHidingAFocusedWidgetRunsOnFocusOut(t *testing.T) {
	left := make(chan struct{}, 1)
	e := TextBox()
	e.OnFocusOut(func() { left <- struct{}{} })
	tt := tester(t, e)
	focus(tt, 1)
	e.SetVisible(false)
	tt.Frame()
	received(t, left)
}

func TestFocusMovesTheFocus(t *testing.T) {
	a, b := TextBox(), TextBox()
	tt := tester(t, VBox(0).Add(a, b))
	b.Focus()
	tt.Frame()
	tt.Type("x")
	tt.Frame()
	if a.Text() != "" || b.Text() != "x" {
		t.Fatalf("a %q, b %q: typing did not go to b", a.Text(), b.Text())
	}
}

func TestTextBoxMaxLengthCutsTypingPastingAndSetText(t *testing.T) {
	tb := TextBox().MaxLength(4)
	tt := tester(t, tb)
	focus(tt, 1)
	tt.Type("héllo wörld") // runes, not bytes
	tt.Frame()
	if got := tb.Text(); got != "héll" {
		t.Fatalf("typed: %q, want héll", got)
	}
	tb.SetText("abcdefgh")
	tt.Frame()
	if got := tb.Text(); got != "abcd" {
		t.Fatalf("SetText past the limit: %q, want abcd", got)
	}
	tb.MaxLength(2) // a lower limit cuts what it holds
	tt.Frame()
	if got := tb.Text(); got != "ab" {
		t.Fatalf("MaxLength(2): %q, want ab", got)
	}
	tb.MaxLength(0).SetText("abcdefgh")
	tt.Frame()
	if got := tb.Text(); got != "abcdefgh" {
		t.Fatalf("no limit: %q", got)
	}
}
