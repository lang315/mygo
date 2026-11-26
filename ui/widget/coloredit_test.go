package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestColorEditPick(t *testing.T) {
	changes := make(chan ui.Color, 4)
	var e *ColorEditWidget
	e = ColorEdit().OnChange(func() { changes <- e.Color() })
	blue := ui.Hex("#3b82f6")
	e.SetColor(blue)
	if e.Color() != blue {
		t.Fatalf("Color %v after SetColor(%v)", e.Color(), blue)
	}
	tt := NewTester(t, e, 400, 500)
	tt.Find(ByRole(ui.RoleColorWell)).Click()
	tt.Find(ByLabel("Green")).Click()
	green := ui.Hex("#22c55e")
	if got := received(t, changes); got != green || e.Color() != green {
		t.Fatalf("a pick of Green: OnChange saw %v, Color %v", got, e.Color())
	}
	if len(changes) != 0 {
		t.Fatalf("one pick ran OnChange %d more times", len(changes))
	}
}

func TestColorEditSetterDoesNotFireOnChange(t *testing.T) {
	changes := make(chan struct{}, 4)
	e := ColorEdit().OnChange(func() { changes <- struct{}{} })
	tt := NewTester(t, e, 400, 500)
	red := ui.Hex("#ef4444")
	e.SetColor(red)
	tt.Settle()
	if v := tt.Find(ByRole(ui.RoleColorWell)).Value(); v != "#ef4444" {
		t.Fatalf("the well shows %q, want #ef4444", v)
	}
	if len(changes) != 0 {
		t.Fatal("SetColor ran OnChange")
	}
}
