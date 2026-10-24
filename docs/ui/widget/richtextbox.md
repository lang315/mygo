# RichTextBox

`richtextbox.New()` is a box of styled text: bold, italic, underline, colors,
links, headings, lists, undo, over [`ui/richtext`](../rich-text-editor.md). It is
a package of its own, `ui/widget/richtextbox`, so that only the apps that use
it link the rich text editor (a window of labels and buttons weighs the same
with or without it).

```go
import "github.com/egoist/mygo/ui/widget/richtextbox"

var notes *richtextbox.Box
notes = richtextbox.New().
	SetMarkup("Hello <b>world</b>").
	SetLabel("Notes").
	OnChange(func() { save(notes.Markdown()) })
bold := widget.Button("B").OnClick(func() { notes.Edit(func(d *richtext.Document) { d.ToggleBold() }) })
```

A `*Box` is a control: put it in a box or a form like any other.

## Text

- `SetText(s)` and `Text()`: plain characters. `SetRich(t)` and `Rich()`: a
  `richtext.Text`. `SetMarkdown(s)` and `Markdown()`. A setter replaces the
  text, forgets the history and formats, and does not run `OnChange`.
- `SetMarkup(s)` and `Markup()`: the subset of Pango markup that
  [`widget.ParseMarkup`](../widget.md) reads: `<b>`, `<i>`, `<u>` and
  `<span foreground background weight>`. Text of that subset round-trips.
  `Markup` drops what the subset cannot say (links, headings, lists, fonts,
  sizes, code, strikethrough) and keeps their text, writes `#rrggbbaa` for a
  translucent color, and normalises: `<i><b>` comes back as `<b><i>`, weight
  `normal` as `400`. Two texts of the subset are equal after a reparse, not
  byte for byte.
- `Edit(fn)` runs `fn` on the `*richtext.Document` (`d.ToggleBold()`,
  `d.Undo()`, `d.SetHeading(1)`) and redraws the windows that show the box, from
  any goroutine, so the change appears and `OnChange` runs. `fn` runs with the
  package's lock held: it must not call widget getters or setters.
  `Document()` returns the document for reading (`Formatting`, `CanUndo`); a
  change made on it directly is safe but is not drawn until some other frame
  comes, as richtext says: use `Edit`.
- `SetReadOnly(true)`, `SetPlaceholder(s)`, `SetLabel(s)` (the accessible name,
  "Rich text" by default), `OnLink(fn)`.

## Selection

`Selection()` and `SetSelection(start, end)` count **runes**, newlines
included, as richtext does (not bytes). `SetSelection` never splits a
grapheme: a start inside one moves back to its beginning, an end inside one
forward to its end, and a caret (start equal to end) inside one goes before it, so "ǎ" or "à" typed as a letter and a combining mark is
selected whole, in Vietnamese precomposed or combining forms alike; offsets
beyond the text are clamped. `SelectedText()` returns what is selected.
`OnSelectionChange(fn)` runs after the selection moves, by the user or
`SetSelection`, but not for the move of the caret a text setter makes.

## OnChange

`OnChange(fn)` runs on the event goroutine once for each frame in which the
text or its formats changed (not once per edit: two edits before a frame are
one report), other than by the four text setters, which do not run it, nor
swallow a report that is due for an edit made before them: typing, paste, an
input method's commit (Telex or Unikey's backspace and replacement arrive as
one edit), undo and redo, and what a toolbar does through `Edit`, such
as `ToggleBold`.

Not here: images and attachments, tags such as mentions, a theme of your own,
and binding (a rich text is not one comparable value). The box shows no
placeholder in a test: `Placeholder` is drawn, not a node.
