package widget

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// FieldsWidget lays out a form: rows of a label and the control it names.
type FieldsWidget struct {
	common[FieldsWidget]
	rows           []fieldRow
	heads          []fieldHead
	stacked        bool
	settings       bool
	rowGap, colGap float32
	gapSet         bool
	tracks         []ui.Track // a card's row tracks, kept between frames: RowTracks copies them
}

type fieldRow struct {
	label *LabelWidget
	w     Control
	key   rune // the label's mnemonic: Alt+key focuses w
	at    int  // its place in the label, in runes
	rule  any  // fieldRule of the label's id, boxed once: the key of the divider above it
	box   any  // fieldRowKey of the label's id, boxed once: the key of a stacked row
	desc  string
	wide  bool // the control under the label, across the form (Wide)
	// action ends the label's line of a wide row, or follows the control (Action).
	action Control
	// said is the control's description, desc then the tooltip saidTip,
	// "" until describe joins them.
	said, saidTip string
	dimmed        bool // describe disabled the label, with the control
}

// fieldHead is a section's heading, shown before row at.
type fieldHead struct {
	label *LabelWidget
	at    int
}

// fieldRowKey keys the box of a stacked row, and fieldHole the cell of a
// two-column row whose label is hidden, by the label's id.
type (
	fieldRowKey int
	fieldHole   int
	fieldRule   int // the divider above a settings row, by the label's id
	fieldBreak  int // the gap of a Section("") before row i
)

// Fields makes a form with no rows. Row adds them; by default the labels
// make a column as wide as the widest of them, the controls fill a second
// column, and each label sits at the middle of its control's height, as
// WinForms and GTK's grids lay out a form.
func Fields() *FieldsWidget {
	f := &FieldsWidget{}
	f.init(f)
	return f
}

// Row adds a row after the others: a label showing label, and c. A row
// without a control, or with one already in the form, is not added. The
// label is a Label of its own: Label(i) returns it to style it.
func (f *FieldsWidget) Row(label string, c Control) *FieldsWidget {
	return f.addRow(label, c, false)
}

// Add adds a wide row with no label: c across the form (in SettingsRows,
// across the card's inside), as Row("", c).Wide() does. Describe and Action
// then apply to it as to any row.
func (f *FieldsWidget) Add(c Control) *FieldsWidget { return f.addRow("", c, true) }

// addRow is Row, the row made wide (Wide) when wide is true, in one change.
func (f *FieldsWidget) addRow(label string, c Control, wide bool) *FieldsWidget {
	if c == nil {
		return f
	}
	display, key, at := parseMnemonic(label)
	l := Label(display)
	f.change(func() {
		if f.has(c, -1) {
			return
		}
		f.rows = append(f.rows, fieldRow{label: l, w: c, key: key, at: at, rule: fieldRule(l.id), box: fieldRowKey(l.id), wide: wide})
		adopt(&f.core, c)
	})
	return f
}

// Section adds a heading, in bold, across the form's width before the rows
// that follow it, up to the next Section. The rows of all sections share one
// label column, and a section after the first has more room above it than a
// row has. A section whose rows are all hidden hides its heading and its room.
// In SettingsRows each section is a card, the heading above it, and each card
// has its own columns. An empty title makes a group break: the room a section
// has above it and no heading, so no node for assistive technology; there is
// none first, nor where the section's rows are all hidden. In SettingsRows it
// starts a card.
func (f *FieldsWidget) Section(title string) *FieldsWidget {
	l := Label(title)
	l.role = &headingText
	f.change(func() { f.heads = append(f.heads, fieldHead{l, len(f.rows)}) })
	return f
}

// headings returns the headings to show before row i: those of the sections
// that start there and have a row that shows. Under mu.
func (f *FieldsWidget) headings(i int) (ls []*LabelWidget) {
	for h, head := range f.heads {
		if head.at != i {
			continue
		}
		end := len(f.rows)
		if h+1 < len(f.heads) {
			end = f.heads[h+1].at
		}
		for _, r := range f.rows[i:end] {
			if !r.w.base().hidden {
				ls = append(ls, head.label)
				break
			}
		}
	}
	return ls
}

// Describe sets the description of the last row added: a line in TextMuted
// under its label, which wraps, read as its control's description. "" removes
// it. Before any row it does nothing.
func (f *FieldsWidget) Describe(text string) *FieldsWidget {
	f.change(func() {
		if n := len(f.rows); n > 0 {
			f.rows[n-1].desc, f.rows[n-1].said = text, ""
		}
	})
	return f
}

