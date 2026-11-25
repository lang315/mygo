package widget

import (
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// proxyOptions are the options of the cases' combo.
var proxyOptions = []Option{{"none", "None"}, {"http", "HTTP"}, {"socks", "SOCKS5"}}

// shownCombo is the cases' setup: a shows for http, b for http and socks.
func shownCombo() (c *ComboBoxWidget, a, b *LabelWidget) {
	a, b = Label("A"), Label("B")
	c = ComboBox(proxyOptions...).Shows("http", a, b).Shows("socks", b)
	return c, a, b
}

func wantVisible(t *testing.T, step string, a, b interface{ Visible() bool }, wa, wb bool) {
	t.Helper()
	if a.Visible() != wa || b.Visible() != wb {
		t.Fatalf("%s: a visible %v, b visible %v; want %v, %v", step, a.Visible(), b.Visible(), wa, wb)
	}
}

// shownFrame is what a frame painted: the text combo showed, and whether
// the pass painted built a and b.
type shownFrame struct {
	text string
	a, b bool
}

// frameRecorder is a Tester of form whose frames record, as they paint,
// the text of combo and which of a and b the pass painted built.
func frameRecorder(form Control, combo *ComboBoxWidget, a, b Control) (*ui.Tester, *[]shownFrame) {
	view := content(form)
	frames := new([]shownFrame)
	tt := ui.NewTester(func(c *ui.Context) {
		a.base().el, b.base().el = ui.Element{}, ui.Element{} // render sets it for each control it builds
		view(c)
		f := shownFrame{combo.Text(), a.base().el != (ui.Element{}), b.base().el != (ui.Element{})}
		ui.Box(c).Width(1).Height(1).DrawOver(func(*ui.Painter, ui.Rect) { *frames = append(*frames, f) })
	}, 800, 600)
	tt.Frame()
	return tt, frames
}

// wantFirstFrame checks the first frame painted since from that shows the
// combo's text as text: it built a and b as want says.
func wantFirstFrame(t *testing.T, step string, frames []shownFrame, from int, text string, wa, wb bool) {
	t.Helper()
	for _, f := range frames[min(from, len(frames)):] {
		if f.text != text {
			continue
		}
		if f.a != wa || f.b != wb {
			t.Fatalf("%s: the first frame showing %q built a %v, b %v; want %v, %v", step, text, f.a, f.b, wa, wb)
		}
		return
	}
	t.Fatalf("%s: no frame painted %q: %+v", step, text, frames[min(from, len(frames)):])
}

// postedTo waits until the handlers posted so far have run, then returns what
// ch holds.
func postedTo[T any](ch chan T) []T {
	idle()
	var got []T
	for {
		select {
		case v := <-ch:
			got = append(got, v)
		default:
			return got
		}
	}
}

// 1. Nothing chosen: both hidden at once.
func TestComboShowsNothingChosen(t *testing.T) {
	_, a, b := shownCombo()
	wantVisible(t, "nothing chosen", a, b, false, false)
}

// 2. Setters choose what shows; a control named for several values shows
// while any is chosen.
func TestComboShowsSetValueAndUnselect(t *testing.T) {
	c, a, b := shownCombo()
	c.SetValue("http")
	wantVisible(t, "http", a, b, true, true)
	c.SetValue("socks")
	wantVisible(t, "socks", a, b, false, true)
	c.Unselect()
	wantVisible(t, "Unselect", a, b, false, false)
	c.SetText("SOCKS5")
	wantVisible(t, "SetText", a, b, false, true)
}

// 3. The user's pick in a form where a and b come before the combo: the
// first frame after it shows them; OnChange runs once a pick.
func TestComboShowsUserPickFirstFrame(t *testing.T) {
	changed := make(chan string, 8)
	c, a, b := shownCombo()
	c.SetAccessibleName("Proxy").OnChange(func() { changed <- c.Value() })
	tt, frames := frameRecorder(VBox(0).Add(a, b, c), c, a, b)
	click(t, tt, "Proxy")
	n := len(*frames)
	if err := tt.Click("HTTP"); err != nil {
		t.Fatal(err)
	}
	wantFirstFrame(t, "pick HTTP", *frames, n, "HTTP", true, true)
	if !tt.HasText("A") || !tt.HasText("B") {
		t.Fatalf("after picking HTTP: %q", tt.Texts())
	}
	if got := postedTo(changed); len(got) != 1 || got[0] != "http" {
		t.Fatalf("OnChange ran with %q after picking HTTP, want [http]", got)
	}
	click(t, tt, "Proxy")
	n = len(*frames)
	if err := tt.Click("None"); err != nil {
		t.Fatal(err)
	}
	wantFirstFrame(t, "pick None", *frames, n, "None", false, false)
	if tt.HasText("A") || tt.HasText("B") {
		t.Fatalf("after picking None: %q", tt.Texts())
	}
	if got := postedTo(changed); len(got) != 1 || got[0] != "none" {
		t.Fatalf("OnChange ran with %q after picking None, want [none]", got)
	}
}

// 4. Shows applies at once to the choice made before it.
func TestComboShowsAppliesToTheCurrentChoice(t *testing.T) {
	a, b := Label("A"), Label("B")
	ComboBox(proxyOptions...).SetValue("http").Shows("http", a).Shows("socks", b)
	wantVisible(t, "Shows after SetValue(http)", a, b, true, false)
}

// 5. An Editable combo: the option of the text typed shows its controls,
// a text of no option none.
func TestComboShowsEditable(t *testing.T) {
	a := Label("A")
	c := ComboBox(proxyOptions...).Editable().Shows("http", a)
	c.SetText("HTTP")
	if !a.Visible() {
		t.Fatal("SetText(HTTP) left a hidden")
	}
	tt, frames := frameRecorder(VBox(0).Add(a, Label("B"), c), c, a, a)
	focus(tt, 1)
	tt.Key(0, ui.KeyEnd)
	n := len(*frames)
	tt.Key(0, ui.KeyBackspace) // HTT: no option's text
	if c.Text() != "HTT" {
		t.Fatalf("typed: text %q", c.Text())
	}
	wantFirstFrame(t, "typed HTT", *frames, n, "HTT", false, false)
	if a.Visible() {
		t.Fatal("HTT typed left a visible")
	}
}

// 6. SetVisible lasts until the choice next changes.
func TestComboShowsSetVisibleUntilTheNextChange(t *testing.T) {
	c, a, _ := shownCombo()
	c.SetValue("socks")
	a.SetVisible(true)
	if !a.Visible() {
		t.Fatal("SetVisible(true) under socks")
	}
	c.SetValue("none")
	if a.Visible() {
		t.Fatal("SetValue(none) left a visible")
	}
}

// 7. A value no option has: no panic, hidden.
func TestComboShowsUnknownValue(t *testing.T) {
	a := Label("A")
	c := ComboBox(proxyOptions...).Shows("nosuch", a)
	c.SetValue("http")
	if a.Visible() {
		t.Fatal("a shows under http")
	}
	c.SetValue("nosuch")
	if a.Visible() {
		t.Fatal("a shows for nosuch, which no option is")
	}
	b := Label("B")
	c.Shows("", b) // "" is no option's value: nothing chosen is not it
	if b.Visible() {
		t.Fatal("b shows for \"\" with nothing chosen")
	}
}

// 8. Hiding takes the keyboard focus, and OnFocusOut runs once.
func TestComboShowsHidingTakesTheFocus(t *testing.T) {
	out := make(chan struct{}, 4)
	b := TextBox().OnFocusOut(func() { out <- struct{}{} })
	c := ComboBox(proxyOptions...).Shows("socks", b).SetValue("socks")
	tt := tester(t, VBox(0).Add(b, c))
	b.Focus()
	tt.Frame()
	c.SetValue("none")
	tt.Frame()
	tt.Frame()
	if b.Visible() {
		t.Fatal("b visible under none")
	}
	if got := postedTo(out); len(got) != 1 {
		t.Fatalf("OnFocusOut ran %d times, want 1", len(got))
	}
}

// 9. Removing the chosen option hides its controls.
func TestComboShowsRemoveChosenOption(t *testing.T) {
	c, a, _ := shownCombo()
	c.SetValue("http")
	if !a.Visible() {
		t.Fatal("a hidden under http")
	}
	c.RemoveOption("http")
	if a.Visible() {
		t.Fatal("RemoveOption(http) left a visible")
	}
}

// 10. Shows, SetValue and picks from several goroutines while the form
// draws (go test -race).
func TestComboShowsRace(t *testing.T) {
	c, a, b := shownCombo()
	c.SetAccessibleName("Proxy")
	tt := tester(t, VBox(0).Add(a, b, c))
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; ; j++ {
				select {
				case <-stop:
					return
				default:
				}
				switch (i + j) % 3 {
				case 0:
					c.SetValue("http")
				case 1:
					c.Shows("socks", a)
				default:
					c.SetValue(proxyOptions[j%3].Value)
				}
				_ = a.Visible()
			}
		}()
	}
	picks := 0
	for i := range 20 {
		if tt.Click("Proxy") != nil {
			continue
		}
		if tt.Click(proxyOptions[i%3].Text) == nil {
			picks++
		}
		tt.Frame()
	}
	close(stop)
	wg.Wait()
	if picks == 0 {
		t.Fatal("no pick reached the popup")
	}
	c.SetValue("none")
	wantVisible(t, "none at the end", a, b, false, false)
}

