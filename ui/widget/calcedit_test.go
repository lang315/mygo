package widget

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// calcTester shows ce, 300 wide, in a window tall enough for its
// calculator, and returns its text field.
func calcTester(t *testing.T, ce *CalcEditWidget) (*Tester, *Node) {
	t.Helper()
	calmOff()
	tt := NewTester(t, ce.SetSize(300, 0), 400, 500)
	tt.Settle()
	return tt, tt.Find(ByRole(ui.RoleTextField))
}

// calcOpen reports whether the calculator shows.
func calcOpen(tt *Tester) bool { return tt.Has(ByRole(ui.RoleButton).Label("=")) }

// calcKey is the calculator's button named name.
func calcKey(tt *Tester, name string) *Node { return tt.Find(ByRole(ui.RoleButton).Label(name)) }

// openCalc opens the calculator with the button at the field's end.
func openCalc(t *testing.T, tt *Tester) {
	t.Helper()
	calcKey(tt, l10n.Open()).Click()
	if !calcOpen(tt) {
		t.Fatal("the button at the field's end did not open the calculator")
	}
}

// calcType types s into the open calculator.
func calcType(tt *Tester, s string) { tt.Tester.Type(s); tt.Settle() }

// calcPress presses key in the open calculator.
func calcPress(tt *Tester, key ui.Key) { tt.Tester.Key(0, key); tt.Settle() }

// calcShows is what the calculator's display shows.
func calcShows(ce *CalcEditWidget) string { mu.Lock(); defer mu.Unlock(); return ce.shows() }

