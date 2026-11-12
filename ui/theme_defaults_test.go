package ui

import "testing"

// defaultThemesFixture returns the app themes the tests of SetDefaultThemes
// set: a light one with its own accent and background, a dark one with its
// own accent.
func defaultThemesFixture() (light, dark *Theme) {
	light = LightTheme()
	light.Accent = Hex("#7c3aed")
	light.Background = Hex("#fafafa")
	dark = DarkTheme()
	dark.Accent = Hex("#a78bfa")
	return light, dark
}

// SetDefaultThemes gives every window the app's theme of each appearance, in
// its next frame, and a later window too; a theme a view sets itself is its
// own; nil leaves an appearance its default.
func TestSetDefaultThemes(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	light, dark := defaultThemesFixture()
	var got Theme
	view := func(c *context) { got = *c.Theme() }

	tt := coreNewTester(view, 100, 100)
	if got.Accent != LightTheme().Accent {
		t.Fatalf("before the call, the accent is %v, want LightTheme's %v", got.Accent, LightTheme().Accent)
	}

	SetDefaultThemes(light, dark)
	tt.Frame()
	if got.Accent != Hex("#7c3aed") || got.Background != Hex("#fafafa") {
		t.Errorf("a window open when it was called shows accent %v and background %v, want #7c3aed and #fafafa", got.Accent, got.Background)
	}
	tt.SetDark(true)
	if got.Accent != Hex("#a78bfa") || !got.Dark {
		t.Errorf("dark, the accent is %v (dark %v), want #a78bfa", got.Accent, got.Dark)
	}
	tt.SetDark(false)
	if got.Accent != Hex("#7c3aed") {
		t.Errorf("back to light, the accent is %v, want #7c3aed", got.Accent)
	}

	other := coreNewTester(view, 100, 100)
	other.SetDark(true)
	if got.Accent != Hex("#a78bfa") {
		t.Errorf("a window opened later, dark, shows %v, want #a78bfa", got.Accent)
	}

	coreNewTester(func(c *context) {
		c.SetTheme(LightTheme())
		got = *c.Theme()
	}, 100, 100)
	if got.Accent != LightTheme().Accent {
		t.Errorf("a theme set by the view has the accent %v, want its own %v", got.Accent, LightTheme().Accent)
	}

	SetDefaultThemes(nil, dark)
	coreNewTester(view, 100, 100)
	if got.Accent != LightTheme().Accent {
		t.Errorf("with no light theme, a light window has the accent %v, want LightTheme's %v", got.Accent, LightTheme().Accent)
	}
	d := coreNewTester(view, 100, 100)
	d.SetDark(true)
	if got.Accent != Hex("#a78bfa") {
		t.Errorf("with only a dark theme, a dark window has the accent %v, want #a78bfa", got.Accent)
	}

	SetDefaultThemes(nil, nil)
	tt.Frame()
	if got.Accent != LightTheme().Accent || got.Background != LightTheme().Background {
		t.Errorf("after nil, nil, the open light window has accent %v and background %v, want LightTheme's", got.Accent, got.Background)
	}
	d.Frame()
	if got.Accent != DarkTheme().Accent {
		t.Errorf("after nil, nil, the open dark window has accent %v, want DarkTheme's %v", got.Accent, DarkTheme().Accent)
	}
}

// SetControlBorder's colors are the app's own setting: they win over the
// ControlBorder of the default themes, and once reset the app theme's own
// comes back.
func TestSetDefaultThemesUnderControlBorder(t *testing.T) {
	t.Cleanup(func() {
		SetDefaultThemes(nil, nil)
		SetControlBorder(Color{}, Color{})
	})
	light, dark := defaultThemesFixture()
	light.ControlBorder = Hex("#abcdef")
	var got Color
	view := func(c *context) { got = c.Theme().ControlBorder }

	SetControlBorder(Hex("#123456"), Color{})
	SetDefaultThemes(light, dark)
	tt := coreNewTester(view, 100, 100)
	if got != Hex("#123456") {
		t.Errorf("light, the control border is %v, want SetControlBorder's #123456", got)
	}
	tt.SetDark(true)
	if got != dark.ControlBorder {
		t.Errorf("dark, with no dark color set, the control border is %v, want the app theme's %v", got, dark.ControlBorder)
	}

	SetControlBorder(Color{}, Color{})
	tt.SetDark(false)
	if got != Hex("#abcdef") {
		t.Errorf("after SetControlBorder was reset, the light control border is %v, want the app theme's #abcdef", got)
	}
}

// The desktop's accent color does not replace an app theme's accent,
// selection or focus; without app themes, or after they are removed, the
// window follows the desktop's accent as before.
func TestSetDefaultThemesKeepsItsAccent(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	light, dark := defaultThemesFixture()
	desktop := Hex("#e11d48")
	var got Theme
	tt := coreNewTester(func(c *context) { got = *c.Theme() }, 100, 100)
	tt.SetPreferences(Preferences{Accent: desktop, TextScale: 1})
	if got.Accent != desktop {
		t.Fatalf("without app themes, the accent is %v, want the desktop's %v", got.Accent, desktop)
	}

	SetDefaultThemes(light, dark)
	tt.Frame()
	if got.Accent != Hex("#7c3aed") {
		t.Errorf("with app themes, the accent is %v, want the app's #7c3aed", got.Accent)
	}
	if got.Selection != light.Selection || got.Focus != light.Focus {
		t.Errorf("with app themes, selection %v and focus %v, want the app theme's %v and %v", got.Selection, got.Focus, light.Selection, light.Focus)
	}
	if got.AccentHover != light.AccentHover || got.AccentText != light.AccentText {
		t.Errorf("with app themes, accent hover %v and text %v, want the app theme's %v and %v", got.AccentHover, got.AccentText, light.AccentHover, light.AccentText)
	}

	// Only the appearance with an app theme keeps its accent: a light window
	// with only a dark app theme follows the desktop.
	SetDefaultThemes(nil, dark)
	tt.Frame()
	if got.Accent != desktop {
		t.Errorf("with only a dark app theme, the light accent is %v, want the desktop's %v", got.Accent, desktop)
	}

	SetDefaultThemes(nil, nil)
	tt.Frame()
	if got.Accent != desktop {
		t.Errorf("after the app themes went, the accent is %v, want the desktop's %v again", got.Accent, desktop)
	}
}

