package widget

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// source is a value a control is bound to, guarded as an app's config is,
// with the values set recorded in order.
type source[T comparable] struct {
	mu   sync.Mutex
	v    T
	sets []T
}

func newSource[T comparable](v T) *source[T] { return &source[T]{v: v} }

func (s *source[T]) get() T { s.mu.Lock(); defer s.mu.Unlock(); return s.v }

func (s *source[T]) set(v T) {
	s.mu.Lock()
	s.v = v
	s.sets = append(s.sets, v)
	s.mu.Unlock()
}

func (s *source[T]) change(v T) { s.mu.Lock(); s.v = v; s.mu.Unlock() }

// calls returns the values set so far, after the handlers Posted so far ran:
// a test that expects no call, or one, sees a wrong count at once instead of
// waiting out a guard.
func (s *source[T]) calls() []T {
	Update(func() {})
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.sets)
}

// only returns the one value set, and fails if there were none or more.
func only[T comparable](t *testing.T, s *source[T]) T {
	t.Helper()
	got := s.calls()
	if len(got) != 1 {
		t.Fatalf("set was called %d times (%v), want once", len(got), got)
	}
	return got[0]
}

func TestBindTextBoxTakesTheSourceAtOnce(t *testing.T) {
	src := newSource("hello")
	tb := Bind(TextBox(), src.get, src.set)
	if tb.Text() != "hello" {
		t.Fatalf("Text %q, want the source's hello", tb.Text())
	}
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("binding wrote %v back to the source", got)
	}
}

func TestBindTextBoxUserEditSetsOnce(t *testing.T) {
	src := newSource("ab")
	tb := Bind(TextBox(), src.get, src.set)
	tt := tester(t, tb)
	typeInto(t, tt, 1, "c")
	if v := only(t, src); v != "abc" {
		t.Fatalf("set got %q, want abc", v)
	}
}

// OnChange that normalizes the text, as one that trims or upcases does, is
// no program setter's value for set: set gets what the user entered.
func TestBindSetGetsTheUsersValueWhenOnChangeSetsText(t *testing.T) {
	src := newSource("")
	var tb *TextBoxWidget
	tb = TextBox().OnChange(func() { tb.SetText(strings.ToUpper(tb.Text()) + "!") })
	Bind(tb, src.get, src.set)
	tt := tester(t, tb)
	typeInto(t, tt, 1, "q")
	if v := only(t, src); v != "q" {
		t.Fatalf("set got %q, want the user's q, not what OnChange made of it", v)
	}
	if tb.Text() != "Q!" {
		t.Fatalf("Text %q, OnChange's SetText must still show", tb.Text())
	}
}

// A program setter that runs between the user's edit and the handler that
// writes it must not take the user's place.
func TestBindSetGetsTheUsersValueWhenAWorkerSetsTextFirst(t *testing.T) {
	src := newSource("")
	tb := Bind(TextBox(), src.get, src.set)
	tt := tester(t, tb)
	release := make(chan struct{})
	Post(func() { <-release }) // holds the event goroutine: the edit's handler waits behind it
	typeInto(t, tt, 1, "u")
	tb.SetText("prog") // another goroutine's setter, before the handler runs
	close(release)
	if v := only(t, src); v != "u" {
		t.Fatalf("set got %q, want the user's u", v)
	}
}

func TestBindProgramSetterDoesNotSet(t *testing.T) {
	src := newSource("a")
	tb := Bind(TextBox(), src.get, src.set)
	tt := tester(t, tb)
	tb.SetText("b")
	tt.Frame()
	tt.Frame()
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("SetText called set with %q: a program setter must not", got)
	}
	if src.get() != "a" || tb.Text() != "b" {
		t.Fatalf("source %q, control %q", src.get(), tb.Text())
	}
}

func TestBindDoesNotReplaceOnChange(t *testing.T) {
	src := newSource("")
	var order []string
	var mu sync.Mutex
	note := func(s string) { mu.Lock(); order = append(order, s); mu.Unlock() }
	tb := TextBox().OnChange(func() { note("OnChange") })
	Bind(tb, src.get, func(v string) { note("set " + v); src.set(v) })
	tt := tester(t, tb)
	typeInto(t, tt, 1, "x")
	only(t, src)
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(order, []string{"OnChange", "set x"}) {
		t.Fatalf("order %q: OnChange, then set", order)
	}
}

func TestBindOnChangeSetAfterBindingStillRuns(t *testing.T) {
	src := newSource("")
	ran := 0
	tb := Bind(TextBox(), src.get, src.set)
	tb.OnChange(func() { ran++ })
	tt := tester(t, tb)
	typeInto(t, tt, 1, "x")
	only(t, src)
	if ran != 1 { // the event goroutine ran it before set, and only(t, src) waited for both
		t.Fatalf("OnChange ran %d times", ran)
	}
}

func TestReloadTakesTheChangedSource(t *testing.T) {
	src := newSource("one")
	tb := Bind(TextBox(), src.get, src.set)
	tt := tester(t, tb)
	src.change("two") // the source changed elsewhere
	Reload(tb)
	tt.Frame()
	if tb.Text() != "two" {
		t.Fatalf("Text %q after Reload, want two", tb.Text())
	}
	if got := src.calls(); len(got) != 0 {
		t.Fatalf("Reload called set with %q", got)
	}
}

func TestReloadOfAControlNeverBoundDoesNothing(t *testing.T) {
	tb := TextBox().SetText("keep")
	Reload(tb)
	Reload(Label("x"))
	if tb.Text() != "keep" {
		t.Fatalf("Text %q", tb.Text())
	}
}

