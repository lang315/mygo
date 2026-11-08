//go:build !mygo_noinspector

package ui

import "testing"

// The inspector's panel has the look of Chrome's tools, whatever theme the view
// set: its fields are the toolbar's face, not the theme's Field.
func TestInspectorPanelFieldsFollowItsSurface(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tt, _ := inspected(func(c *context) {
			th := LightTheme()
			if dark {
				th = DarkTheme()
			}
			c.SetTheme(th)
			coreColumn(c).Padding(20).Children(func() { coreText(c, "Hello") })
		})
		openInspector(tt)
		if th := &tt.rt.insp.theme; th.Field.A != 0 || th.FieldFill() != th.Surface {
			t.Errorf("dark=%v: the inspector's Field is %v, want none: its fields are %v", dark, th.Field, th.Surface)
		}
	}
}