// Wide makes the last row added a wide one: its label, and its description,
// across the form's width, and its control under them, as wide as the form
// (in SettingsRows, as the card's inside) unless its Align says otherwise.
// A wide row with an empty label has no label. Stacked rows are all wide.
// Before any row it does nothing.
func (f *FieldsWidget) Wide() *FieldsWidget {
	f.change(func() {
		if n := len(f.rows); n > 0 {
			f.rows[n-1].wide = true
		}
	})
	return f
}

// Action gives the last row added an action control, such as a LinkLabel:
// at the right end of its label's line when the row is wide (Wide, Stacked),
// after its control otherwise. A second call replaces it; nil removes it.
// A control already in the form, but this row's own action, is not taken.
// Before any row it does nothing.
func (f *FieldsWidget) Action(c Control) *FieldsWidget {
	f.change(func() {
		if n := len(f.rows); n > 0 && (c == nil || !f.has(c, n-1)) {
			f.rows[n-1].action = c
		}
	})
	return f
}

// has reports whether c is a control or an action of a row, but the action
// of row skip. Under mu.
func (f *FieldsWidget) has(c Control, skip int) bool {
	for i, r := range f.rows {
		if r.w == c || i != skip && r.action == c {
			return true
		}
	}
	return false
}

// describe gives the control its row's description, its tooltip after it,
// joined once each time either changes, so that a frame builds no string.
// Under mu, before the control builds.
func (r *fieldRow) describe() {
	b := r.w.base()
	// The label dims with its control, and comes back with it, unless
	// it was disabled by hand.
	if b.disabled && !r.label.disabled {
		r.label.disabled, r.dimmed = true, true
	} else if !b.disabled && r.dimmed {
		r.label.disabled, r.dimmed = false, false
	}
	if r.desc == "" {
		b.desc = ""
		return
	}
	tip := b.tooltip
	if b.superTip != nil {
		tip = b.superTip.said // the SuperTip takes the tooltip's place
	}
	if r.said == "" || r.saidTip != tip {
		r.said, r.saidTip = r.desc, tip
		if tip != "" {
			// A description that ends a sentence takes a space, others ". ".
			sep := ". "
			if last, _ := utf8.DecodeLastRuneInString(r.desc); strings.ContainsRune(".!?…:", last) {
				sep = " "
			}
			r.said += sep + tip
		}
	}
	b.desc = r.said
}

// Stacked puts each label above its control, over the form's width, as a
// VBox of a label and a control does, instead of in a column of their own.
func (f *FieldsWidget) Stacked() *FieldsWidget {
	f.change(func() { f.stacked, f.settings = true, false })
	return f
}

// SettingsRows lays the form out as a page of settings: each section is a
// card (the skin's SettingsCard look, else its Card look), its heading above
// it, and rows before the first Section make a card with no heading. In a
// card each label takes the width the control leaves and wraps, the control
// sits at the right edge in its own size (a control's Align overrides), and a
// divider separates one row from the next. A hidden control takes its label
// and its divider away, and a section whose rows are all hidden its card. Of
// Stacked and SettingsRows, the last call wins. A ProgressBar and an empty
// ListBox have no width in the fit column, and a ListBox with items or a
// table a narrow one: give them SetSize, or make their row Wide. The label column is
// no narrower than its longest word, so in a narrow window the controls are
// pushed toward or past the right edge (a ComboBox beside "Notifications"
// needs about 229 DIPs): give the window a SetMinimumSize wide enough. In a
// ScrollPanel use Vertical, or the form gets no width to wrap at.
func (f *FieldsWidget) SettingsRows() *FieldsWidget {
	f.change(func() { f.settings, f.stacked, buildCards = true, false, (*FieldsWidget).cards })
	return f
}

// buildCards builds a form of SettingsRows (cards), set by SettingsRows: an
// app that lays out no settings page does not link the cards. Under mu.
var buildCards func(f *FieldsWidget, c *ui.Context, rowGap, colGap, sectionGap float32) ui.Element

// Gap sets the space between rows, and between the label column and the
// control column, in DIPs; stacked, between a label and its control. The
// default is the theme's: 2 units between rows and 3 between the columns,
// or 1 between a stacked label and its control. In SettingsRows the row gap
// applies on each side of a divider.
func (f *FieldsWidget) Gap(row, col int) *FieldsWidget {
	f.change(func() { f.rowGap, f.colGap, f.gapSet = float32(max(row, 0)), float32(max(col, 0)), true })
	return f
}

