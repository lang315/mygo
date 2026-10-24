package widget

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
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
	if !received(t, ran) {
		t.Fatal("OnClick ran off the event goroutine")
	}
}

func TestDisabledButtonDoesNotClick(t *testing.T) {
	ran := make(chan struct{}, 1)
	tt := NewTester(t, Button("No").SetDisabled(true).OnClick(func() { ran <- struct{}{} }), 0, 0)
	if !tt.Find(ByLabel("No")).Disabled() {
		t.Fatal("the button is not disabled")
	}
	if msg := failure(func(tb *fakeTB) { tt.tb = tb; tt.Find(ByLabel("No")).Click() }); !strings.Contains(msg, "disabled") {
		t.Fatalf("clicking it: %q", msg)
	}
	select {
	case <-ran:
		t.Fatal("a disabled button ran OnClick")
	default:
	}
}

func TestButtonVariantsBuild(t *testing.T) {
	tt := tester(t, HBox(4).Add(Button("A").Suggested(), Button("B").Destructive(), Button("C").SetText("C2"), LinkLabel("Docs", "https://example.com")))
	for _, s := range []string{"A", "B", "C2", "Docs"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestCheckboxClickAndSet(t *testing.T) {
	changed := make(chan bool, 4)
	var cb *CheckBoxWidget
	CheckBox("Remember").AssignTo(&cb).OnChange(func() { changed <- cb.Checked() })
	tt := NewTester(t, cb, 0, 0)
	if !tt.Find(ByLabel("Remember")).Click().Checked() || !received(t, changed) {
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
	g := CheckBoxGroup(false, Option{"a", "Apple"}, Option{"b", "Banana"}, Option{"c", "Cherry"})
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
	tt := NewTester(t, g, 0, 0)
	tt.Find(ByLabel("Large")).Click()
	if v := received(t, changed); v != "l" {
		t.Fatalf("OnChange saw %q", v)
	}
}

func lightTester(t *testing.T, w Control) *ui.Tester {
	t.Helper()
	tt := tester(t, w)
	tt.SetDark(false)
	tt.Frame()
	return tt
}

func TestSuggestedButtonLooksAsTwSuggestedAction(t *testing.T) {
	th := ui.LightTheme()
	tt := lightTester(t, Button("Save").Suggested())
	if c := pixelLeft(t, tt, "Save"); !near(c, th.Accent) {
		t.Fatalf("suggested %v, want the accent %v", c, th.Accent)
	}
	x, y := centerOf(t, tt, "Save")
	tt.Move(x, y)
	if c := pixelLeft(t, tt, "Save"); !near(c, th.AccentHover) {
		t.Fatalf("hovered %v, want %v", c, th.AccentHover)
	}
	tt.Press(x, y)
	if c := pixelLeft(t, tt, "Save"); !near(c, th.AccentPressed) {
		t.Fatalf("pressed %v, want %v", c, th.AccentPressed)
	}
}

func TestDestructiveButtonLooksAsTwDestructiveAction(t *testing.T) {
	th := ui.LightTheme()
	tt := lightTester(t, Button("Delete").Destructive())
	if c := pixelLeft(t, tt, "Delete"); !near(c, th.Danger) {
		t.Fatalf("destructive %v, want danger %v", c, th.Danger)
	}
	x, y := centerOf(t, tt, "Delete")
	pg := page(tt)
	tt.Move(x, y)
	if c := pixelLeft(t, tt, "Delete"); !near(c, mix(th.Danger, pg, 0.9)) {
		t.Fatalf("hovered %v, want danger at 0.9 over the page", c)
	}
	tt.Press(x, y)
	if c := pixelLeft(t, tt, "Delete"); !near(c, mix(th.Danger, pg, 0.8)) {
		t.Fatalf("pressed %v, want danger at 0.8 over the page", c)
	}
}

func TestSuggestedButtonIsTheWidgetsOwnStyle(t *testing.T) {
	th := ui.LightTheme()
	// An applied style loses to the control's own, whatever the order.
	b := Button("Save").Apply(NewStyle().Background(Fixed(red))).Suggested()
	if c := pixelLeft(t, lightTester(t, b), "Save"); !near(c, th.Accent) {
		t.Fatalf("applied red over Suggested: %v, want the accent", c)
	}
	// An applied style still sets what Suggested does not.
	b = Button("Save").Suggested().Apply(NewStyle().Radius(0).Padding(9))
	tt := lightTester(t, b)
	if c := pixelLeft(t, tt, "Save"); !near(c, th.Accent) {
		t.Fatalf("applied padding: %v, want the accent", c)
	}
}

func TestSuggestedKeepsTheOwnHoverTheButtonHad(t *testing.T) {
	th := ui.LightTheme()
	for name, order := range map[string]func(b *ButtonWidget) *ButtonWidget{
		"before": func(b *ButtonWidget) *ButtonWidget {
			return b.own(func(s Style) Style { return s.Hover(NewStyle().Opacity(0.5)) }).Suggested()
		},
	} {
		t.Run(name, func(t *testing.T) {
			tt := lightTester(t, order(Button("Save")))
			x, y := centerOf(t, tt, "Save")
			pg := page(tt)
			tt.Move(x, y)
			// The hover's opacity and Suggested's hover color both show.
			if c := pixelLeft(t, tt, "Save"); !near(c, mix(th.AccentHover, pg, 0.5)) {
				t.Fatalf("hovered %v, want the accent hover at 0.5 over the page", c)
			}
		})
	}
}