func TestCalcArithmetic(t *testing.T) {
	cases := []struct {
		name, keys string
		enter      bool    // Enter after keys
		shows      string  // the display before Enter ("" for not checked)
		want       float64 // the value after
	}{
		{"chained left to right", "12+3*2=", false, "", 30},
		{"division", "10/4=", false, "", 2.5},
		{"times percent", "200*50%", true, "100", 100},
		{"plus percent", "200+10%", true, "220", 220},
		{"minus percent", "200-10%", true, "180", 180},
		{"divide percent", "200/50%", true, "400", 400},
		{"lone percent", "50%", true, "0.5", 0.5},
		{"decimals typed", "1.5+1.25=", false, "", 2.75},
		{"a leading point", ".5=", false, "", 0.5},
		{"a second point is ignored", "1..2=", false, "", 1.2},
		{"an operator replaces the last", "6+*2=", false, "", 12},
		{"the signs of other keyboards", "6×2÷3−1=", false, "", 3},
		{"Enter computes", "7-2", true, "2", 5},
		{"equals with no operator", "42=", false, "", 42},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ce := CalcEdit()
			tt, _ := calcTester(t, ce)
			openCalc(t, tt)
			calcType(tt, tc.keys)
			if tc.shows != "" {
				if s := calcShows(ce); s != tc.shows {
					t.Fatalf("after %q the display shows %q, want %q", tc.keys, s, tc.shows)
				}
				if !tt.Has(ByLabel(tc.shows)) {
					t.Fatalf("after %q no %q is on screen", tc.keys, tc.shows)
				}
			}
			if tc.enter {
				calcPress(tt, ui.KeyEnter)
			}
			if v := ce.Value(); v != tc.want {
				t.Fatalf("%q gave %v, want %v", tc.keys, v, tc.want)
			}
			if calcOpen(tt) {
				t.Fatalf("after %q the calculator is still open", tc.keys)
			}
		})
	}

	t.Run("plus-minus negates", func(t *testing.T) {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		calcType(tt, "12")
		calcKey(tt, "±").Click()
		if s := calcShows(ce); s != "-12" {
			t.Fatalf("± after 12 shows %q, want -12", s)
		}
		calcKey(tt, "±").Click()
		if s := calcShows(ce); s != "12" {
			t.Fatalf("± twice shows %q, want 12", s)
		}
		calcKey(tt, "±").Click()
		calcType(tt, "+2=")
		if v := ce.Value(); v != -10 {
			t.Fatalf("-12+2 gave %v, want -10", v)
		}
		// ± on a result negates it.
		openCalc(t, tt)
		calcKey(tt, "±").Click()
		calcPress(tt, ui.KeyEnter)
		if v := ce.Value(); v != 10 {
			t.Fatalf("± on -10 gave %v, want 10", v)
		}
	})

	t.Run("backspace removes a digit", func(t *testing.T) {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		calcType(tt, "123")
		calcPress(tt, ui.KeyBackspace)
		if s := calcShows(ce); s != "12" {
			t.Fatalf("Backspace after 123 shows %q, want 12", s)
		}
		calcPress(tt, ui.KeyBackspace)
		calcPress(tt, ui.KeyBackspace)
		calcPress(tt, ui.KeyBackspace) // nothing left: no change
		if s := calcShows(ce); s != "0" {
			t.Fatalf("Backspace past the last digit shows %q, want 0", s)
		}
		calcType(tt, "5=")
		if v := ce.Value(); v != 5 {
			t.Fatalf("5 after the Backspaces gave %v", v)
		}
		// The second number emptied is 0, not the first again.
		openCalc(t, tt)
		calcType(tt, "12+34")
		calcPress(tt, ui.KeyBackspace)
		calcPress(tt, ui.KeyBackspace)
		if s := calcShows(ce); s != "0" {
			t.Fatalf("12+34, Backspace twice shows %q, want 0", s)
		}
		calcType(tt, "=")
		if v := ce.Value(); v != 12 {
			t.Fatalf("12+34, Backspace twice, = gave %v, want 12", v)
		}
	})

	t.Run("C clears all", func(t *testing.T) {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		calcType(tt, "12+3")
		calcKey(tt, "C").Click()
		if s := calcShows(ce); s != "0" {
			t.Fatalf("C shows %q, want 0", s)
		}
		calcType(tt, "5=")
		if v := ce.Value(); v != 5 {
			t.Fatalf("C then 5= gave %v, want 5: the pending 12+ survived", v)
		}
	})

	t.Run("CE clears the entry", func(t *testing.T) {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		calcType(tt, "12+3")
		calcPress(tt, ui.KeyDelete)
		if s := calcShows(ce); s != "0" {
			t.Fatalf("CE (Delete) shows %q, want 0", s)
		}
		calcType(tt, "4=")
		if v := ce.Value(); v != 16 {
			t.Fatalf("12+3, CE, 4= gave %v, want 16", v)
		}
	})

	t.Run("it starts from the field's value", func(t *testing.T) {
		ce := CalcEdit().SetValue(7)
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		if s := calcShows(ce); s != "7" {
			t.Fatalf("the calculator opens showing %q, want the field's 7", s)
		}
		calcType(tt, "*3=")
		if v := ce.Value(); v != 21 {
			t.Fatalf("7, *3= gave %v, want 21", v)
		}
		openCalc(t, tt)
		calcType(tt, "4=") // a digit starts a new number
		if v := ce.Value(); v != 4 {
			t.Fatalf("21, 4= gave %v, want 4", v)
		}
	})
}

func TestCalcBounds(t *testing.T) {
	t.Run("sixteen digits at most", func(t *testing.T) {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		calcType(tt, "12345678901234567890")
		if s := calcShows(ce); s != "1234567890123456" {
			t.Fatalf("twenty digits show %q, want the first sixteen", s)
		}
	})
	t.Run("keys typed after = go nowhere", func(t *testing.T) {
		ce := CalcEdit()
		tt, in := calcTester(t, ce)
		openCalc(t, tt)
		calcType(tt, "12=34=")
		if v := ce.Value(); v != 12 {
			t.Fatalf("12=34= gave %v, want 12", v)
		}
		if v := in.Value(); v != "12.00" {
			t.Fatalf("12=34= left the field at %q, want 12.00", v)
		}
	})
	t.Run("hiding closes it", func(t *testing.T) {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		ce.SetVisible(false)
		tt.Settle()
		ce.SetVisible(true)
		tt.Settle()
		if calcOpen(tt) {
			t.Fatal("the calculator shows again after the field was hidden")
		}
	})
}