// Label returns the label of row i, from 0, or nil if there is no such row.
func (f *FieldsWidget) Label(i int) *LabelWidget {
	mu.Lock()
	defer mu.Unlock()
	if i < 0 || i >= len(f.rows) {
		return nil
	}
	return f.rows[i].label
}

func (f *FieldsWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	rowGap, colGap := t.Space(2), t.Space(3)
	if f.stacked {
		colGap = t.Space(1)
	}
	if f.gapSet {
		rowGap, colGap = f.rowGap, f.colGap
	}
	// A section after the first starts further down than a row would.
	sectionGap := t.Space(4)
	// A Section("") is a spacer in a row of its own, between two row gaps.
	breakGap := max(sectionGap-rowGap, 0)
	if f.stacked {
		return ui.Column(c).Gap(rowGap).Children(func() {
			// The rows are a column, whatever the box around the form.
			outer := axis
			axis = vertical
			defer func() { axis = outer }()
			shown := false
			for i, r := range f.rows {
				titled, brk := false, false
				for _, h := range f.headings(i) {
					if h.text == "" {
						brk = true
						continue
					}
					titled = true
					e := render(c, h).Role(ui.RoleHeading)
					if shown {
						e.Margin(sectionGap, 0, 0, 0)
					}
				}
				if brk && !titled && shown && breakGap > 0 {
					ui.Box(c.Key(fieldBreak(i))).Height(breakGap)
				}
				if r.w.base().hidden { // the row goes, its label with it
					f.hold(r)
					continue
				}
				shown = true
				f.rows[i].describe()
				ui.Column(c.Key(r.box)).Gap(colGap).Children(func() {
					r.underline(c)
					e := f.labelLine(c, &r, func() ui.Element {
						e := render(c, r.label) // holds a hidden label
						if !r.label.hidden && r.label.place.align == 0 {
							e.AlignSelf(ui.Start)
						}
						return e
					})
					if r.key != 0 {
						w := r.w // not r, which the closure would move to the heap
						registerMnemonic(r.key, e, w, func() { buildingForm.focusRow(w) })
					}
					if r.desc != "" && !r.label.hidden {
						ui.Text(c, r.desc).TextColor(c.Theme().TextMuted).Disabled(r.w.base().disabled)
					}
					render(c, r.w).NamedBy(e)
				})
			}
		})
	}
	if f.settings {
		return buildCards(f, c, rowGap, colGap, sectionGap)
	}
	return ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(colGap).GapY(rowGap).Children(func() {
		// A cell's Align and VAlign are across and along the cell, not a
		// box's axes: build the cells as for a row, whose cross axis is vertical.
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		shown := false // a row is up
		for i, r := range f.rows {
			titled, brk := false, false
			for _, h := range f.headings(i) {
				if h.text == "" {
					brk = true
					continue
				}
				titled = true
				e := render(c, h).Role(ui.RoleHeading).ColumnSpan(2)
				if shown {
					e.Margin(sectionGap, 0, 0, 0)
				}
			}
			if brk && !titled && shown && breakGap > 0 {
				ui.Box(c.Key(fieldBreak(i))).ColumnSpan(2).Height(breakGap)
			}
			if r.w.base().hidden {
				f.hold(r)
				continue
			}
			shown = true
			f.cells(c, i)
		}
	})
}

