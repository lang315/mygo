package widget

import (
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// popupTester shows a PopupEdit holding a check box labelled "Inside", at the
// top of a window with room below it.
func popupTester(t *testing.T, p *PopupEditWidget) (*Tester, *Node) {
	t.Helper()
	tt := NewTester(t, VBox(8).Add(p, Label("Below")), 400, 300)
	return tt, tt.Find(ByRole(ui.RolePopUpButton))
}

func newPopupEdit() *PopupEditWidget { return PopupEdit(CheckBox("Inside")) }

// opened clicks the field and fails unless the popup opened.
func opened(t *testing.T, p *PopupEditWidget, f *Node) {
	t.Helper()
	f.Click()
	if !p.IsOpen() {
		t.Fatal("a click did not open the popup")
	}
}

// firedOnce fails unless ch received exactly once once the queue settled:
// Settle runs what was posted.
func firedOnce[T any](t *testing.T, tt *Tester, ch <-chan T, what string) {
	t.Helper()
	tt.Settle()
	if n := len(ch); n != 1 {
		t.Fatalf("%s ran %d times, want once", what, n)
	}
}

// noneWithin fails when ch receives anything after the queue settled.
func noneWithin[T any](t *testing.T, tt *Tester, ch <-chan T, what string) {
	t.Helper()
	tt.Settle()
	select {
	case <-ch:
		t.Fatalf("%s ran", what)
	default:
	}
}

func TestPopupEditOpensOnClick(t *testing.T) {
	p := newPopupEdit()
	tt, f := popupTester(t, p)
	if p.IsOpen() || tt.Has(ByLabel("Inside")) {
		t.Fatal("the popup shows before a click")
	}
	f.Click()
	if !p.IsOpen() {
		t.Fatal("a click did not open the popup: IsOpen is false")
	}
	if !tt.Has(ByLabel("Inside")) {
		t.Fatalf("the controls inside are not built: %q", tt.Texts())
	}
}

func TestPopupEditAltDownOpens(t *testing.T) {
	p := newPopupEdit()
	tt, f := popupTester(t, p)
	f.Key(ui.Alt, ui.KeyDown)
	if !p.IsOpen() || !tt.Has(ByLabel("Inside")) {
		t.Fatal("Alt+Down did not open the popup")
	}
}

// Enter and Space open it too, as a click does (the spec).
func TestPopupEditEnterAndSpaceOpen(t *testing.T) {
	for _, k := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		p := newPopupEdit()
		_, f := popupTester(t, p)
		f.Key(0, k)
		if !p.IsOpen() {
			t.Fatalf("key %v did not open the popup", k)
		}
	}
}

func TestPopupEditEscapeClosesAndFiresOnClosed(t *testing.T) {
	closed := make(chan struct{}, 4)
	p := newPopupEdit().OnClosed(func() { closed <- struct{}{} })
	tt, f := popupTester(t, p)
	opened(t, p, f)
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if p.IsOpen() || tt.Has(ByLabel("Inside")) {
		t.Fatal("Escape did not close the popup")
	}
	firedOnce(t, tt, closed, "OnClosed")
}

func TestPopupEditClickOutsideCloses(t *testing.T) {
	closed := make(chan struct{}, 4)
	p := newPopupEdit().OnClosed(func() { closed <- struct{}{} })
	tt, f := popupTester(t, p)
	opened(t, p, f)
	tt.ClickAt(390, 290) // the page, below and beside everything
	tt.Settle()
	if p.IsOpen() || tt.Has(ByLabel("Inside")) {
		t.Fatal("a click outside did not close the popup")
	}
	firedOnce(t, tt, closed, "OnClosed")
}

// A click on the field of an open popup closes it, as a ComboBox's does: a
// press on the anchor is not one outside the popup.
func TestPopupEditClickOnFieldCloses(t *testing.T) {
	closed := make(chan struct{}, 4)
	p := newPopupEdit().OnClosed(func() { closed <- struct{}{} })
	tt, f := popupTester(t, p)
	opened(t, p, f)
	f.Click()
	if p.IsOpen() {
		t.Fatal("a second click on the field left the popup open")
	}
	firedOnce(t, tt, closed, "OnClosed")
}

