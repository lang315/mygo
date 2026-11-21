# Skins

Package `ui/widget/skin` gives every control of [`ui/widget`](widget.md) one
look, set by tokens: colors, sizes, radii, type, shadows, the focus ring and
the scroll bars, in light and dark. A `Skin` is a value. An app turns one on with
one line, makes its own from another's tokens, or reads one from a file.

## Using a skin

```go
import "github.com/egoist/mygo/ui/widget/skin"

skin.Use(skin.Calm)
```

`skin.Use` gives every window the skin in its next frame. Call it at startup
as a rule. It may change the skin while the app runs, from any goroutine.
It does not change the appearance, light or dark: `widget.SetTheme` chooses
that, and a view that sets its own theme keeps it. `skin.Active` returns the
skin `Use` set, or nil.

`Use(nil)` brings back the plain look. So does an app that never calls `Use`,
and one that does not import the package links none of it.

MyGo ships three skins, each made only of tokens, palettes and looks, as an
app's own can be:

- **`Calm`** is a calm light ground with the accent only on what acts or is
  chosen: the primary action, the selection, the open tab and checked
  controls. Green, amber and red appear only in status. Controls are 36 DIPs
  high, table rows 30.
- **`Classic`** is for screens full of data and for people who know WinForms:
  corners of 2 DIPs, outlined buttons and group boxes, boxed tabs whose open
  one joins its page, grid headers filled with the chrome. It is `Compact` by
  default: controls 30 DIPs high, rows 26.
- **`HighContrast`** is for people who need text to stand out: text at 7:1 or
  more, no muted text, every control outlined 2 DIPs wide, a 3 DIP focus ring
  and no shadows. It does not replace the system's own High Contrast, which
  still applies over any skin. It has no palettes, but `Skin.WithPalette` and
  `Skin.WithAccent` work on it, and `Check` reports what falls under 7:1.

`skin.Builtin` returns the three, in that order, in a new slice each call. It
builds none of them: `Classic` and `HighContrast` are made at their first
use.

## Palettes and accent

The accent of a skin is its `Palette`: the accent family, the Info family
(the running state), the selected row and the text selection, in light and
dark. `Skin.WithPalette` returns a new skin with a palette; the skin it is
called on does not change, and `WithPalette(nil)` returns it.

```go
skin.Use(skin.Calm.WithPalette(skin.Teal))
```

`Skin.Palettes` lists the palettes made for a skin, its own first, in a new
slice. `Skin.Palette` returns the one it has, nil for a skin made with `New`
until it is given one. `Palette.Name` returns a palette's name.

| Skin | Palettes |
|---|---|
| `Calm` | `Blue` (its own), `Cham` (named "Chàm"), `Indigo`, `Ink`, `Teal` |
| `Classic` | `ClassicIndigo` (its own, named "Indigo"), `ClassicSlate` ("Slate") |
| `HighContrast` | none |

Each palette MyGo ships passes the skin's contrast floors and keeps its Info
apart from the status colors, for a red-green color-blind eye too. `Teal`
has a blue Info of its own, as its accent sits too near green.

### A brand color

`Skin.WithAccent` makes a palette from one color, named by its hex
(`"#7c3aed"`), and gives it to the skin:

```go
skin.Use(skin.Calm.WithAccent(ui.Hex("#7c3aed")))
```

It moves the color only as far as the floors need. In light, the color stays
as it is when white text on it reads, it stands out on the chrome (3:1) and
text reads on the selected row mixed from it. A lighter color, as a yellow,
darkens to the lightest of its hue that passes; a near-black one lightens just
enough. In dark, it is lightened until the dark color of its hue reads on it.
Info is the accent, unless the accent sits near the status colors: then Info
stays the skin's. The alpha of the color is ignored.

### A palette of your own

`NewPalette` makes a palette from the accent families of light and dark, and
its Info is its accent. `LightFamily` and `DarkFamily` make each family from
a base, hover, pressed and text color (and, in dark, the color on the base):

