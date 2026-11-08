package widget

import "testing"

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