// cards builds SettingsRows: the rows of each section in a card, its title
// above it, outside the card. A section whose rows are all hidden has
// neither. Under mu.
func (f *FieldsWidget) cards(c *ui.Context, rowGap, colGap, sectionGap float32) ui.Element {
	t := c.Theme()
	// With a skin, a row is at least its SettingsRowHeight: the grid's rows,
	// one track each, those of the fields at least that high.
	least := float32(0)
	tk, scale := skinOf(c)
	if tk != nil {
		least = tk.SettingsRowHeight * scale
	}
	row := func(min float32) {
		if least > 0 {
			f.tracks = append(f.tracks, ui.FitContent().Min(min))
		}
	}
	return ui.Column(c).Children(func() {
		outer := axis
		axis = vertical
		defer func() { axis = outer }()
		shown := false // a card is up
		for i := 0; i < len(f.rows); {
			start, end := i, f.sectionEnd(i)
			i = end
			if !f.anyShown(start, end) {
				for _, r := range f.rows[start:end] {
					f.hold(r)
				}
				continue
			}
			gap := float32(0) // above the title, or the card when it has none
			if shown {
				gap = sectionGap
			}
			for _, h := range f.headings(start) {
				if h.text != "" {
					render(c, h).Role(ui.RoleHeading).Margin(gap, 0, t.Space(1.5), 0)
					gap = 0
				}
			}
			shown = true
			// The card, as Card draws one, is the grid of its rows.
			g := ui.Grid(c).ColumnTracks(ui.Fr(1).MinContent(), ui.FitContent()).GapX(colGap).GapY(rowGap).
				Margin(gap, 0, 0, 0).Padding(t.Space(1), t.Space(3)).Border(1, t.Border).Radius(t.Radius).
				Background(t.Surface)
			if tk != nil {
				g.Radius(tk.RadiusCard)
				if sh := tk.CardShadow; sh.Color.A != 0 {
					g.Shadow(0, sh.Y, sh.Blur, sh.Spread, sh.Color)
				}
			}
			if look := lookOf(skinhook.KindSettingsCard); look != nil {
				st := look.effective(t.Dark, g.Hovered(), false, false, g.Pressed(), f.disabled)
				st.apply(c, g, f.textKey)
			}
			f.tracks = f.tracks[:0]
			g.Children(func() {
				// The cells are built as for a row, as in the plain grid.
				axis = horizontal
				defer func() { axis = vertical }()
				first := true // no divider is due
				for j := start; j < end; j++ {
					r := &f.rows[j]
					if r.w.base().hidden {
						f.hold(*r)
						continue
					}
					if !first {
						ui.Divider(c.Key(r.rule)).ColumnSpan(2)
						row(0)
					}
					row(least)
					first = false
					f.cells(c, j)
				}
			})
			g.RowTracks(f.tracks...)
		}
	})
}

// sectionEnd returns the end of the rows of the section row i is in: the
// next row a section starts at, or the last. Under mu.
func (f *FieldsWidget) sectionEnd(i int) int {
	end := len(f.rows)
	for _, h := range f.heads {
		if h.at > i && h.at < end {
			end = h.at
		}
	}
	return end
}

// anyShown reports whether a row of rows [i, end) shows. Under mu.
func (f *FieldsWidget) anyShown(i, end int) bool {
	for _, r := range f.rows[i:end] {
		if !r.w.base().hidden {
			return true
		}
	}
	return false
}

// cells builds row i of a grid: its label, with its description under it,
// and its control, across its column where it asks, else stretched or, in
// SettingsRows, at its end. Under mu.
func (f *FieldsWidget) cells(c *ui.Context, i int) {
	r := &f.rows[i]
	r.underline(c)
	r.describe()
	if r.wide {
		f.wideCell(c, r)
		return
	}
	// e is the label, which names the control; cell the label's cell, the
	// label or a column of it and the description.
	var e, cell ui.Element
	if r.desc != "" && !r.label.hidden {
		cell = ui.Column(c).Children(func() {
			e = render(c, r.label)
			ui.Text(c, r.desc).TextColor(c.Theme().TextMuted).Disabled(r.w.base().disabled)
		})
	} else {
		e = render(c, r.label) // holds a hidden label
		cell = e
	}
	if r.key != 0 {
		w := r.w
		registerMnemonic(r.key, e, w, func() { buildingForm.focusRow(w) })
	}
	if r.label.hidden { // keep its cell, or ui flows the control into it
		ui.Box(c.Key(fieldHole(r.label.id)))
	} else {
		place(cell, r.label, ui.Start)
		if r.label.place.valign == 0 {
			cell.AlignSelf(ui.Center)
		}
	}
	def := ui.Stretch
	if f.settings {
		def = ui.End
	}
	var ce ui.Element // the control's cell: the control, or a row of it and the action
	if r.acts() {
		ce = ui.Row(c).Gap(f.colSpace(c)).Children(func() {
			w := render(c, r.w).NamedBy(e)
			if r.w.base().place.align == 0 {
				w.Grow(1) // fills the cell, as the control alone does
			}
			if !f.settings && r.w.base().place.valign == 0 {
				w.AlignSelf(ui.Stretch) // down the cell too, as the control alone does
			}
			render(c, r.action)
		})
	} else {
		ce = render(c, r.w).NamedBy(e)
	}
	place(ce, r.w, def)
	// A control keeps its own height next to a wrapped label, and sits
	// at its middle, as the label does.
	if f.settings && r.w.base().place.valign == 0 && !fillsAcross(r.w) {
		ce.AlignSelf(ui.Center)
	}
}

