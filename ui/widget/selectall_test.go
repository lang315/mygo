package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// SelectAll selects the text of a box given the focus by Focus, whose caret is
// at its end otherwise: what is typed next replaces it.
func TestTextBoxSelectAll(t *testing.T) {
	for _, tc := range []struct {
		name string
		all  bool
		want string
	}{
		{"selected", true, "x"},
		{"not asked", false, "name.txtx"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tb := TextBox().SetText("name.txt").Focus()
			if tc.all {
				tb.SelectAll()
			}
			tt := tester(t, tb)
			tt.Type("x")
			tt.Frame()
			if got := tb.Text(); got != tc.want {
				t.Fatalf("Text %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMultilineTextBoxSelectAll(t *testing.T) {
	tb := MultilineTextBox().SetText("one\ntwo").Focus().SelectAll()
	tt := tester(t, tb)
	tt.Type("x")
	tt.Frame()
	if got := tb.Text(); got != "x" {
		t.Fatalf("Text %q, want x", got)
	}
}

// SelectAll asked after the box was shown still selects, once.
func TestTextBoxSelectAllAfterShown(t *testing.T) {
	tb := TextBox().SetText("name.txt").Focus()
	tt := tester(t, tb)
	tb.SelectAll()
	tt.Frame()
	tt.Type("x")
	tt.Frame()
	tt.Type("y")
	tt.Frame()
	if got := tb.Text(); got != "xy" {
		t.Fatalf("Text %q, want xy: selected once, then typing goes on", got)
	}
}

// In the dark appearance, whose Field differs from the window's, a
// DomainUpDown is filled with the Field and a button with the Surface (the
// inputs of ui are checked in package ui).
func TestInputsAreFilledWithFieldAndButtonsWithSurface(t *testing.T) {
	th := ui.DarkTheme()
	for _, c := range []struct {
		name  string
		w     Control
		label string
		want  ui.Color
	}{
		{"DomainUpDown", DomainUpDown("Mon", "Tue").SetSelectedItem("Tue"), "Tue", th.Field},
		{"Button", Button("Press"), "Press", th.Surface},
	} {
		tt := tester(t, c.w)
		tt.SetDark(true)
		tt.Frame()
		if got := pixelLeft(t, tt, c.label); !near(got, c.want) {
			t.Errorf("%s is filled %v, want %v", c.name, got, c.want)
		}
	}
}
