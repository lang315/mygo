package widget

import "testing"

// SelectAll asked of a box on a page that is not shown selects when the page
// is shown, whether it never was or was shown and hidden before.
func TestTextBoxSelectAllOnAHiddenPage(t *testing.T) {
	for _, shownBefore := range []bool{false, true} {
		name := "never shown"
		if shownBefore {
			name = "shown, hidden, then asked"
		}
		t.Run(name, func(t *testing.T) {
			tb := TextBox().SetText("name.txt")
			tabs := TabControl().AddPage("A", Label("first")).AddPage("B", tb)
			tt := tester(t, tabs)
			if shownBefore {
				tabs.SetPage(1)
				tt.Frame()
				tabs.SetPage(0)
				tt.Frame()
			}
			tb.SelectAll().Focus()
			tt.Frame()
			tt.Frame() // the page is still hidden: nothing was spent
			tabs.SetPage(1)
			tt.Frame()
			tt.Type("x")
			tt.Frame()
			if got := tb.Text(); got != "x" {
				t.Fatalf("Text %q, want x: the whole text selected and replaced", got)
			}
		})
	}
}
