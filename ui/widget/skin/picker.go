package skin

import (
	"strconv"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui/widget"
)

// Picker returns a form of three boxes for the app's user to choose a skin
// among skins (Builtin when none), its palette among the skin's Palettes,
// and the density, as DevExpress's skin selectors. Each choice calls Use. It
// starts at Active, or at the first skin when Active is none of them, and
// turns nothing on until the user chooses. Choosing another skin keeps the
// palette of the same name, if the skin has one, and the density.
//
// The app keeps the choice in its own settings, as the skin's ID, and
// restores it at startup:
//
//	settings.Skin = skin.Active().ID() // when it saves its settings
//	if s := skin.FromID(settings.Skin); s != nil {
//		skin.Use(s)
//	}
func Picker(skins ...*Skin) widget.Control {
	if len(skins) == 0 {
		skins = Builtin()
	}
	// The choice: a skin of skins, a palette (nil for the skin's own) and
	// the density. Only the boxes' handlers change it, on the event
	// goroutine.
	i, palette, density := 0, (*Palette)(nil), skins[0].Density()
	if a := Active(); a != nil {
		for j, s := range skins {
			if nameOf(s) == a.Name() {
				i, palette, density = j, a.Palette(), a.Density()
				break
			}
		}
	}

	var opts []widget.Option
	for j, s := range skins {
		opts = append(opts, widget.Option{Value: strconv.Itoa(j), Text: nameOf(s)})
	}
	skinBox := widget.ComboBox(opts...).SetValue(strconv.Itoa(i))
	paletteBox := widget.ComboBox()
	densityBox := widget.ComboBox(
		widget.Option{Value: "standard", Text: l10n.StandardDensity()},
		widget.Option{Value: "compact", Text: l10n.CompactDensity()})

	// palettes lists the chosen skin's palettes in the palette box, the
	// chosen one shown. It builds the skin.
	palettes := func() {
		var opts []widget.Option
		for _, p := range skins[i].Palettes() {
			opts = append(opts, widget.Option{Value: p.Name(), Text: p.Name()})
		}
		p := palette
		if p == nil {
			p = skins[i].Palette()
		}
		paletteBox.SetOptions(opts...).SetDisabled(len(opts) == 0)
		if p != nil {
			paletteBox.SetValue(p.Name())
		}
	}
	use := func() {
		Use(skins[i].WithPalette(palette).WithDensity(density))
	}
	palettes()
	if density == Compact {
		densityBox.SetValue("compact")
	} else {
		densityBox.SetValue("standard")
	}

	skinBox.OnChange(func() {
		i, _ = strconv.Atoi(skinBox.Value())
		var kept *Palette
		if palette != nil {
			for _, p := range skins[i].Palettes() {
				if p.Name() == palette.Name() {
					kept = p
					break
				}
			}
		}
		palette = kept
		palettes()
		use()
	})
	paletteBox.OnChange(func() {
		for _, p := range skins[i].Palettes() {
			if p.Name() == paletteBox.Value() {
				palette = p
			}
		}
		use()
	})
	densityBox.OnChange(func() {
		density = Standard
		if densityBox.Value() == "compact" {
			density = Compact
		}
		use()
	})
	return widget.Fields().
		Row(l10n.SkinPickerSkin(), skinBox).
		Row(l10n.SkinPickerPalette(), paletteBox).
		Row(l10n.SkinPickerDensity(), densityBox)
}
