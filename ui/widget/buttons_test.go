package widget

import (
	"slices"
	"testing"
)

func click(t *testing.T, tt interface {
	Click(string) error
	Frame()
}, s string) {
	t.Helper()
	if err := tt.Click(s); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
}

func TestButtonClickRunsOnClickOnTheEventGoroutine(t *testing.T) {
	ran := make(chan bool, 1)
	tt := tester(t, Button("Go").OnClick(func() { ran <- onEvents() }))
	click(t, tt, "Go")
	if !wait(t, ran) {
		t.Fatal("OnClick ran off the event goroutine")
	}
}

func TestDisabledButtonDoesNotClick(t *testing.T) {
	ran := make(chan struct{}, 1)
	tt := tester(t, Button("No").SetDisabled(true).OnClick(func() { ran <- struct{}{} }))
	click(t, tt, "No")
	Update(func() {})
	select {
	case <-ran:
		t.Fatal("a disabled button ran OnClick")
	default:
	}
}

func TestButtonVariantsBuild(t *testing.T) {
	tt := tester(t, HBox(4).Add(Button("A").Suggested(), Button("B").Destructive(), Button("C").SetText("C2"), Link("Docs", "https://example.com")))
	for _, s := range []string{"A", "B", "C2", "Docs"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestCheckboxClickAndSet(t *testing.T) {
	changed := make(chan bool, 4)
	var cb *CheckboxWidget
	Checkbox("Remember").AssignTo(&cb).OnChange(func() { changed <- cb.Checked() })
	tt := tester(t, cb)
	click(t, tt, "Remember")
	if !wait(t, changed) || !cb.Checked() {
		t.Fatal("click did not check")
	}
	cb.SetChecked(false)
	tt.Frame()
	if cb.Checked() {
		t.Fatal("SetChecked(false) did not stick")
	}
}

func TestToggleAndSwitch(t *testing.T) {
	tg, sw := Toggle("Bold"), Switch()
	tt := tester(t, VBox(0).Add(tg, sw))
	click(t, tt, "Bold")
	if !tg.Active() {
		t.Fatal("toggle not active after click")
	}
	sw.SetOn(true)
	tt.Frame()
	if !sw.On() {
		t.Fatal("switch not on")
	}
}

func TestCheckboxGroupValuesInOptionOrder(t *testing.T) {
	g := CheckboxGroup(false, Option{"a", "Apple"}, Option{"b", "Banana"}, Option{"c", "Cherry"})
	tt := tester(t, g)
	click(t, tt, "Cherry")
	click(t, tt, "Apple")
	if got := g.Values(); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("Values %v, want [a c]", got)
	}
	g.SetValues("b")
	tt.Frame()
	if got := g.Values(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("after SetValues: %v", got)
	}
}

func TestRadioGroup(t *testing.T) {
	changed := make(chan string, 2)
	var g *RadioGroupWidget
	RadioGroup(true, Option{"s", "Small"}, Option{"l", "Large"}).AssignTo(&g).OnChange(func() { changed <- g.Value() })
	if g.Value() != "s" {
		t.Fatalf("first value %q, want s", g.Value())
	}
	tt := tester(t, g)
	click(t, tt, "Large")
	if v := wait(t, changed); v != "l" {
		t.Fatalf("OnChange saw %q", v)
	}
}