```go
brand := skin.NewPalette("Brand",
	skin.LightFamily(ui.Hex("#7c3aed"), ui.Hex("#6d28d9"), ui.Hex("#5b21b6"), ui.Hex("#6d28d9")),
	skin.DarkFamily(ui.Hex("#a78bfa"), ui.Hex("#c4b5fd"), ui.Hex("#8b5cf6"), ui.Hex("#ddd6fe"), ui.Hex("#1e1033")))
skin.Use(skin.Calm.WithPalette(brand))
```

`LightFamily`'s `On` is always white. For a light base, set the returned
`Family`'s `On` to a dark color: `Check` reports white on it. An accent near
the status colors makes an Info near them too, which `Check` reports; give
`Tokens.Info` a family of its own through the tokens, as in
[Making a skin](#making-a-skin).

## Density

`Density` is how tight a skin sets its controls: `Standard`, the skin's own
sizes, or `Compact`, for screens full of data.

```go
skin.Use(skin.Calm.WithDensity(skin.Compact))
```

`Skin.WithDensity` returns a new skin. `Compact` makes controls and tab
headers 6 DIPs lower, rows, headers and the menu bar 4, a settings row 8, and
the tool strip's room as much smaller as its controls. Icons, check boxes,
switches, corners and text keep their size. `Skin.Density` returns the density.

The heights are least heights, never under the font size plus 8: a control
whose text and padding need more stays taller, as Windows' Segoe UI does at
`Calm`'s 30.

The sizes change at `Use`. `Skin.Tokens` stays `Standard`, so a skin made with
`New` from a compact skin's tokens is `Standard` until you give it `Compact`
again. `Skin.Derive` keeps the density.

## Making a skin

`Tokens` are every value a skin decides in one appearance. The pieces are
`Family` (a role's colors: the accent, Info and the status colors), `Type` (a
role of text: its size and weight) and `Shadow` (a drop shadow). Take a skin's
tokens, change what you want, and make a skin of them:

```go
l, d := skin.Calm.Tokens(false), skin.Calm.Tokens(true)
l.Background, l.RowHeight = ui.Hex("#fffaf0"), 28
ui.RegisterFont(interTTF, "Inter")
l.Font, d.Font = "Inter", "Inter"
s := skin.New("Brand", l, d).WithPalette(brand)
skin.Use(s)
```

`skin.New` copies the tokens and checks nothing. It loses what the skin they
came from had besides tokens. `Skin.Derive` keeps it: it returns a new skin
with a name and tokens of your choice, and the palette, palette list, density,
looks and contrast floor of the skin it is called on.

```go
skin.Use(skin.Calm.WithDensity(skin.Compact).Derive("Brand", l, d))
```

`Skin.Tokens(dark)` returns a copy of the tokens of one appearance, and
`Skin.Name` the name.

### Check

`Skin.Check` measures the contrast of both appearances and returns an error
naming the appearance and every failing pair, nil when all pass:

- text on its grounds, at the skin's floor: 4.5:1, and 7:1 for a skin made for
  low vision (`HighContrast`);
- outlines, the accent line and each chart color, 3:1;
- the status families as badges and info bars draw them: `Base` on `Subtle`
  (the running dot, the icon) 3:1, and `Text` on `Subtle` laid over a selected
  and a hovered row at the text floor, as "Info Text on Subtle over
  RowSelected"; `TextMuted` on `SurfaceHover` (`StatusNone`);
- the control states (hover, pressed) apart from their ground and from each
  other;
- Info apart from the status colors.

A skin that fails still works: `Use` does not check. Call `Check` in a test of
the app, or log its error at startup.

```go
if err := s.Check(); err != nil {
	log.Print(err) // the pairs under their floors
}
```

## Looks

Tokens set colors and sizes. `Looks` set the shape of each kind of control as
Tailwind classes ([the cheat sheet](widget/tw.md)) with their states, so two
skins can differ as much as their controls do. A field left "" keeps the
kind's plain look.

| Field | Styles |
|---|---|
| `Button`, `ToolButton` | a `Button`'s face; a `ToolStrip`'s buttons |
| `Field` | the box of `TextBox`, `MultilineTextBox`, `MaskedTextBox`, `NumericUpDown`, `DomainUpDown`, `DateTimePicker` and an editable `ComboBox` |
| `ComboBox` | a drop-down `ComboBox` |
| `CheckBox`, `RadioButton` | the whole control, with its text |
| `GroupBox`, `Card` | the frames of the controls so named |
| `TabStrip`, `TabHeader` | a `TabControl`'s strip; each tab, `selected:` the open one |
| `GridHeader` | each column header of a `DataGridView`, and of a `ListView` in Details view |
| `ToolStrip`, `StatusStrip`, `MenuBar`, `HeaderBar` | the strips so named, and a `Form`'s header bar |
| `Badge` | a `Badge`'s pill, without its detail |
| `InfoBar` | an `InfoBar`'s box |
| `EmptyState` | an `EmptyState`'s box, around its icon, text and action |
| `SettingsCard` | each section of a `Fields`' `SettingsRows`, the card around its rows; "" takes the `Card` look |

```go
boxy := skin.Calm.WithLooks(skin.Looks{
	Button:    "rounded-none border border-button",
	TabHeader: "rounded-t border border-b-0 selected:bg-surface",
})
skin.Use(boxy)
```

`Skin.WithLooks` returns a new skin. Each kind of the `Looks` you give
replaces the skin's look of that kind whole, and the kinds you leave "" go: to
keep part of `Classic`'s look, start from its `Skin.Looks`, change a field and
pass the result to `Skin.WithLooks`.
A class `tw` does not know is logged once and skipped; the rest of the look
applies. `Skin.Looks` returns the looks.

A look may name the skin's colors by role (`bg-chrome`, `border-control`, …)
and its states with `hover:`, `focus:`, `active:`, `disabled:`, `dark:` and
`selected:` (a `CheckBox` or `RadioButton` checked, a toggle `ToolButton` on,
the open tab). A look does not place a control: its `halign-*`, `hexpand` and
the like are ignored.

A control's own methods win over its look, and the look wins over the rest of
the skin. The full order, with a `TabControl`'s `StripStyle`, is in
[Widgets that live on](widget.md#styles), where the style methods are.

An app's own controls draw in the skin's colors through the skin roles of
`ui/widget`: `widget.RoleChrome`, `RoleBorderStrong`, `RoleControlBorder`,
`RoleButtonBorder`, `RoleRowHover`, `RoleRowSelected` and `RoleInfo`. Set a
color through a role, not a fixed hex, and it follows the skin and the dark
appearance.

The status roles, `widget.RoleStatus`, `RoleStatusText` and
`RoleStatusBorder` (`bg-status`, `text-status` and `border-status` in a look),
are the fill, text and border of the status of the control they style: the
status family's `Subtle`, `Text` and `Border` (`TextMuted` on `SurfaceHover`,
in `Border`, for `StatusNone`). They are meant for the `Badge` and `InfoBar`
looks, so that one look serves every status, as Classic's
`rounded-sm border border-status` does; in any other control they are
`StatusNone`'s colors. Without a skin the fill is the status's theme color at
14 % alpha, the text the theme's `Text`, and the border none.

## Skin files

Package `ui/widget/skin/skinfile` reads and writes skins as JSON files. A
file names the skin it starts from, its palette or accent, its density, the
tokens it changes in light and dark, and its looks:

```json
{
  "version": 1,
  "name": "Brand",
  "base": "Calm",
  "accent": "#7c3aed",
  "density": "compact",
  "light": { "Background": "#fffaf0", "Font": "Inter", "Danger": { "Base": "#c62828" } },
  "dark": { "Background": "#121212" },
  "looks": { "Button": "rounded-none border border-button" }
}
```

`skinfile.Parse` returns the skin of a file's bytes. An app embeds the file
and turns it on at startup; `mygo dev` relaunches the app on each edit of an
embedded file, so the skin shows as it is saved:

```go
//go:embed brand.json
var brandJSON []byte

s, err := skinfile.Parse(brandJSON)
if err != nil {
	log.Fatal(err)
}
skin.Use(s)
```

An app may also read a file at run time and pass its bytes to `Parse`.

### The keys

| Key | Value |
|---|---|
| `version` | `1`. Required. |
| `name` | the skin's name. Required. |
| `base` | the skin to start from: `"Calm"`, `"Classic"` or `"HighContrast"`. Required. |
| `palette` | the name of one of the base's `Palettes`, as `"Teal"`. |
| `accent` | a color `"#rrggbb"` or `"#rrggbbaa"`, as `skin.Skin.WithAccent` takes it (its alpha is ignored). |
| `density` | `"standard"` or `"compact"`. |
| `light`, `dark` | the tokens to change in each appearance. |
| `looks` | the looks to change, by the names of `Looks`' fields. |

Give `palette` or `accent`, not both. `HighContrast` has no palettes, so it
takes `accent` only.

Parse applies the file in this order: the base, its palette or the accent,
the density, the tokens of `light` and `dark`, then the looks. So a token in
`light` wins over the accent, and a kind of look replaces the base's whole.

The keys of `light` and `dark` are the names of `Tokens`' fields. A field
that is a struct (a `Family`, a `Type`, a `Shadow`, the `ScrollbarStyle`) is an
object of its own fields, each optional. Colors are `"#rrggbb"` or
`"#rrggbbaa"`, sizes numbers of DIPs, and `Chart` an array of all 8 colors.
The tokens are the `Standard` ones: `density` applies on top, at `Use`.

An error names the path and the line, as `path: message (line N)`:

```
light.Background: want a color #rrggbb or #rrggbbaa, got "red" (line 6)
```

An unknown key, a value of the wrong kind, a key twice, an unknown palette, a
class `tw` does not know and data after the object are errors. `Parse` does
not check the skin's contrast: call `Skin.Check`.

### Writing a file

`skinfile.Format` returns the file of a skin as what it changes from a base:
its palette or accent, its density if not the base's, the tokens that differ
from the base's after that palette and density, and the kinds of looks that
differ. A nil base writes every token and every look, against `Calm`.

```go
data := skinfile.Format(s, skin.Calm)
```

A palette that is neither one of the base's `Palettes` nor an accent's is not
named: the tokens carry it. Keys come in the order of `Tokens`' fields.

An app that imports `skin` but not `skinfile` links neither it nor its JSON
reader.

## The studio

`examples/skin-studio` shows every control of `ui/widget` in every skin,
palette and appearance. Three combo boxes in the tool strip switch them while
the app runs. The pages are the families of controls: buttons, text, numbers,
the grid, lists, containers, charts, dialogs.

Its **Edit** tab is the editor for a skin of your own. Choose a base, a
palette or an accent, the density and the appearance to show; change any
token with its own control (a hex box and a color dialog for a color, a
number box for a size, a check box for a flag) and any look as classes. Each edit turns the skin on at once, and the
status strip reports `Check`: how many pairs are under their floors, with the
pairs in its tooltip. **Open…** reads a skin file into the editor, with a
`Parse` error shown by the button; **Save…** writes the file that
`skinfile.Format` makes against the base.

```sh
go run ./examples/skin-studio
```

## Letting the user choose

`Picker` returns a form of three boxes for the user to choose a skin among
the skins given (`Builtin` when none), its palette among the skin's own, and
the density. Each choice calls `Use`. It starts at `Active`, and turns nothing
on until the user chooses. Choosing another skin keeps the palette of the same
name, if the skin has one, and the density.

```go
w.SetContent(skin.Picker())
```

The app keeps the choice in its own settings as `Skin.ID`, and gets the skin
back at startup with `FromID`:

```go
settings.Skin = skin.Active().ID() // when it saves its settings

if s := skin.FromID(settings.Skin); s != nil {
	skin.Use(s)
} else {
	skin.Use(skin.Calm) // first run, or a skin that is gone
}
```

An ID names the skin, its palette and its density, as `"Calm/Teal/compact"`:
the palette by its name, a `WithAccent` one by its color as `"#rrggbb"`, and
nothing for a skin without a palette (`"HighContrast//standard"`). Looks are
not in it: they are the skin's of that name.

`FromID` looks for the skin by name among the skins you pass after the ID (an
app's own), then in `Builtin`; it returns nil when any part of the ID is
unknown, for the app to fall back to its default.

```go
if s := skin.FromID(saved, brand); s != nil {
	skin.Use(s)
}
```
