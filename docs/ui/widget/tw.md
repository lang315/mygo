# Tailwind classes

A cheat sheet of `ui/widget/tw`: the classes `tw.S` and `tw.Parse` turn into a
[`widget.Style`](../widget.md#tailwind-classes). Values are Tailwind v3's. An
app that does not import `tw` pays nothing for it.

```go
import "github.com/egoist/mygo/ui/widget/tw"

save := widget.Button("Save").Apply(tw.S("px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover transition"))
```

`tw.S` parses a string once and caches it; a class it does not know is logged
once and ignored, and the others still apply. `tw.Parse` returns an `*tw.Error`
naming every bad class instead. `mygo vet` reports the bad classes of a literal
string. On the same property the **later class wins** (Tailwind decides by its
CSS order instead).

The tables below are made from the parser's own tables by
`go test ./internal/twclass -run TestCheatSheet -update`, and the test fails
when the page and the parser disagree: what is listed here is what parses.
Every example in the `tw` blocks is parsed by the same test, and every class in
a `tw-bad` block must be refused for the reason it gives.

## Classes with a number

`{n}` is a value of the scale at the end of this section. A class with a value
that is not in its list (`p-13`, `w-[200px]`) is an unknown class: there are no
arbitrary values; use the `Style` method in Go. Padding, margin and radius are in
theme units, so `p-[13px]` is `Padding(3.25)` with the default 4 DIP unit; a
width is in DIPs: `Width(200)`.

<!-- twclass:boxes -->
| Class | Sets | Value |
|---|---|---|
| `p-{n}` | padding of all four | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `px-{n}` | padding of right + left | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `py-{n}` | padding of top + bottom | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `pt-{n}` | padding of top | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `pr-{n}` | padding of right | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `pb-{n}` | padding of bottom | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `pl-{n}` | padding of left | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default) |
| `m-{n}` `-m-{n}` | margin of all four | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `mx-{n}` `-mx-{n}` | margin of right + left | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `my-{n}` `-my-{n}` | margin of top + bottom | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `mt-{n}` `-mt-{n}` | margin of top | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `mr-{n}` `-mr-{n}` | margin of right | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `mb-{n}` `-mb-{n}` | margin of bottom | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `ml-{n}` `-ml-{n}` | margin of left | `n` theme units (`Theme.Space(n)`, 4 DIPs each by default); negative with the `-` form |
| `w-{n}` | width | `n` × 4 DIPs (fixed, not theme units) |
| `h-{n}` | height | `n` × 4 DIPs (fixed, not theme units) |
| `min-w-{n}` | minimum width | `n` × 4 DIPs (fixed, not theme units) |
| `max-w-{n}` | maximum width | `n` × 4 DIPs (fixed, not theme units) |
| `min-h-{n}` | minimum height | `n` × 4 DIPs (fixed, not theme units) |
| `max-h-{n}` | maximum height | `n` × 4 DIPs (fixed, not theme units) |
| `max-w-none` | no maximum width (a `w-` is no longer capped at its container's room) | |
<!-- /twclass:boxes -->

<!-- twclass:spacing -->
`n` is one of `0`, `px`, `0.5` … `4` (step 0.5), `5` … `12` (step 1), `14`, `16` … `64` (step 4), `72`, `80`, `96`.

`px` is a quarter of a unit. In padding and margin a unit is `Theme.Space(1)`, 4 DIPs by default; a width or a height is `n` × 4 DIPs whatever the theme.
<!-- /twclass:spacing -->

## Radius

<!-- twclass:radius -->
| Class | Radius (a theme unit is 4 DIPs by default) |
|---|---|
| `rounded-none` | 0 theme units |
| `rounded-sm` | 0.5 theme units |
| `rounded` | 1 theme unit |
| `rounded-md` | 1.5 theme units |
| `rounded-lg` | 2 theme units |
| `rounded-xl` | 3 theme units |
| `rounded-2xl` | 4 theme units |
| `rounded-3xl` | 6 theme units |
| `rounded-full` | a full pill or circle |

Before the size, a side or a corner: `b` (bottom right + bottom left), `bl` (bottom left), `br` (bottom right), `l` (top left + bottom left), `r` (top right + bottom right), `t` (top left + top right), `tl` (top left), `tr` (top right). `rounded-l`, `rounded-tr-2xl` and `rounded-l-none` all parse.
<!-- /twclass:radius -->

## Border

<!-- twclass:border -->
| Class | Width |
|---|---|
| `border-0` | 0 DIP |
| `border` | 1 DIP |
| `border-2` | 2 DIP |
| `border-4` | 4 DIP |
| `border-8` | 8 DIP |

One side or two: `b` (bottom), `l` (left), `r` (right), `t` (top), `x` (right + left), `y` (top + bottom), as `border-t`, `border-x-2`, `border-b-0`. `border-solid` and `border-dashed` set the style. A color is `border-{color}` (see Colors).
<!-- /twclass:border -->

## Text

<!-- twclass:text -->
| Class | Size |
|---|---|
| `text-xs` | `widget.Size` step 1 (0 is the theme's base size) |
| `text-sm` | `widget.Size` step 2 (0 is the theme's base size) |
| `text-base` | `widget.Size` step 0 (0 is the theme's base size) |
| `text-lg` | `widget.Size` step 3 (0 is the theme's base size) |
| `text-xl` | `widget.Size` step 4 (0 is the theme's base size) |
| `text-2xl` | `widget.Size` step 5 (0 is the theme's base size) |
| `text-3xl` | `widget.Size` step 6 (0 is the theme's base size) |
| `text-4xl` | 36 DIPs |
| `text-5xl` | 48 DIPs |
| `text-6xl` | 60 DIPs |
| `text-7xl` | 72 DIPs |
| `text-8xl` | 96 DIPs |
| `text-9xl` | 128 DIPs |

| Class | Weight |
|---|---|
| `font-thin` | 100 |
| `font-extralight` | 200 |
| `font-light` | 300 |
| `font-normal` | 400 |
| `font-medium` | 500 |
| `font-semibold` | 600 |
| `font-bold` | 700 |
| `font-extrabold` | 800 |
| `font-black` | 900 |
<!-- /twclass:text -->

## Colors

<!-- twclass:palette -->
Palette (Tailwind v3.4): `{color}-{shade}` after `bg-`, `text-` or `border-`.

Hues: `amber` `blue` `cyan` `emerald` `fuchsia` `gray` `green` `indigo` `lime` `neutral` `orange` `pink` `purple` `red` `rose` `sky` `slate` `stone` `teal` `violet` `yellow` `zinc`

Shades: `50` `100` `200` `300` `400` `500` `600` `700` `800` `900` `950`

Also `black`, `white`, `transparent`, and `default`, the theme's color for the property (`bg-default` is the background, `text-default` the text, `border-default` the border).

Palette colors are the same in the light and the dark theme.
<!-- /twclass:palette -->

### Theme colors

After `bg-`, `text-` or `border-`:

<!-- twclass:roles -->
| Class suffix | Theme role |
|---|---|
| `bg-accent`, `text-accent`, `border-accent` | `accent` |
| `bg-accent-hover`, `text-accent-hover`, `border-accent-hover` | `accent-hover` |
| `bg-accent-pressed`, `text-accent-pressed`, `border-accent-pressed` | `accent-pressed` |
| `bg-danger`, `text-danger`, `border-danger` | `danger` |
| `bg-focus`, `text-focus`, `border-focus` | `focus` |
| `bg-inverse`, `text-inverse`, `border-inverse` | `inverse` |
| `bg-muted`, `text-muted`, `border-muted` | `text-muted` |
| `bg-on-accent`, `text-on-accent`, `border-on-accent` | `accent-text` |
| `bg-on-inverse`, `text-on-inverse`, `border-on-inverse` | `inverse-text` |
| `bg-selection`, `text-selection`, `border-selection` | `selection` |
| `bg-success`, `text-success`, `border-success` | `success` |
| `bg-surface`, `text-surface`, `border-surface` | `surface` |
| `bg-surface-hover`, `text-surface-hover`, `border-surface-hover` | `surface-hover` |
| `bg-surface-pressed`, `text-surface-pressed`, `border-surface-pressed` | `surface-pressed` |
| `bg-warning`, `text-warning`, `border-warning` | `warning` |

The roles are `background` `surface` `surface-hover` `surface-pressed` `border` `text` `text-muted` `accent` `accent-hover` `accent-pressed` `accent-text` `danger` `warning` `success` `selection` `focus` `inverse` `inverse-text`. Theme colors follow the theme, light and dark.
<!-- /twclass:roles -->

## Effects

<!-- twclass:effects -->
| Class | Layers (x, y, blur, spread, alpha) |
|---|---|
| `shadow` | (0, 1, 3, 0, 0.1) + (0, 1, 2, -1, 0.1) |
| `shadow-2xl` | (0, 25, 50, -12, 0.25) |
| `shadow-inner` | (0, 2, 4, 0, 0.05) inset |
| `shadow-lg` | (0, 10, 15, -3, 0.1) + (0, 4, 6, -4, 0.1) |
| `shadow-md` | (0, 4, 6, -1, 0.1) + (0, 2, 4, -2, 0.1) |
| `shadow-none` | none: clears the shadows |
| `shadow-sm` | (0, 1, 2, 0, 0.05) |
| `shadow-xl` | (0, 20, 25, -5, 0.1) + (0, 8, 10, -6, 0.1) |

`opacity-{n}`, `n` one of: `0` … `100` (step 5), in percent.

`cursor-{name}`, `name` one of: `crosshair` `default` `grab` `grabbing` `move` `not-allowed` `pointer` `text`.
<!-- /twclass:effects -->

## Layout, text style and the rest

The `halign-`, `valign-`, `hexpand` and `vexpand` classes are gota's; they set
how a control takes its place in the box that holds it.

<!-- twclass:fixed -->
| Class | Sets |
|---|---|
| `font-mono` | the monospace font |
| `halign-center` | places the control horizontally in a `VBox`: center (layout: an error in a variant) |
| `halign-end` | places the control horizontally in a `VBox`: end (layout: an error in a variant) |
| `halign-fill` | places the control horizontally in a `VBox`: fill (layout: an error in a variant) |
| `halign-start` | places the control horizontally in a `VBox`: start (layout: an error in a variant) |
| `hexpand` | takes the free width of its `HBox` (layout: an error in a variant) |
| `italic` | italic text |
| `not-italic` | not italic (clears an `italic` earlier in the string; an error in a variant) |
| `valign-center` | places the control vertically in an `HBox`: center (layout: an error in a variant) |
| `valign-end` | places the control vertically in an `HBox`: end (layout: an error in a variant) |
| `valign-fill` | places the control vertically in an `HBox`: fill (layout: an error in a variant) |
| `valign-start` | places the control vertically in an `HBox`: start (layout: an error in a variant) |
| `vexpand` | takes the free height of its `VBox` (layout: an error in a variant) |
<!-- /twclass:fixed -->

## gota's button classes

<!-- twclass:buttons -->
| Class | Same as |
|---|---|
| `destructive-action` | `bg-danger text-on-accent hover:opacity-90 active:opacity-80` |
| `suggested-action` | `bg-accent text-on-accent hover:bg-accent-hover active:bg-accent-pressed` |
<!-- /twclass:buttons -->

## Motion

<!-- twclass:motion -->
| Class | Duration |
|---|---|
| `transition` | 150 ms unless a `duration-` class is given |
| `duration-75` | 75 ms |
| `duration-100` | 100 ms |
| `duration-150` | 150 ms |
| `duration-200` | 200 ms |
| `duration-300` | 300 ms |
| `duration-500` | 500 ms |
| `duration-700` | 700 ms |
| `duration-1000` | 1000 ms |
<!-- /twclass:motion -->

## Variants

`hover:`, `active:` (pressed), `focus:`, `disabled:` and `dark:`. `dark:` stacks
on one other (`dark:hover:bg-slate-700`); two states do not stack. Theme colors
follow the theme; palette colors are the same in both.

```tw
hover:bg-accent-hover
active:bg-accent-pressed
focus:border-accent
disabled:opacity-50
dark:bg-slate-800
dark:hover:bg-slate-700
```

A variant sets a look. Layout (`halign-*`, `valign-*`, `hexpand`, `vexpand`),
`transition` and `duration-*` belong to the control and are refused in one, and
so is `not-italic` (a variant cannot clear what the base sets):

```tw-bad
hover:hexpand | layout in a variant
focus:halign-center | layout in a variant
hover:transition | transition in a variant
hover:duration-300 | transition in a variant
dark:not-italic | not-italic in a variant
hover:focus:p-2 | unsupported variant stack
dark:dark:p-2 | unsupported variant stack
```

A class with no value, or with a prefix tw does not know:

```tw-bad
p- | unknown class
rounded- | unknown class
sm:p-2 | unknown class
group-hover:bg-accent | unknown class
```

## What each class sets

<!-- twclass:styles -->
| What a class sets | `widget.Style` method |
|---|---|
| Padding | `PaddingTop … PaddingLeft` |
| Margin | `MarginTop … MarginLeft` |
| Radius | `RadiusTopLeft … RadiusBottomLeft` |
| BorderWidth | `BorderTop … BorderLeft` |
| BorderStyle | `BorderStyle` |
| Width | `Width` |
| Height | `Height` |
| MinWidth | `MinWidth` |
| MaxWidth | `MaxWidth` |
| MinHeight | `MinHeight` |
| MaxHeight | `MaxHeight` |
| Background | `Background` |
| TextColor | `TextColor` |
| BorderColor | `BorderColor` |
| TextSize | `TextSize` |
| FontSize | `FontSize` |
| FontWeight | `FontWeight` |
| Italic | `Italic` |
| Mono | `Mono` |
| Opacity | `Opacity` |
| Cursor | `Cursor` |
| Shadows | `Shadow, InsetShadow, NoShadow` |
| Align | `Align` |
| VAlign | `VAlign` |
| HExpand | `HExpand` |
| VExpand | `VExpand` |
| Transition | `Transition` |
<!-- /twclass:styles -->

## To copy

```tw
px-3 py-2 rounded-lg bg-accent text-on-accent hover:bg-accent-hover active:bg-accent-pressed transition
px-3 py-2 rounded-lg border border-default bg-surface hover:bg-surface-hover disabled:opacity-50
p-4 rounded-xl border border-default bg-surface shadow-sm
text-sm text-muted
font-semibold text-lg
w-64 min-w-32 max-w-96
-mt-1 mb-2
border-b border-default
rounded-l-none border-l-0
dark:bg-slate-800 dark:text-slate-100 bg-white text-slate-900
destructive-action
suggested-action rounded-lg px-4
halign-start valign-center hexpand
cursor-pointer
opacity-60 italic
font-mono text-xs
```

## From the web: what Tailwind has and tw does not

A web screen is laid out by CSS; a mygo screen by containers and a few
properties of the control. These classes are refused (`tw.S` logs them, `mygo
vet` reports them), and this is what to do instead.

Alignment and growth depend on the box that holds the control: in a `VBox`,
`halign-*` places it across (horizontally) and `vexpand` gives it the free height;
in an `HBox`, `valign-*` places it across (vertically) and `hexpand` gives it the
free width. The other axis of each does nothing.

| Tailwind | Why not | In mygo |
|---|---|---|
| `flex`, `flex-row`, `flex-col` | there is no `display`; the container decides | `widget.HBox(gap)`, `widget.VBox(gap)` |
| `gap-N`, `space-x-N`, `space-y-N` | spacing is the container's | the argument of `HBox`, `VBox` and `FlowPanel`, in DIPs; `Grid(colGap, rowGap)`; `Fields().Gap(row, col)` |
| `grid`, `grid-cols-N`, `col-span-N` | no CSS grid | `widget.Grid(colGap, rowGap)` and `.Attach(w, col, row, width, height)`; `widget.Fields()` for rows of a label and a control |
| `items-*`, `self-*` | alignment is set on the child | across the box: `halign-{start,center,end,fill}` in a `VBox`, `valign-*` in an `HBox` |
| `justify-*`, `place-*` | no class: the box places its items as they are packed | `box.Pack(w, expand, fill)` and `PackEnd`; `hexpand` / `vexpand` take the free room |
| `m-auto`, `mx-auto` | no `auto` margin | `halign-center` in a `VBox` (`valign-center` in an `HBox`) |
| `flex-1`, `grow` | | `hexpand` in an `HBox`, `vexpand` in a `VBox` (`Expand()` in Go) |
| `w-full` | no percentages | `halign-fill` in a `VBox`; `h-full`: `valign-fill` in an `HBox`; along the box, `hexpand` / `vexpand` |
| `w-1/2`, `w-screen` | no fractions or viewport units | `hexpand` on controls that share a box; `w-{n}` for a fixed width |
| `overflow-auto`, `overflow-scroll` | scrolling is a control | `widget.ScrollPanel(w)` |
| `hidden`, `block` | | `SetVisible(false)` / `SetVisible(true)`: a hidden control takes no room |
| `invisible` | a hidden control gives up its room | `opacity-0` hides it and keeps the room; it still takes clicks and focus |
| `truncate` | `Style` has no text overflow | list controls (ListBox, ListView, DataGridView) keep a line each; for a `Label`, `widget.Custom` with `ui.Text(c, s).SingleLine()` (cut with an ellipsis) |
| `text-left`, `text-center`, `text-right` | `halign-*` moves the control, not the lines of its text; `Style` has no text alignment | `widget.Custom` with `ui.Text(c, s).TextAlign(ui.Center)` |
| `absolute`, `relative`, `top-N`, `z-N` | no positioning | `widget.Custom` and `ui` for what overlaps |
| `bg-blue-600/50`, `text-black/70` | no color alpha | `opacity-N` on the whole control; `Style.Background` with a color that has alpha in Go |
| `bg-gradient-to-r`, `from-*`, `to-*` | | `Style.Gradient` in Go |
| `blur-*` | | `Style.Blur(sigma)` in Go: it blurs the control and its children |
| `backdrop-blur-*` | `Style.Blur` blurs the control itself; nothing blurs what is behind it | none |
| `p-[13px]`, `w-[200px]`, `text-[#123456]` | no arbitrary values | the `Style` method in Go: `Padding(3.25)` (theme units), `Width(200)` (DIPs), `TextColor(widget.Fixed(ui.RGB(18, 52, 86)))` |
| `ring`, `ring-N`, `outline-*` | the focus ring is the control's | drawn by the control in the theme's `focus` color |
| `sm:`, `md:`, `lg:`, `group-hover:`, `first:` | no breakpoints or relations between elements | `hover:`, `active:`, `focus:`, `disabled:`, `dark:` only |
| `transition-all`, `ease-*`, `delay-*` | one transition, of the background, border and text colors | `transition` with `duration-N` |
| `underline`, `line-through` | text decoration is not in `Style` | `Label.Rich(ui.Span{Text: s, Underline: true})`, `Strikethrough: true` |
| `tracking-*` | | `ui.Span{LetterSpacing: …}`, or `Element.LetterSpacing` in `widget.Custom` |
| `leading-*` | | `Element.LineHeight(multiple)` in `widget.Custom` |
| `uppercase`, `lowercase`, `capitalize` | no text transform | `strings.ToUpper(s)` and the like |

```tw-bad
flex | unknown class
flex-col | unknown class
gap-2 | unknown class
space-x-2 | unknown class
grid | unknown class
grid-cols-3 | unknown class
items-center | unknown class
self-end | unknown class
justify-between | unknown class
place-items-center | unknown class
m-auto | unknown class
mx-auto | unknown class
flex-1 | unknown class
grow | unknown class
w-full | unknown class
h-full | unknown class
w-1/2 | unknown class
w-screen | unknown class
overflow-auto | unknown class
hidden | unknown class
invisible | unknown class
truncate | unknown class
text-center | unknown class
absolute | unknown class
bg-blue-600/50 | unknown class
bg-gradient-to-r | unknown class
blur-sm | unknown class
backdrop-blur-sm | unknown class
p-[13px] | unknown class
ring | unknown class
transition-all | unknown class
underline | unknown class
line-through | unknown class
tracking-wide | unknown class
leading-tight | unknown class
uppercase | unknown class
```

## From gota

gota's classes are Tailwind v2's, whose palette names differ from v3's: v2
`green`, `yellow` and `purple` are v3 `emerald`, `amber` and `violet` (the same
colors, renamed). A class that translates by name parses but gives another color.

| gota | In mygo |
|---|---|
| `halign-*`, `valign-*`, `hexpand`, `vexpand` | the same classes |
| `suggested-action`, `destructive-action` | the same classes |
| `text-green-N`, `bg-green-N`, `border-green-N` | the same with `emerald-N` |
| `text-yellow-N`, `bg-yellow-N`, `border-yellow-N` | the same with `amber-N` |
| `text-purple-N`, `bg-purple-N`, `border-purple-N` | the same with `violet-N` |
| other palette classes (`red`, `blue`, `indigo`, `pink`, `gray`, …) | the same names and shades |
| `text-italic` | has no effect in gota's own stylesheet (it defines `.italic` only): leave it out, and use `italic` only if you want the text slanted |
| `combobox-rounded-r-none` | `rounded-r-none` |
| `text-green`, a color with no shade | has no effect in gota's own stylesheet: leave it out (do not guess a shade) |
| `gota-card`, `gota-table`, `gota-tag-select*` | not classes: widget controls |

```tw
text-emerald-600
bg-amber-500
border-violet-500
```

```tw-bad
text-italic | unknown class
text-green | unknown class
combobox-rounded-r-none | unknown class
```