// A setter that leaves the choice as it is does not apply the shows again,
// so a SetVisible of the program lasts: the same value, an option added,
// a binding reloaded with the same value.
func TestComboShowsSameChoiceKeepsSetVisible(t *testing.T) {
	c, a, _ := shownCombo()
	c.SetValue("socks")
	Bind(c, func() string { return "socks" }, func(string) {})
	if a.Visible() {
		t.Fatal("a visible under socks")
	}
	a.SetVisible(true)
	c.SetValue("socks")
	c.AddOption(Option{"ftp", "FTP"})
	Reload(c)
	c.SetText("SOCKS5")
	if !a.Visible() {
		t.Fatal("a setter keeping socks applied the shows again")
	}
}

// No controls, and a nil one: no panic.
func TestComboShowsNoControlsAndNil(t *testing.T) {
	a := Label("A")
	c := ComboBox(proxyOptions...).Shows("http").Shows("http", nil, a)
	if a.Visible() {
		t.Fatal("a visible with nothing chosen")
	}
	c.SetValue("http")
	if !a.Visible() {
		t.Fatal("a hidden under http")
	}
}

// A binding applying, at Bind and at Reload, applies the shows.
func TestComboShowsBinding(t *testing.T) {
	src := "http"
	c, a, b := shownCombo()
	Bind(c, func() string { return src }, func(v string) { src = v })
	wantVisible(t, "bound to http", a, b, true, true)
	src = "socks"
	Reload(c)
	wantVisible(t, "reloaded socks", a, b, false, true)
}