func TestPopupEditSetText(t *testing.T) {
	p := newPopupEdit().SetText("3 selected")
	tt, f := popupTester(t, p)
	if v := f.Value(); v != "3 selected" {
		t.Fatalf("the field's value is %q", v)
	}
	if count(tt.Tester, "3 selected") != 1 {
		t.Fatalf("the field does not show the text: %q", tt.Texts())
	}
	p.SetText("none")
	tt.Settle()
	if v := tt.Find(ByRole(ui.RolePopUpButton)).Value(); v != "none" || p.Text() != "none" {
		t.Fatalf("after SetText the field shows %q, Text is %q", v, p.Text())
	}
}

// ShowPopup and ClosePopup are setters: they open and close it, and do not
// run OnClosed, as TrackBar's SetValue does not run OnChange.
func TestPopupEditShowClosePopupDoNotFireOnClosed(t *testing.T) {
	closed := make(chan struct{}, 4)
	p := newPopupEdit().OnClosed(func() { closed <- struct{}{} })
	tt, _ := popupTester(t, p)
	p.ShowPopup()
	tt.Settle()
	if !p.IsOpen() || !tt.Has(ByLabel("Inside")) {
		t.Fatal("ShowPopup did not open the popup")
	}
	p.ClosePopup()
	tt.Settle()
	if p.IsOpen() || tt.Has(ByLabel("Inside")) {
		t.Fatal("ClosePopup did not close the popup")
	}
	noneWithin(t, tt, closed, "OnClosed after ClosePopup")
}

func TestPopupEditDisabled(t *testing.T) {
	p := newPopupEdit().SetDisabled(true)
	tt, f := popupTester(t, p)
	if !f.Disabled() {
		t.Fatal("the field is not disabled")
	}
	b := f.Bounds()
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Settle()
	// A disabled field takes no focus: the keys go to the window.
	for _, k := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{0, ui.KeyEnter}, {0, ui.KeySpace}, {ui.Alt, ui.KeyDown}} {
		tt.Key(k.mods, k.key)
	}
	tt.Settle()
	if p.IsOpen() || tt.Has(ByLabel("Inside")) {
		t.Fatal("a disabled PopupEdit opened")
	}
}

// Hiding the control closes its popup, as SetVisible promises.
func TestPopupEditHidingCloses(t *testing.T) {
	p := newPopupEdit()
	tt, f := popupTester(t, p)
	f.Click()
	p.SetVisible(false)
	tt.Settle()
	p.SetVisible(true)
	tt.Settle()
	if p.IsOpen() || tt.Has(ByLabel("Inside")) {
		t.Fatal("the popup stayed open through a hide")
	}
}

func TestPopupEditShowClosePopupFromGoroutine(t *testing.T) {
	p := newPopupEdit()
	tt, _ := popupTester(t, p)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			if i%2 == 0 {
				p.ShowPopup()
			} else {
				p.ClosePopup()
			}
			p.SetText("text " + string(rune('a'+i%26)))
			_ = p.IsOpen()
			_ = p.Text()
		}
		p.ShowPopup()
		p.SetText("last")
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
	if !p.IsOpen() || !tt.Has(ByLabel("Inside")) {
		t.Fatal("the last ShowPopup does not show")
	}
	if v := tt.Find(ByRole(ui.RolePopUpButton)).Value(); v != "last" {
		t.Fatalf("the field shows %q, want last", v)
	}
}

// A closed PopupEdit at rest allocates nothing in a frame.
func TestPopupEditAllocNothingAtRest(t *testing.T) {
	calmOff()
	pop := ui.NewTester(content(newPopupEdit().SetText("One")), 400, 300)
	for range 10 {
		pop.Frame()
	}
	if n := allocrun.Least(30, pop.Frame); n != 0 {
		t.Fatalf("a frame of a closed PopupEdit allocates %v", n)
	}
}
