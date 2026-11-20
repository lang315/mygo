---
name: mygo-widget-skin
description: Skin a MyGo ui/widget app with package ui/widget/skin and skin files (skin.Use, Calm, Classic, HighContrast, palettes, a brand color or accent, dark look, compact density, high contrast, looks, a skin file, a skin picker). Use when the user asks for a theme, a skin, a brand color, a denser or higher-contrast look, or to let the user of the app choose a look; use mygo-widget-app for the controls themselves.
---

# MyGo widget skins

A skin gives every `ui/widget` control one look: colors, sizes, radii, type, shadows, in light and dark. An app that never imports `skin` keeps the plain look and links none of it. Check the app's pinned MyGo version first and read the source it pins: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, then `docs/ui/skin.md` there. Confirm every API you use against that source.

## Turn a skin on

`skin.Use(skin.Calm)` at startup, before the window shows. Every window takes it in its next frame; call it from any goroutine. `skin.Use(nil)` brings back the plain look. A skin does not choose light or dark: `widget.SetTheme` does.

- `skin.Calm`: a calm light ground, the accent only on what acts or is chosen. The default choice.
- `skin.Classic`: WinForms-like (outlined controls, boxed tabs, filled grid headers), compact by default. For screens full of data.
- `skin.HighContrast`: text at 7:1, 2 DIP outlines. For users who need it; offer it, do not make it the default.

## Brand color and density

`skin.Skin.WithAccent` makes a palette from one color and moves it only as far as the contrast floors need. `skin.Skin.WithDensity(skin.Compact)` sets controls lower, at `Use`. `skin.Skin.WithPalette` takes one of the skin's own: `skin.Skin.Palettes` lists them.

```go
package main

import (
	"log"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

func main() {
	skin.Use(skin.Calm.WithAccent(ui.Hex("#7c3aed")).WithDensity(skin.Compact))
	err := widget.Run("Brand", func(w *widget.Form) {
		w.SetContent(widget.VBox(8).Add(
			widget.Label("Appearance"),
			skin.Picker(),
		).Padding(4))
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

Every `With...` returns a new skin and leaves the one it is called on. `skin.HighContrast` has no palettes, but `WithAccent` works on it.

## Derive a skin from the tokens

Take a skin's tokens, change what you want, and make a skin with `skin.Skin.Derive`, which keeps the palette, the density, the looks and the contrast floor. `skin.New` keeps none of them. `skin.Skin.Tokens` returns the sizes of `skin.Standard`, whatever the density. A font of the app's own comes in with `ui.RegisterFont`, then `Tokens.Font`.

## Looks

`skin.Looks` sets the shape of each kind of control as `tw` classes with states: `skin.Skin.WithLooks` replaces a kind whole and drops the kinds left empty. Start from `skin.Skin.Looks` to keep the rest.

The `Badge`, `InfoBar` and `EmptyState` looks style those controls. In the `Badge` and `InfoBar` looks, `bg-status`, `text-status` and `border-status` (`widget.RoleStatus`, `widget.RoleStatusText`, `widget.RoleStatusBorder`) are the colors of the control's own status, so one look serves every status: `Badge: "rounded-sm border border-status"`.

The `SettingsCard` look styles each section's card of `widget.FieldsWidget.SettingsRows`; left "", the cards take the `Card` look.

```go
package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
)

func warm() *skin.Skin {
	l, d := skin.Calm.Tokens(false), skin.Calm.Tokens(true)
	l.Background = ui.Hex("#fffaf0")
	return skin.Calm.Derive("Warm", l, d)
}

func boxy() *skin.Skin {
	return warm().WithLooks(skin.Looks{
		Button:    "rounded-none border border-button",
		TabHeader: "rounded-t border border-b-0 selected:bg-surface",
	})
}
```

## A skin file shipped with the app

A designer edits JSON, not Go. Embed the file, parse it, and fall back to a built-in skin when it does not read: `mygo dev` relaunches the app on each edit of an embedded file. The keys and examples are in [skinfile.md](skinfile.md); `skinfile.Format` writes a skin back as a file.

```go
package app

import (
	_ "embed"
	"log"

	"github.com/egoist/mygo/ui/widget/skin"
	"github.com/egoist/mygo/ui/widget/skin/skinfile"
)

//go:embed brand.json
var brandJSON []byte

// brand is the app's skin: its file's, or Calm when the file does not read.
func brand() *skin.Skin {
	s, err := skinfile.Parse(brandJSON)
	if err != nil {
		log.Print(err)
		return skin.Calm
	}
	return s
}

// Settings is what the app keeps between runs.
type Settings struct{ Skin string }

// start turns on the skin the user chose last, or the brand's.
func start(st Settings) {
	if s := skin.FromID(st.Skin, brand()); s != nil {
		skin.Use(s)
		return
	}
	skin.Use(brand())
}

// save keeps the choice: the ID names the skin, its palette and its density.
func save(st *Settings) {
	if s := skin.Active(); s != nil {
		st.Skin = s.ID()
	}
}
```

`skin.Picker` lets the user choose the skin, its palette and the density; pass it the app's own skins (`skin.Picker(brand(), skin.Classic)`) or none for the built-in three. `skin.FromID` returns nil when any part of an ID is unknown: always fall back.

## Check the contrast in a test

`skin.Skin.Check` returns an error naming every pair under its floor (4.5:1 for text, 7:1 for `HighContrast`, 3:1 for outlines). `Use` does not check, and a failing skin still works, so a test is where it is caught. Parse does not check either.

```go
package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/skin"
)

func TestSkinsPassContrast(t *testing.T) {
	for _, s := range []*skin.Skin{
		skin.Calm.WithAccent(ui.Hex("#7c3aed")),
		skin.Classic,
		skin.HighContrast,
	} {
		if err := s.Check(); err != nil {
			t.Errorf("%s: %v", s.Name(), err)
		}
	}
}
```

## Pitfalls

- Set colors in an app's own controls through roles (`widget.RoleChrome`, `widget.RoleBorderStrong`, `widget.RoleControlBorder`, `widget.RoleRowSelected`, `widget.RoleInfo`) or `tw` classes such as `bg-chrome`, not fixed hex, so the skin and the dark appearance reach them.
- A control's own methods and `Apply`'d styles win over its look, and the look wins over the rest of the skin. A look that seems to do nothing is usually beaten by a style on the control.
- A look does not place a control: its `halign-*` and `hexpand` are ignored.
- `WithDensity` changes sizes at `Use`, not in `Tokens`: a skin made with `skin.New` from a compact skin's tokens is standard until given `skin.Compact` again.
- `palette` and `accent` do not go together in a file; `HighContrast` has no palettes for `palette` to name.
- A skin of an accent near the status colors can fail `Check` on Info: give `Tokens.Info` a family of its own.
