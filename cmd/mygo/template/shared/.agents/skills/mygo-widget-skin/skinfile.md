# Skin files

A skin file is JSON that `skinfile.Parse` reads into a `skin.Skin`. Read it with the guide's section on skin files (`docs/ui/skin.md` in the pinned MyGo) when a key here is not enough.

## Keys

| Key | Value |
|---|---|
| `version` | `1`. Required. |
| `name` | the skin's name. Required. |
| `base` | `"Calm"`, `"Classic"` or `"HighContrast"`. Required. Nothing else is a base. |
| `palette` | the name of one of the base's palettes: Calm has `Blue`, `Chàm`, `Indigo`, `Ink`, `Teal`; Classic has `Indigo`, `Slate`; HighContrast has none. |
| `accent` | one color, `"#rrggbb"` or `"#rrggbbaa"`; the skin gets a palette made from it. |
| `density` | `"standard"` or `"compact"`. |
| `light`, `dark` | the tokens to change in each appearance. |
| `looks` | the looks to change. |

Give `palette` or `accent`, not both. Every key but the three required ones is optional, and a key given twice is an error.

Parse applies the file in this order: the base, its palette or the accent, the density, the tokens of `light` and `dark`, then the looks. A token set in `light` wins over the accent.

## Tokens

The keys of `light` and `dark` are the names of the fields of `skin.Tokens`, each optional:

- Colors, `"#rrggbb"` or `"#rrggbbaa"`: `Chrome`, `Background`, `Border`, `BorderStrong`, `Text`, `TextMuted`, `TextPlaceholder`, `Surface`, `SurfaceHover`, `SurfacePressed`, `ButtonBorder`, `ControlBorder`, `Selection`, `RowHover`, `RowSelected`.
- Families, objects of `Base`, `Hover`, `Pressed`, `Subtle`, `Border`, `Text`, `On`: `Accent`, `Success`, `Warning`, `Danger`, `Info`.
- `Chart`: an array of all 8 colors.
- Sizes in DIPs, numbers: `ControlHeight`, `RowHeight`, `HeaderHeight`, `SettingsRowHeight`, `TabHeight`, `MenuBarStripHeight`, `MenuBarItemHeight`, `RadiusControl`, `RadiusCard`, `ToolbarPadTop`, `ToolbarPadBottom`, `ToolbarGap`, `ToolbarGroupGap`, `Icon`, `ToolbarIcon`, `CheckSize`, `SwitchWidth`, `SwitchHeight`, `FontSize`.
- Type roles, objects of `Size` (number) and `Weight` (integer, 100 to 900): `Caption`, `Label`, `Heading`, `Title`.
- `Font`: a string, the family of text; `""` is the system's. A font of the app's own needs `ui.RegisterFont` in Go.
- Shadows, objects of `Y`, `Blur`, `Spread` and `Color`: `CardShadow`, `PopupShadow`.
- `FocusRingWidth`, `FocusRingOffset`, `ScrollbarWidth`: numbers. `ScrollbarStyle`: an object of the fields of `ui.ScrollbarStyle`.

The sizes are the standard ones: `density` applies on top, when the skin is used. A key that is not a field is an error.

## Looks

`looks` names kinds: `Button`, `ToolButton`, `Field`, `ComboBox`, `CheckBox`, `RadioButton`, `GroupBox`, `Card`, `TabStrip`, `TabHeader`, `GridHeader`, `ToolStrip`, `StatusStrip`, `MenuBar`, `HeaderBar`. Each value is a string of `tw` classes. A kind named in the file replaces the base's look of that kind whole; a kind left out keeps the base's. A class `tw` does not know is an error.

## Errors

An error is `path: message (line N)`, as `light.Background: want a color #rrggbb or #rrggbbaa, got "red" (line 6)`. Parse does not check contrast: call `Check` on the result.

## Examples

A file that changes little:

```json
{
  "version": 1,
  "name": "Teal Calm",
  "base": "Calm",
  "palette": "Teal"
}
```

A brand: an accent, a tighter density, a warm ground and a squarer button.

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

Every kind of value: a type role, a shadow, the chart, the scroll bars.

```json
{
  "version": 1,
  "name": "Wide",
  "base": "Classic",
  "light": {
    "RowHeight": 28,
    "Heading": { "Size": 18, "Weight": 600 },
    "CardShadow": { "Y": 2, "Blur": 8, "Spread": 0, "Color": "#00000026" },
    "Chart": ["#1c71d8", "#26a269", "#e5a50a", "#c01c28", "#813d9c", "#2190a4", "#865e3c", "#5e5c64"],
    "ScrollbarStyle": { "ThumbWidth": 8, "Square": true }
  },
  "looks": { "GridHeader": "bg-chrome border-r border-strong" }
}
```