func TestCalcDivideByZero(t *testing.T) {
	for _, keys := range []string{"5/0=", "0/0=", "5/0%", "0/0%", "5/0+"} {
		t.Run(keys, func(t *testing.T) {
			var log editLog
			ce := CalcEdit().SetValue(7).OnChange(log.add("change"))
			tt, in := calcTester(t, ce)
			openCalc(t, tt)
			calcType(tt, keys)
			if s := calcShows(ce); s != "E" {
				t.Fatalf("%q shows %q, want the error E", keys, s)
			}
			if !calcOpen(tt) {
				t.Fatalf("%q closed the calculator", keys)
			}
			calcPress(tt, ui.KeyEnter)
			calcType(tt, "3=")
			if !calcOpen(tt) {
				t.Fatal("Enter or = in the error closed the calculator")
			}
			if v := ce.Value(); v != 7 {
				t.Fatalf("the error changed the value to %v", v)
			}
			if v := in.Value(); v != "7.00" {
				t.Fatalf("the error changed the field to %q", v)
			}
			if s := calcShows(ce); s != "E" {
				t.Fatalf("digits after the error show %q, want E until C", s)
			}
			calcKey(tt, "C").Click()
			if s := calcShows(ce); s != "0" {
				t.Fatalf("C after the error shows %q, want 0", s)
			}
			calcType(tt, "3=")
			if v := ce.Value(); v != 3 {
				t.Fatalf("after C, 3= gave %v, want 3", v)
			}
			if got := log.all(t); !slices.Equal(got, []string{"change"}) {
				t.Fatalf("ran %q, want one change, for 3=", got)
			}
		})
	}
}

func TestCalcEnterAndEscape(t *testing.T) {
	var log editLog
	ce := CalcEdit().SetValue(7).OnChange(log.add("change"))
	tt, in := calcTester(t, ce)
	openCalc(t, tt)
	calcType(tt, "+1")
	calcPress(tt, ui.KeyEnter)
	if v := ce.Value(); v != 8 {
		t.Fatalf("7+1 Enter gave %v, want 8", v)
	}
	if v := in.Value(); v != "8.00" {
		t.Fatalf("the field shows %q, want 8.00", v)
	}
	if calcOpen(tt) {
		t.Fatal("Enter left the calculator open")
	}
	if !in.Focused() {
		t.Fatal("after Enter the keyboard is not back in the field")
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("Enter ran %q, want one change", got)
	}

	log.reset()
	openCalc(t, tt)
	calcType(tt, "99")
	calcPress(tt, ui.KeyEscape)
	if calcOpen(tt) {
		t.Fatal("Escape left the calculator open")
	}
	if v := ce.Value(); v != 8 {
		t.Fatalf("Escape changed the value to %v", v)
	}
	if v := in.Value(); v != "8.00" {
		t.Fatalf("after Escape the field shows %q", v)
	}
	if !in.Focused() {
		t.Fatal("after Escape the keyboard is not back in the field")
	}

	// Enter with the value unchanged changes nothing.
	openCalc(t, tt)
	calcPress(tt, ui.KeyEnter)
	if calcOpen(tt) {
		t.Fatal("Enter on the value as it was left the calculator open")
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("Escape, and Enter on the same value, ran %q", got)
	}

	// The button at the end closes it again, keeping the value.
	openCalc(t, tt)
	calcType(tt, "5")
	calcKey(tt, l10n.Open()).Click()
	if calcOpen(tt) || ce.Value() != 8 {
		t.Fatalf("the button closed it: open %v, value %v, want closed, 8", calcOpen(tt), ce.Value())
	}
}