// wideCell builds the wide row r of a grid as one cell across both columns,
// a column of its label, its description and its control, stretched unless
// its Align says otherwise, as a stacked row is. A label with no text has no
// node. Under mu.
func (f *FieldsWidget) wideCell(c *ui.Context, r *fieldRow) {
	gap := c.Theme().Space(1)
	if f.gapSet {
		gap = f.colGap
	}
	col := ui.Column(c).Gap(gap).ColumnSpan(2)
	if f.settings {
		// A control wider than the card is the card's to clip: the columns
		// do not make room for it, which would shrink the other rows' controls.
		col.MaxWidthPercent(100)
	}
	if f.settings && r.label.text == "" {
		col.AlignSelf(ui.Center) // no label line: the control in the middle of its band, as a row's is
	}
	col.Children(func() {
		// The cell's children are a column's: Align is across it.
		axis = vertical
		defer func() { axis = horizontal }()
		// e is the label, which names the control.
		e := f.labelLine(c, r, func() (e ui.Element) {
			if r.label.text == "" {
				keeping.hold(r.label.base())
				return e
			}
			e = render(c, r.label) // holds a hidden label
			if !r.label.hidden && r.label.place.align == 0 {
				e.AlignSelf(ui.Start)
			}
			return e
		})
		if r.key != 0 {
			w := r.w
			registerMnemonic(r.key, e, w, func() { buildingForm.focusRow(w) })
		}
		if r.desc != "" && !r.label.hidden {
			ui.Text(c, r.desc).TextColor(c.Theme().TextMuted).Disabled(r.w.base().disabled)
		}
		render(c, r.w).NamedBy(e)
	})
}

// labelLine builds a wide row's label with label, and its action at the end
// of the label's line: a row of the two, the label growing, both at its
// middle. A label with an Align of its own sits by it in the room left of
// the action. It returns the label. Under mu, in a column.
func (f *FieldsWidget) labelLine(c *ui.Context, r *fieldRow, label func() ui.Element) ui.Element {
	if !r.acts() {
		return label()
	}
	var e ui.Element
	ui.Row(c).Gap(f.colSpace(c)).Justify(ui.End).Children(func() {
		axis = horizontal
		defer func() { axis = vertical }()
		if a := r.label.place.align; a != 0 {
			// The label fills the line, so a box of that room places it.
			ui.Row(c).Grow(1).MinWidth(0).Justify(a.ui()).Children(func() {
				e = label().AlignSelf(ui.Center)
			})
		} else {
			e = label().Grow(1).MinWidth(0).AlignSelf(ui.Center) // a long word clips, not push the action out
		}
		render(c, r.action)
	})
	return e
}

// acts reports whether r has an action that shows; it holds a hidden one.
// Under mu.
func (r *fieldRow) acts() bool {
	if r.action == nil {
		return false
	}
	if r.action.base().hidden {
		keeping.hold(r.action.base())
		return false
	}
	return true
}

// colSpace is the space between the label column and the control column,
// and before an action.
func (f *FieldsWidget) colSpace(c *ui.Context) float32 {
	if f.gapSet {
		return f.colGap
	}
	return c.Theme().Space(3)
}

// hold keeps the widgets of a row the form does not build, so that showing
// one tells the windows that show the form. Under mu.
func (f *FieldsWidget) hold(r fieldRow) {
	keeping.hold(r.w.base())
	keeping.hold(r.label.base())
	if r.action != nil {
		keeping.hold(r.action.base())
	}
}

// place puts e, the cell of w, across its column: where w asks, else def.
func place(e ui.Element, w Control, def ui.Align) {
	if a := w.base().place.align; a != 0 {
		e.JustifySelf(a.ui())
	} else if def != ui.Stretch {
		e.JustifySelf(def)
	}
}

func (f *FieldsWidget) children() []Control {
	var cs []Control
	for _, r := range f.rows {
		if r.label != nil {
			cs = append(cs, r.label)
		}
		cs = append(cs, r.w)
		if r.action != nil {
			cs = append(cs, r.action)
		}
	}
	return cs
}

// underline shows the label's mnemonic underlined while Alt is held, and only
// where a form answers it. Under mu, while building.
func (r fieldRow) underline(c *ui.Context) {
	if r.key != 0 && !macMenus && buildingForm != nil && c.Modifiers()&ui.Alt != 0 {
		r.label.spans = mnemonicSpans(r.label.text, r.at)
	} else {
		r.label.spans = nil
	}
}