func TestBindingAgainReplacesTheFirst(t *testing.T) {
	first, second := newSource("a"), newSource("b")
	tb := Bind(TextBox(), first.get, first.set)
	Bind(tb, second.get, second.set)
	if tb.Text() != "b" {
		t.Fatalf("Text %q, want the second source's b", tb.Text())
	}
	tt := tester(t, tb)
	typeInto(t, tt, 1, "!")
	only(t, second)
	if got := first.calls(); len(got) != 0 {
		t.Fatalf("the replaced binding was called with %q", got)
	}
	first.change("zzz")
	Reload(tb)
	if tb.Text() != "b!" {
		t.Fatalf("Reload read the replaced binding: Text %q", tb.Text())
	}
}

// set runs off the package lock: it may read the control it is bound to.
func TestBindSetMayReadItsControl(t *testing.T) {
	seen := make(chan string, 4)
	var tb *TextBoxWidget
	tb = Bind(TextBox(), func() string { return "" }, func(string) { seen <- tb.Text() })
	tt := tester(t, tb)
	typeInto(t, tt, 1, "q")
	if v := received(t, seen); v != "q" {
		t.Fatalf("set read %q from the control, want q", v)
	}
}

func TestBindCheckBox(t *testing.T) {
	src := newSource(true)
	cb := Bind(CheckBox("Remember"), src.get, src.set)
	if !cb.Checked() {
		t.Fatal("not checked from the source")
	}
	tt := tester(t, cb)
	click(t, tt, "Remember")
	if v := only(t, src); v {
		t.Fatalf("set got %v, want false after unchecking", v)
	}
	cb.SetChecked(true)
	tt.Frame()
	src.change(false)
	Reload(cb)
	if cb.Checked() || len(src.calls()) != 1 {
		t.Fatalf("checked %v, set calls %v", cb.Checked(), src.calls())
	}
}

func TestBindComboBox(t *testing.T) {
	src := newSource("apple")
	cb := Bind(ComboBox(Option{"apple", "Apple"}, Option{"kale", "Kale"}), src.get, src.set)
	if cb.Value() != "apple" {
		t.Fatalf("Value %q, want apple", cb.Value())
	}
	tt := tester(t, cb)
	click(t, tt, "Apple")
	click(t, tt, "Kale")
	if v := only(t, src); v != "kale" {
		t.Fatalf("set got %q", v)
	}
	if got := src.calls(); !slices.Equal(got, []string{"kale"}) {
		t.Fatalf("set calls %q", got)
	}
	cb.SetValue("apple")
	tt.Frame()
	if got := src.calls(); len(got) != 1 {
		t.Fatalf("SetValue called set: %q", got)
	}
}

// A value with no option leaves the combo empty, as SetValue does, and is
// not written back.
func TestBindComboBoxUnknownValueKeepsItEmpty(t *testing.T) {
	src := newSource("nope")
	cb := Bind(ComboBox(Option{"apple", "Apple"}), src.get, src.set)
	tt := tester(t, cb)
	tt.Frame()
	if cb.Value() != "" {
		t.Fatalf("Value %q, want none", cb.Value())
	}
	if got := src.calls(); len(got) != 0 || src.get() != "nope" {
		t.Fatalf("source %q, set calls %q: an unknown value must not be written back", src.get(), got)
	}
}

func TestBindNumericUpDown(t *testing.T) {
	src := newSource(5.0)
	n := Bind(NumericUpDown(0, 100, 1), src.get, src.set)
	if n.Value() != 5 {
		t.Fatalf("Value %v", n.Value())
	}
	tt := tester(t, n)
	focus(tt, 1)
	tt.Key(0, ui.KeyEnd)
	tt.Type("0")
	tt.Frame()
	if v := only(t, src); v != 50 {
		t.Fatalf("set got %v, want 50", v)
	}
}

// The source's value is clamped to the range as the control takes it, and
// the clamped value is not written back.
func TestBindNumericUpDownClampsWithoutWritingBack(t *testing.T) {
	src := newSource(500.0)
	n := Bind(NumericUpDown(0, 100, 1), src.get, src.set)
	tt := tester(t, n)
	tt.Frame()
	if n.Value() != 100 {
		t.Fatalf("Value %v, want 100", n.Value())
	}
	if got := src.calls(); len(got) != 0 || src.get() != 500 {
		t.Fatalf("source %v, set calls %v: a clamp must not be written back", src.get(), got)
	}
}

func TestBindIntNumericUpDown(t *testing.T) {
	src := newSource(7)
	n := BindInt(NumericUpDown(0, 100, 1), src.get, src.set)
	if n.Int() != 7 {
		t.Fatalf("Int %d", n.Int())
	}
	tt := tester(t, n)
	focus(tt, 1)
	tt.Key(0, ui.KeyEnd)
	tt.Type("2")
	tt.Frame()
	if v := only(t, src); v != 72 {
		t.Fatalf("set got %d, want 72", v)
	}
	src.change(9)
	Reload(n)
	if n.Int() != 9 {
		t.Fatalf("Int %d after Reload", n.Int())
	}
}

// get and set touch a guarded map while frames build, edits arrive and the
// source is reloaded from another goroutine (go test -race).
func TestBindFromManyGoroutines(t *testing.T) {
	var mu sync.Mutex
	config := map[string]string{"k": ""}
	get := func() string { mu.Lock(); defer mu.Unlock(); return config["k"] }
	set := func(v string) { mu.Lock(); config["k"] = v; mu.Unlock() }
	tb := Bind(TextBox(), get, set)
	tt := tester(t, tb)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 100 {
			mu.Lock()
			config["k"] = string(rune('a' + i%26))
			mu.Unlock()
			Reload(tb)
		}
	}()
	focus(tt, 1)
	for range 20 {
		tt.Type("x")
		tt.Frame()
	}
	<-done
	Update(func() {})
}
