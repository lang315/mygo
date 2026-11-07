package ui

import "testing"

// SetDefaultBorder gives the default themes of both appearances the border
// colors, for apps with no Context of their own to set a Theme with, and a
// change of appearance keeps them.
func TestSetDefaultBorder(t *testing.T) {
	var got Color
	view := func(c *context) { got = c.Theme().Border }
	light, dark := Hex("#9ca3af"), Hex("#6b7280")

	tt := coreNewTester(view, 100, 100)
	if got != LightTheme().Border {
		t.Fatalf("before the call, the light border is %v, want the theme's %v", got, LightTheme().Border)
	}

	SetDefaultBorder(light, dark)
	t.Cleanup(func() { SetDefaultBorder(Color{}, Color{}) })

	tt.Frame()
	if got != light {
		t.Errorf("a window open when it was called shows the light border %v, want %v", got, light)
	}
	tt.SetDark(true)
	if got != dark {
		t.Errorf("after the appearance went dark, the border is %v, want %v", got, dark)
	}
	tt.SetDark(false)
	if got != light {
		t.Errorf("after it came back, the border is %v, want %v", got, light)
	}

	other := coreNewTester(view, 100, 100)
	other.SetDark(true)
	if got != dark {
		t.Errorf("a window opened later, dark, shows %v, want %v", got, dark)
	}

	// A Theme the view sets itself is its own: the call changes the defaults only.
	own := LightTheme()
	coreNewTester(func(c *context) {
		c.SetTheme(own)
		got = c.Theme().Border
	}, 100, 100)
	if got != own.Border {
		t.Errorf("a theme set by the view has the border %v, want its own %v", got, own.Border)
	}

	// A zero color leaves a theme's own, in each appearance.
	SetDefaultBorder(Color{}, dark)
	coreNewTester(view, 100, 100)
	if got != LightTheme().Border {
		t.Errorf("with no light color, the light border is %v, want the theme's %v", got, LightTheme().Border)
	}
}