// The user's pick applies before OnChange runs: OnChange sees the controls
// shown, and a SetVisible of its own lasts.
func TestComboShowsBeforeOnChange(t *testing.T) {
	seen := make(chan bool, 4)
	c, a, b := shownCombo()
	c.SetAccessibleName("Proxy").OnChange(func() {
		seen <- a.Visible()
		b.SetVisible(false)
	})
	tt := tester(t, VBox(0).Add(a, b, c))
	wantVisible(t, "nothing chosen", a, b, false, false)
	click(t, tt, "Proxy")
	click(t, tt, "HTTP")
	if got := postedTo(seen); len(got) != 1 || !got[0] {
		t.Fatalf("OnChange saw a visible %v, want [true]", got)
	}
	tt.Frame()
	if b.Visible() || !a.Visible() {
		t.Fatalf("after OnChange: a %v, b %v; want true, false", a.Visible(), b.Visible())
	}
}

// A combo with a submenu takes the pick while it builds: the first frame
// after it still shows the controls placed before it.
func TestComboShowsSubmenuPickFirstFrame(t *testing.T) {
	a, b := Label("A"), Label("B")
	c := ComboBox(proxyOptions...).AddSubmenu("More", nil, Option{"ftp", "FTP"}).
		SetAccessibleName("Proxy").Shows("http", a, b).Shows("ftp", b)
	tt, frames := frameRecorder(VBox(0).Add(a, b, c), c, a, b)
	wantFirstFrame(t, "nothing chosen", *frames, 0, "", false, false)
	click(t, tt, "Proxy")
	n := len(*frames)
	if err := tt.Click("HTTP"); err != nil {
		t.Fatal(err)
	}
	wantFirstFrame(t, "pick HTTP", *frames, n, "HTTP", true, true)
	if c.Value() != "http" {
		t.Fatalf("value %q", c.Value())
	}
}

// An Editable combo's choice from its popup, taken while it builds: the
// first frame after it shows the controls placed before it.
func TestComboShowsEditablePopupChoiceFirstFrame(t *testing.T) {
	a, b := Label("A"), Label("B")
	c := ComboBox(proxyOptions...).Editable().Shows("http", a, b)
	tt, frames := frameRecorder(VBox(0).Add(a, b, c), c, a, b)
	wantFirstFrame(t, "nothing chosen", *frames, 0, "", false, false)
	focus(tt, 1)
	tt.Key(0, ui.KeyDown) // opens the popup
	tt.Frame()
	n := len(*frames)
	if err := tt.Click("HTTP"); err != nil {
		t.Fatal(err)
	}
	if c.Value() != "http" {
		t.Fatalf("value %q, texts %q", c.Value(), tt.Texts())
	}
	wantFirstFrame(t, "chose HTTP", *frames, n, "HTTP", true, true)
}

// Editable can change the choice: of two options of one text, the text
// chooses the first.
func TestComboShowsEditableChangesTheChoice(t *testing.T) {
	a, b := Label("A"), Label("B")
	c := ComboBox(Option{"a", "Same"}, Option{"b", "Same"}).SetValue("b").Shows("a", a).Shows("b", b)
	wantVisible(t, "b chosen", a, b, false, true)
	c.Editable()
	if c.Value() != "a" {
		t.Fatalf("Editable: value %q, want a", c.Value())
	}
	wantVisible(t, "Editable", a, b, true, false)
}

// An Editable combo's options changing under its text: the text no
// longer an option's hides, and SetOptionText making it one shows.
func TestComboShowsEditableOptionsChange(t *testing.T) {
	a := Label("A")
	c := ComboBox(proxyOptions...).Editable().Shows("http", a).SetText("HTTP")
	c.SetOptionText("http", "Web")
	if a.Visible() {
		t.Fatal("HTTP no longer an option's text, a visible")
	}
	c.SetText("Web")
	if !a.Visible() {
		t.Fatal("Web chosen, a hidden")
	}
	c.RemoveOption("http")
	if a.Visible() {
		t.Fatal("http removed, a visible")
	}
}