// In a form, Enter and Escape in the calculator are the calculator's, not
// the form's accept and cancel buttons'.
func TestCalcKeysInAForm(t *testing.T) {
	calmOff()
	var log editLog
	f := NewForm("T", 400, 500)
	ok := Button("OK").OnClick(log.add("ok"))
	cancel := Button("Cancel").OnClick(log.add("cancel"))
	ce := CalcEdit().OnChange(log.add("change"))
	f.SetContent(VBox(0).Add(ce, ok, cancel)).SetAcceptButton(ok).SetCancelButton(cancel)
	tt := TestForm(t, f)
	tt.Settle()
	openCalc(t, tt)
	calcType(tt, "2+2")
	calcPress(tt, ui.KeyEnter)
	openCalc(t, tt)
	calcPress(tt, ui.KeyEscape)
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("Enter and Escape in the calculator ran %q, want the change alone", got)
	}
	if v := ce.Value(); v != 4 {
		t.Fatalf("2+2 Enter in a form gave %v", v)
	}
}

// Alt+Down and Ctrl+Enter in the field open the calculator.
func TestCalcOpenByKey(t *testing.T) {
	for _, k := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{ui.Alt, ui.KeyDown}, {ui.Ctrl, ui.KeyEnter}} {
		ce := CalcEdit()
		tt, in := calcTester(t, ce)
		in.Key(k.mods, k.key)
		if !calcOpen(tt) {
			t.Fatalf("%v+%v in the field did not open the calculator", k.mods, k.key)
		}
		calcType(tt, "6*7")
		calcPress(tt, ui.KeyEnter)
		if v := ce.Value(); v != 42 {
			t.Fatalf("6*7 by keys gave %v", v)
		}
	}
}

func TestCalcDecimals(t *testing.T) {
	ce := CalcEdit().SetDecimals(0)
	tt, in := calcTester(t, ce)
	openCalc(t, tt)
	calcType(tt, "2.5+0.5=")
	if v := in.Value(); v != "3" {
		t.Fatalf("SetDecimals(0): 2.5+0.5 shows %q, want 3", v)
	}
	ce.SetDecimals(3)
	tt.Settle()
	if v := in.Value(); v != "3.000" {
		t.Fatalf("SetDecimals(3) shows %q, want 3.000", v)
	}
	openCalc(t, tt)
	calcType(tt, "1/3+")
	if s := calcShows(ce); s != "0.333" {
		t.Fatalf("SetDecimals(3): 1/3 shows %q in the calculator, want 0.333", s)
	}
	calcType(tt, "0=")
	if v := in.Value(); v != "0.333" {
		t.Fatalf("SetDecimals(3): 1/3 shows %q, want 0.333", v)
	}
	ce.SetDecimals(-2) // below 0 is 0
	tt.Settle()
	if v := in.Value(); v != "0" {
		t.Fatalf("SetDecimals(-2) shows %q, want 0", v)
	}
	ce.SetValue(-0.001).SetDecimals(2)
	tt.Settle()
	if v := in.Value(); v != "0.00" {
		t.Fatalf("-0.001 with 2 decimals shows %q, want 0.00", v)
	}
	openCalc(t, tt)
	if s := calcShows(ce); s != "0" {
		t.Fatalf("-0.001 with 2 decimals shows %q in the calculator, want 0", s)
	}
}