// High Contrast and the user's text size still apply to an app theme.
func TestSetDefaultThemesFollowsContrastAndTextSize(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	light, dark := defaultThemesFixture()
	p := Preferences{HighContrast: true, TextScale: 1.5}
	var got Theme
	SetDefaultThemes(light, dark)
	tt := coreNewTester(func(c *context) { got = *c.Theme() }, 100, 100)
	tt.SetPreferences(p)
	if got.FontSize != light.FontSize*1.5 {
		t.Errorf("FontSize is %v, want %v at 150%%", got.FontSize, light.FontSize*1.5)
	}
	want := light.WithPreferences(p).ControlBorder
	if got.ControlBorder == light.ControlBorder || got.ControlBorder != want {
		t.Errorf("under High Contrast, ControlBorder is %v, want %v (the app theme's is %v)", got.ControlBorder, want, light.ControlBorder)
	}
}

// The call copies the themes: changing one afterwards changes no window.
func TestSetDefaultThemesCopies(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	light, dark := defaultThemesFixture()
	var got Color
	tt := coreNewTester(func(c *context) { got = c.Theme().Accent }, 100, 100)
	SetDefaultThemes(light, dark)
	light.Accent = Hex("#000000")
	tt.Frame()
	if got != Hex("#7c3aed") {
		t.Errorf("after the caller changed its theme, the window reads %v, want the copy's #7c3aed", got)
	}
	// A frame that rebuilds the theme (a change of appearance and back)
	// still reads the copy.
	tt.SetDark(true)
	tt.SetDark(false)
	if got != Hex("#7c3aed") {
		t.Errorf("after the theme was rebuilt, the window reads %v, want the copy's #7c3aed", got)
	}
}

// A theme that already followed the desktop's preferences (WithPreferences)
// is taken as the app wrote it: the accent it was given stays, not the one
// following put back.
func TestSetDefaultThemesTakesAFollowedTheme(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	x := Hex("#0d9488")
	light := LightTheme().WithPreferences(Preferences{Accent: x, TextScale: 1})
	var got Color
	tt := coreNewTester(func(c *context) { got = c.Theme().Accent }, 100, 100)
	SetDefaultThemes(light, nil)
	tt.Frame()
	if got != x {
		t.Errorf("a theme from WithPreferences gives the accent %v, want its own %v", got, x)
	}
}

// The rows a list replays repaint with the new default theme: painting reads
// the theme, which the generation the call bumps covers.
func TestRowsPaintAgainWithNewDefaultThemes(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	light, _ := defaultThemesFixture()
	var list ListState
	tt := coreNewTester(func(c *context) {
		coreList(c, &list, 100, func(i int) {
			coreBox(c).Size(20, 20).Background(c.Theme().Accent)
		}).Grow(1)
	}, 300, 400)
	for range 4 {
		tt.Frame()
	}
	tt.rt.replayCheck = true
	tt.Frame()
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame: the test shows nothing")
	}
	SetDefaultThemes(light, nil)
	tt.Frame()
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows differ from painting them", tt.rt.replayMismatch)
	}
	want := Hex("#7c3aed")
	if px := tt.Image().RGBAAt(5, 5); px.R != want.R || px.G != want.G || px.B != want.B {
		t.Fatalf("the box in a row is %v, want the new accent %v", px, want)
	}
}

// The rows a list keeps (RowVersion) were built with the theme the pass lent
// them: new default themes build them again (themeGen).
func TestKeptRowsRebuildWithNewDefaultThemes(t *testing.T) {
	t.Cleanup(func() { SetDefaultThemes(nil, nil) })
	light, _ := defaultThemesFixture()
	s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
	var built []Color
	tt := coreNewTester(func(c *context) {
		coreList(c, s, 6, func(i int) {
			built = append(built, c.Theme().Accent)
			coreBox(c).Size(20, 20).Background(c.Theme().Accent)
		}).Grow(1)
	}, 300, 300)
	tt.Frame()
	tt.Frame()
	if tt.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test shows nothing")
	}
	built = built[:0]
	SetDefaultThemes(light, nil)
	tt.Frame()
	if len(built) == 0 {
		t.Fatal("no row was built again after the default themes changed")
	}
	for _, a := range built {
		if a != Hex("#7c3aed") {
			t.Errorf("a kept row was built with the accent %v, want #7c3aed", a)
		}
	}
	want := Hex("#7c3aed")
	if px := tt.Image().RGBAAt(5, 5); px.R != want.R || px.G != want.G || px.B != want.B {
		t.Errorf("the box in a kept row is %v, want the new accent %v", px, want)
	}
}
