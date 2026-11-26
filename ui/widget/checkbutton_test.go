package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCheckButtonToggles(t *testing.T) {
	changed := make(chan bool, 4)
	var b *CheckButtonWidget
	b = CheckButton("Bold").OnChange(func() { changed <- b.Checked() })
	tt := NewTester(t, b, 400, 300)
	tt.Find(ByLabel("Bold")).Click()
	if got := received(t, changed); !got || !b.Checked() {
		t.Fatalf("a click gave Checked %v", got)
	}
	tt.Find(ByLabel("Bold")).Click()
	if got := received(t, changed); got || b.Checked() {
		t.Fatalf("a second click gave Checked %v", got)
	}
	select {
	case <-changed:
		t.Fatal("OnChange ran more than once a click")
	default:
	}
}

func TestCheckButtonSpaceToggles(t *testing.T) {
	changed := make(chan bool, 4)
	var b *CheckButtonWidget
	b = CheckButton("Bold").OnChange(func() { changed <- b.Checked() })
	tt := NewTester(t, b, 400, 300)
	tt.Find(ByLabel("Bold")).Key(0, ui.KeySpace)
	if got := received(t, changed); !got {
		t.Fatal("Space did not check it")
	}
	if !tt.Find(ByLabel("Bold")).Checked() {
		t.Fatal("the button does not show pressed")
	}
}

func TestCheckButtonSetterDoesNotFireOnChange(t *testing.T) {
	changed := make(chan struct{}, 4)
	b := CheckButton("Bold").OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, b, 400, 300)
	b.SetChecked(true)
	tt.Settle()
	if !b.Checked() || !tt.Find(ByLabel("Bold")).Checked() {
		t.Fatal("SetChecked(true) does not show")
	}
	select {
	case <-changed:
		t.Fatal("SetChecked ran OnChange")
	default:
	}
}

func TestCheckButtonDisabled(t *testing.T) {
	changed := make(chan struct{}, 4)
	b := CheckButton("Bold").OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, b.SetDisabled(true), 400, 300)
	tt.ClickAt(middle(tt.Find(ByLabel("Bold")).Bounds()))
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	tt.Settle()
	if b.Checked() {
		t.Fatal("a disabled CheckButton was checked")
	}
	select {
	case <-changed:
		t.Fatal("a disabled CheckButton ran OnChange")
	default:
	}
}

func TestCheckButtonIconKeepsItsName(t *testing.T) {
	b := CheckButton("Bold").SetIcon(chevronDown)
	tt := NewTester(t, b, 400, 300)
	tt.Find(ByLabel("Bold")).Click()
	if !b.Checked() {
		t.Fatal("a click on the icon button did not check it")
	}
}

func TestCheckButtonBindsABool(t *testing.T) {
	v := true
	b := Bind(CheckButton("Bold"), func() bool { return v }, func(x bool) { v = x })
	tt := NewTester(t, b, 400, 300)
	if !tt.Find(ByLabel("Bold")).Checked() {
		t.Fatal("the button does not show the source's true")
	}
	tt.Find(ByLabel("Bold")).Click()
	if v {
		t.Fatal("the source was not set to false")
	}
}