// The field takes typed numbers as NumericUpDown does: what parses is the
// value at once; leaving the field shows it rounded, and puts back the last
// value where the text does not parse.
func TestCalcTypedNumber(t *testing.T) {
	var log editLog
	ce := CalcEdit().OnChange(log.add("change"))
	calmOff()
	tt := NewTester(t, VBox(0).Add(ce, Button("Next")), 400, 500)
	tt.Settle()
	in := tt.Find(ByRole(ui.RoleTextField))
	in.Key(ui.Cmd, ui.KeyA)
	in.Type("12.5")
	if v := ce.Value(); v != 12.5 {
		t.Fatalf("typing 12.5 gave %v", v)
	}
	in.Key(0, ui.KeyTab)
	if v := in.Value(); v != "12.50" {
		t.Fatalf("left, the field shows %q, want 12.50", v)
	}
	n := len(log.all(t))
	if n == 0 {
		t.Fatal("typing ran no change")
	}
	for _, bad := range []string{"abc", "nan", "inf", "1e999"} {
		log.reset()
		in.Key(ui.Cmd, ui.KeyA)
		in.Type(bad)
		in.Key(0, ui.KeyTab)
		if v := ce.Value(); v != 12.5 {
			t.Fatalf("typing %q gave %v, want 12.5 kept", bad, v)
		}
		if v := in.Value(); v != "12.50" {
			t.Fatalf("after %q the field shows %q, want 12.50 back", bad, v)
		}
		if got := log.all(t); len(got) != 0 {
			t.Fatalf("typing %q ran %q", bad, got)
		}
	}
	// A value with more decimals than shown is not rounded by focusing and
	// leaving the field.
	log.reset()
	ce.SetValue(1.0 / 3)
	tt.Settle()
	in.Focus()
	in.Key(0, ui.KeyTab)
	if v := ce.Value(); v != 1.0/3 {
		t.Fatalf("focus and Tab changed 1/3 to %v", v)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("focus and Tab ran %q", got)
	}
}

func TestCalcClicks(t *testing.T) {
	// Each button by pointer, against the key it stands for.
	cases := []struct {
		clicks []string
		keys   string
		shows  string
	}{
		{[]string{"1", "2", "+", "3", "×", "2", "+"}, "12+3*2+", "30"},
		{[]string{"4", "5", "6", "7", "8", "9", "0"}, "4567890", "4567890"},
		{[]string{"1", "0", "÷", "4", "−", "1", "+"}, "10/4-1+", "1.5"},
		{[]string{"2", "0", "0", "×", "5", "0", "%"}, "200*50%", "100"},
		{[]string{"1", ".", "5"}, "1.5", "1.5"},
	}
	for _, tc := range cases {
		ce := CalcEdit()
		tt, _ := calcTester(t, ce)
		openCalc(t, tt)
		for _, k := range tc.clicks {
			calcKey(tt, k).Click()
		}
		if s := calcShows(ce); s != tc.shows {
			t.Fatalf("clicking %q shows %q, want %q", tc.clicks, s, tc.shows)
		}
		ce2 := CalcEdit()
		tt2, _ := calcTester(t, ce2)
		openCalc(t, tt2)
		calcType(tt2, tc.keys)
		if s := calcShows(ce2); s != tc.shows {
			t.Fatalf("typing %q shows %q, want %q", tc.keys, s, tc.shows)
		}
	}

	ce := CalcEdit()
	tt, in := calcTester(t, ce)
	openCalc(t, tt)
	calcKey(tt, "9").Click()
	calcKey(tt, "8").Click()
	calcKey(tt, "⌫").Click()
	if s := calcShows(ce); s != "9" {
		t.Fatalf("⌫ after 98 shows %q, want 9", s)
	}
	calcKey(tt, "+").Click()
	calcKey(tt, "5").Click()
	calcKey(tt, "CE").Click()
	if s := calcShows(ce); s != "0" {
		t.Fatalf("CE shows %q, want 0", s)
	}
	calcKey(tt, "1").Click()
	calcKey(tt, "=").Click()
	if v := ce.Value(); v != 10 || calcOpen(tt) {
		t.Fatalf("9+5, CE, 1, = by clicks: value %v, open %v; want 10, closed", ce.Value(), calcOpen(tt))
	}
	if !in.Focused() {
		t.Fatal("after = by click the keyboard is not back in the field")
	}
}

