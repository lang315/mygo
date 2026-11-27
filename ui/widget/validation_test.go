package widget

import (
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

func TestRuleRequired(t *testing.T) {
	ep := ErrorProvider()
	tb := Rule(ep, TextBox(), Required("Name is required"))
	tt := NewTester(t, VBox(8).Add(Label("Name"), tb), 400, 200)
	box := tt.Find(ByRole(ui.RoleTextField)).Bounds()
	before := redPixels(tt, box)
	if ep.Validate() {
		t.Error("Validate said true with an empty TextBox")
	}
	tt.Settle()
	if got := ep.Error(tb); got != "Name is required" {
		t.Errorf("Error = %q, want the rule's message", got)
	}
	if redPixels(tt, box) <= before {
		t.Error("no mark on the empty TextBox")
	}
	tb.SetText("   ")
	if ep.Validate() || ep.Error(tb) != "Name is required" {
		t.Error("text of spaces alone passed Required")
	}
	tt.Find(ByRole(ui.RoleTextField)).Focus().Type("Ann")
	if !ep.Validate() {
		t.Error("Validate said false once the TextBox has text")
	}
	tt.Settle()
	if ep.Error(tb) != "" || ep.HasErrors() || redPixels(tt, box) != before {
		t.Errorf("the mark stayed: Error %q", ep.Error(tb))
	}
}

func TestRuleFirstFailingWins(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox()
	Rule(ep, tb, Required("required"))
	Rule(ep, tb, MatchRegexp(regexp.MustCompile(`^\d+$`), "digits only"))
	Rule(ep, tb, func(string) string { return "always" })
	NewTester(t, tb, 400, 100)
	ep.Validate()
	if got := ep.Error(tb); got != "required" {
		t.Errorf("empty: Error = %q, want the first rule's", got)
	}
	tb.SetText("abc")
	ep.Validate()
	if got := ep.Error(tb); got != "digits only" {
		t.Errorf("abc: Error = %q, want the second rule's", got)
	}
	tb.SetText("42")
	ep.Validate()
	if got := ep.Error(tb); got != "always" {
		t.Errorf("42: Error = %q, want the third rule's", got)
	}
}

// Each rule runs once a Validate, however many rules its control has; a
// provider's Validate runs its own rules only.
func TestRuleValidateRunsEachRuleOnce(t *testing.T) {
	ep, other := ErrorProvider(), ErrorProvider()
	a, b := TextBox(), TextBox()
	runs := 0
	pass := func(string) string { runs++; return "" }
	Rule(ep, a, pass)
	Rule(ep, a, pass)
	Rule(other, b, Required("b required"))
	Rule(other, a, Required("a required by other"))
	NewTester(t, VBox(4).Add(a, b), 400, 200)
	if !ep.Validate() || runs != 2 {
		t.Errorf("Validate ran the rules %d times, want 2", runs)
	}
	if other.Error(a) != "" || other.Error(b) != "" {
		t.Error("a provider's Validate ran another provider's rule")
	}
}

func TestRuleInRange(t *testing.T) {
	ep := ErrorProvider()
	n := Rule(ep, NumericUpDown(0, 100, 1), InRange(1, 10, "1 to 10"))
	NewTester(t, n, 400, 100)
	for _, c := range []struct {
		v  float64
		ok bool
	}{{0, false}, {5, true}} {
		n.SetValue(c.v)
		ok := ep.Validate()
		msg := ep.Error(n)
		if ok != c.ok || (msg == "1 to 10") == c.ok {
			t.Errorf("value %v: Validate %v, Error %q; want ok %v", c.v, ok, msg, c.ok)
		}
	}
	// The bounds are in the range; just past them is not.
	in := InRange(1, 10, "m")
	for v, want := range map[float64]string{0.999: "m", 1: "", 10: "", 10.001: "m", -5: "m"} {
		if got := in(v); got != want {
			t.Errorf("InRange(1, 10)(%v) = %q, want %q", v, got, want)
		}
	}
}

// A ruled control the app dropped is let go, as a marked one is.
func TestRuleLetsGoOfControlsTheAppDropped(t *testing.T) {
	ep := ErrorProvider()
	func() {
		tb := Rule(ep, TextBox(), Required("x"))
		ep.SetError(tb, "gone soon")
	}()
	for range 5 {
		runtime.GC()
	}
	if ep.HasErrors() || !ep.Validate() {
		t.Error("the provider keeps a ruled control nothing else holds")
	}
}

func TestRuleMatchRegexp(t *testing.T) {
	ep := ErrorProvider()
	re := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[a-z]+$`)
	tb := Rule(ep, TextBox(), MatchRegexp(re, "not an email"))
	NewTester(t, tb, 400, 100)
	for _, c := range []struct {
		s  string
		ok bool
	}{{"ann@example.com", true}, {"ann@example", false}, {"ann example.com", false}, {"", false}} {
		tb.SetText(c.s)
		if ok := ep.Validate(); ok != c.ok || (ep.Error(tb) == "not an email") == c.ok {
			t.Errorf("%q: Validate %v, Error %q; want ok %v", c.s, ok, ep.Error(tb), c.ok)
		}
	}
}

func TestRuleValidateOnLeave(t *testing.T) {
	var panics []any
	OnPanic(func(v any, _ []byte) { panics = append(panics, v) })
	t.Cleanup(func() {
		OnPanic(nil)
		if len(panics) > 0 {
			t.Errorf("a handler panicked: %v", panics)
		}
	})
	ep := ErrorProvider()
	a, b, c := TextBox(), TextBox(), TextBox()
	runs := map[string]int{}
	count := func(name string) func(string) string {
		return func(v string) string { runs[name]++; return Required(name + " required")(v) }
	}
	Rule(ep, a, count("a"))
	Rule(ep, b, count("b"))
	tt := NewTester(t, VBox(4).Add(a, b, c), 400, 200)
	tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focus()
	tt.Find(ByRole(ui.RoleTextField).Nth(2)).Focus()
	tt.Settle()
	if len(runs) != 0 || ep.HasErrors() {
		t.Fatalf("off: leaving a field ran rules %v", runs)
	}
	ep.SetValidateOnLeave(true)
	tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focus()
	tt.Find(ByRole(ui.RoleTextField).Nth(2)).Focus()
	tt.Settle()
	if runs["a"] != 1 || runs["b"] != 0 || ep.Error(a) != "a required" || ep.Error(b) != "" {
		t.Errorf("on: runs %v, errors a %q b %q; want a's rules alone, once", runs, ep.Error(a), ep.Error(b))
	}
	ep.SetValidateOnLeave(false)
	tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	tt.Find(ByRole(ui.RoleTextField).Nth(2)).Focus()
	tt.Settle()
	if runs["b"] != 0 {
		t.Errorf("off again: leaving b ran its rules %d times", runs["b"])
	}
}

// The app's own OnFocusOut still runs when rules run on leave, before them.
func TestRuleValidateOnLeaveKeepsOnFocusOut(t *testing.T) {
	ep := ErrorProvider().SetValidateOnLeave(true)
	var order []string
	a := TextBox().OnFocusOut(func() { order = append(order, "focusout") })
	Rule(ep, a, func(string) string { order = append(order, "rule"); return "" })
	tt := NewTester(t, VBox(4).Add(a, TextBox()), 400, 200)
	tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focus()
	tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	tt.Settle()
	if !slices.Equal(order, []string{"focusout", "rule"}) {
		t.Errorf("ran %v, want OnFocusOut then the rule", order)
	}
}

func TestRuleValidateChildren(t *testing.T) {
	ep := ErrorProvider()
	name, age := TextBox(), TextBox()
	var ran []string
	name.OnValidating(func() bool { ran = append(ran, "handler name"); return true })
	age.OnValidating(func() bool { ran = append(ran, "handler age"); return true })
	Rule(ep, name, func(v string) string { ran = append(ran, "rule"); return Required("required")(v) })
	f := NewForm("v", 400, 300)
	f.SetContent(VBox(4).Add(name, age))
	tt := TestForm(t, f)
	tt.Settle()
	if f.ValidateChildren() {
		t.Error("ValidateChildren said true with a failing rule")
	}
	if want := []string{"handler name", "handler age", "rule"}; !slices.Equal(ran, want) {
		t.Errorf("ran %v, want %v: rules after the OnValidating funcs", ran, want)
	}
	if ep.Error(name) != "required" {
		t.Errorf("Error = %q", ep.Error(name))
	}
	tt.Frame()
	if got := tt.Announcements(); len(got) != 1 || got[0] != "required" {
		t.Errorf("announced %q, want the rule's message once", got)
	}
	name.SetText("Ann")
	if !f.ValidateChildren() || ep.HasErrors() {
		t.Error("ValidateChildren with a good name")
	}
}

// A provider's rules on a control outside the form, or hidden in it, do not
// run in the form's ValidateChildren; Validate runs them all.
func TestRuleValidateChildrenOnlyTheFormsControls(t *testing.T) {
	ep := ErrorProvider()
	in, hidden, out := TextBox(), TextBox(), TextBox()
	ran := map[string]int{}
	rule := func(name string) func(string) string {
		return func(string) string { ran[name]++; return name }
	}
	Rule(ep, in, rule("in"))
	Rule(ep, hidden, rule("hidden"))
	Rule(ep, out, rule("out"))
	hidden.SetVisible(false)
	f := NewForm("v", 400, 300)
	f.SetContent(VBox(4).Add(in, hidden))
	tt := TestForm(t, f)
	tt.Settle()
	if f.ValidateChildren() {
		t.Error("ValidateChildren said true")
	}
	if ran["in"] != 1 || ran["hidden"] != 0 || ran["out"] != 0 || ep.Error(out) != "" {
		t.Errorf("ran %v: only the form's shown controls", ran)
	}
	if ep.Validate() || ran["hidden"] != 1 || ran["out"] != 1 {
		t.Errorf("Validate ran %v: every rule", ran)
	}
}

func TestRuleNoFocusMove(t *testing.T) {
	ep := ErrorProvider()
	a, b := TextBox(), TextBox()
	Rule(ep, a, Required("a required"))
	Rule(ep, b, func(string) string { return "" }) // passes after a failed: Validate is still false
	tt := NewTester(t, VBox(4).Add(a, b), 400, 200)
	tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	tt.Settle()
	if ep.Validate() {
		t.Fatal("Validate passed")
	}
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() || tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focused() {
		t.Error("Validate moved the focus")
	}
}

// With no rule, a provider and a form behave as before: Validate passes and
// touches no mark set by hand, ValidateChildren runs the handlers alone, and
// leaving a field runs its OnFocusOut once.
func TestRuleOffIsUnchanged(t *testing.T) {
	ep := ErrorProvider()
	a, b := TextBox(), TextBox()
	var ran []string
	a.OnValidating(func() bool { ran = append(ran, "a"); return true })
	outs := 0
	a.OnFocusOut(func() { outs++ })
	f := NewForm("v", 400, 300)
	f.SetContent(VBox(4).Add(a, b))
	tt := TestForm(t, f)
	tt.Settle()
	ep.SetError(b, "by hand")
	if !ep.Validate() || ep.Error(b) != "by hand" {
		t.Errorf("Validate with no rule: Error(b) %q", ep.Error(b))
	}
	if !f.ValidateChildren() || !slices.Equal(ran, []string{"a"}) || ep.Error(b) != "by hand" {
		t.Errorf("ValidateChildren ran %v, Error(b) %q", ran, ep.Error(b))
	}
	ep.SetValidateOnLeave(true)
	tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focus()
	tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	tt.Settle()
	if outs != 1 || ep.Error(a) != "" {
		t.Errorf("OnFocusOut ran %d times, Error(a) %q", outs, ep.Error(a))
	}
}

// A frame at rest of a ruled TextBox, validating on leave and marked,
// allocates no more than a plain TextBox marked by hand.
func TestRuleAllocNothingAtRest(t *testing.T) {
	plainEP, ep := ErrorProvider(), ErrorProvider().SetValidateOnLeave(true)
	plainTB, tb := TextBox(), Rule(ep, TextBox(), Required("required"))
	plainEP.SetError(plainTB, "required")
	ep.Validate()
	plain := ui.NewTester(content(VBox(4).Add(plainTB, TextBox())), 300, 100)
	with := ui.NewTester(content(VBox(4).Add(tb, TextBox())), 300, 100)
	for range 10 {
		plain.Frame()
		with.Frame()
	}
	want, got := allocrun.Least(30, plain.Frame), allocrun.Least(30, with.Frame)
	if got > want {
		t.Fatalf("a frame of a ruled TextBox allocates %v, a plain one's %v", got, want)
	}
}

func TestRuleFromGoroutine(t *testing.T) {
	ep := ErrorProvider()
	tb := TextBox()
	tt := NewTester(t, VBox(8).Add(Label("Name"), tb, TextBox()), 400, 200)
	box := tt.Find(ByRole(ui.RoleTextField).Nth(0)).Bounds()
	before := redPixels(tt, box)
	var wg sync.WaitGroup
	wg.Add(1)
	var last bool
	stop := make(chan struct{})
	go func() {
		defer wg.Done()
		Rule(ep, tb, Required("required"))
		for i := 0; ; i++ {
			select {
			case <-stop:
				tb.SetText("")
				last = ep.Validate()
				return
			default:
			}
			tb.SetText(strings.Repeat("x", i%2))
			ep.Validate()
			for j := range 50 {
				ep.SetValidateOnLeave(j%2 == 0)
			}
		}
	}()
	for i := range 50 {
		tt.Find(ByRole(ui.RoleTextField).Nth(i % 2)).Focus() // the focus leaves the other: its rules may run
	}
	close(stop)
	wg.Wait()
	tt.Settle()
	if last {
		t.Fatal("Validate passed on another goroutine")
	}
	tt.Frame()
	if redPixels(tt, box) <= before || ep.Error(tb) != "required" {
		t.Error("the mark set from another goroutine did not show on the next frame")
	}
}