func TestCalcSetterDoesNotFire(t *testing.T) {
	var log editLog
	ce := CalcEdit().OnChange(log.add("change"))
	tt, in := calcTester(t, ce)
	ce.SetValue(12.346)
	tt.Settle()
	if v := in.Value(); v != "12.35" {
		t.Fatalf("SetValue(12.346) shows %q, want 12.35", v)
	}
	ce.SetDecimals(1).ShowPopup()
	tt.Settle()
	if !calcOpen(tt) {
		t.Fatal("ShowPopup did not open the calculator")
	}
	if s := calcShows(ce); s != "12.3" {
		t.Fatalf("the calculator opened by ShowPopup shows %q, want 12.3", s)
	}
	calcType(tt, "+1")
	ce.ClosePopup()
	tt.Settle()
	if calcOpen(tt) {
		t.Fatal("ClosePopup left the calculator open")
	}
	if v := ce.Value(); v != 12.346 {
		t.Fatalf("ClosePopup changed the value to %v", v)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("setters ran %q", got)
	}
}

func TestCalcFromGoroutine(t *testing.T) {
	ce := CalcEdit()
	tt, in := calcTester(t, ce)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			ce.SetValue(float64(i)).SetDecimals(i % 4)
			if i%2 == 0 {
				ce.ShowPopup()
			} else {
				ce.ClosePopup()
			}
			_ = ce.Value()
		}
		ce.SetDecimals(2).SetValue(42).ClosePopup()
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	if v := in.Value(); v != "42.00" {
		t.Fatalf("the field shows %q, want 42.00", v)
	}
	if calcOpen(tt) {
		t.Fatal("the calculator shows after ClosePopup from another goroutine")
	}
	ce.ShowPopup()
	tt.Settle()
	if s := calcShows(ce); s != "42" || !calcOpen(tt) {
		t.Fatalf("ShowPopup from another goroutine: open %v, shows %q", calcOpen(tt), s)
	}
}

func TestCalcBind(t *testing.T) {
	var mu sync.Mutex
	v := 5.0
	get := func() float64 { mu.Lock(); defer mu.Unlock(); return v }
	ce := Bind(CalcEdit(), get, func(x float64) { mu.Lock(); v = x; mu.Unlock() })
	tt, in := calcTester(t, ce)
	if s := in.Value(); s != "5.00" {
		t.Fatalf("the field shows %q, not the source's 5", s)
	}
	in.Key(ui.Cmd, ui.KeyA)
	in.Type("8")
	flushEvents(t)
	if x := get(); x != 8 {
		t.Fatalf("after typing the source has %v", x)
	}
	openCalc(t, tt)
	calcType(tt, "*2")
	calcPress(tt, ui.KeyEnter)
	flushEvents(t)
	if x := get(); x != 16 {
		t.Fatalf("after the calculator the source has %v, want 16", x)
	}
	mu.Lock()
	v = 3
	mu.Unlock()
	Reload(ce)
	tt.Settle()
	if s := in.Value(); s != "3.00" {
		t.Fatalf("after Reload the field shows %q", s)
	}
}

// In RTL the button sits at the field's start, the left.
func TestCalcRTL(t *testing.T) {
	ce := CalcEdit()
	tt, in := calcTester(t, ce)
	tt.SetDirection(ui.RightToLeft)
	tt.Settle()
	if b, f := calcKey(tt, l10n.Open()).Bounds(), in.Bounds(); b.X+b.W > f.X+0.5 {
		t.Fatalf("in RTL the button at %v is not left of the text at %v", b, f)
	}
	openCalc(t, tt)
	calcType(tt, "1+1=")
	if v := ce.Value(); v != 2 {
		t.Fatalf("1+1= in RTL gave %v", v)
	}
}

func TestCalcAllocNothingAtRest(t *testing.T) {
	calmOff()
	box := ui.NewTester(content(TextBox().SetText("hello")), 400, 120)
	edit := ui.NewTester(content(CalcEdit().SetValue(12.5).OnChange(func() {})), 400, 120)
	for range 10 {
		box.Frame()
		edit.Frame()
	}
	want, got := allocrun.Least(30, box.Frame), allocrun.Least(30, edit.Frame)
	if got > want {
		t.Fatalf("a frame of the CalcEdit at rest allocates %v, more than a TextBox's %v", got, want)
	}
	t.Logf("a frame allocates %v, a TextBox's %v", got, want)
}

var _ = fmt.Sprint
