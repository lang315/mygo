package widget

import "github.com/egoist/mygo/ui"

// FieldsWidget lays out a form: rows of a label and the control it names.
type FieldsWidget struct {
	common[FieldsWidget]
	rows           []fieldRow
	heads          []fieldHead
	stacked        bool
	settings       bool
	rowGap, colGap float32
	gapSet         bool
}

type fieldRow struct {
	label *LabelWidget
	w     Control
	key   rune // the label's mnemonic: Alt+key focuses w
	at    int  // its place in the label, in runes
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
	if c == nil {
		return f
	}
	display, key, at := parseMnemonic(label)
	l := Label(display)
	f.change(func() {
		for _, r := range f.rows {
			if r.w == c {
				return
			}
		}
		f.rows = append(f.rows, fieldRow{l, c, key, at})
		adopt(&f.core, c)
	})
	return f
}

// Section adds a heading, in bold, across the form's width before the rows
// that follow it, up to the next Section. The rows of all sections share one
// label column, and a section after the first has more room above it than a
// row has. A section whose rows are all hidden hides its heading and its room.
// In SettingsRows the heading takes the place of the divider before its rows.
func (f *FieldsWidget) Section(title string) *FieldsWidget {
	l := Label(title).Bold()
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

// Stacked puts each label above its control, over the form's width, as a
// VBox of a label and a control does, instead of in a column of their own.
func (f *FieldsWidget) Stacked() *FieldsWidget {
	f.change(func() { f.stacked, f.settings = true, false })
	return f
}

// SettingsRows lays the form out as a page of settings: each label takes the
// width the control leaves and wraps, the control sits at the right edge in
// its own size (a control's Align overrides), and a divider separates one row
// from the next. A hidden control takes its label and its divider away. Of
// Stacked and SettingsRows, the last call wins. A ProgressBar and an empty
// ListBox have no width in the fit column, and a ListBox with items or a
// table a narrow one: give them SetSize, or use Stacked. The label column is
// no narrower than its longest word, so in a narrow window the controls are
// pushed toward or past the right edge (a ComboBox beside "Notifications"
// needs about 229 DIPs): give the window a SetMinimumSize wide enough. In a
// ScrollPanel use Vertical, or the form gets no width to wrap at.
func (f *FieldsWidget) SettingsRows() *FieldsWidget {
	f.change(func() { f.settings, f.stacked = true, false })
	return f
}

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
	if f.stacked {
		return ui.Column(c).Gap(rowGap).Children(func() {
			// The rows are a column, whatever the box around the form.
			outer := axis
			axis = vertical
			defer func() { axis = outer }()
			shown := false
			for i, r := range f.rows {
				for _, h := range f.headings(i) {
					e := render(c, h)
					if shown {
						e.Margin(sectionGap, 0, 0, 0)
					}
					shown = true
				}
				if r.w.base().hidden { // the row goes, its label with it
					f.hold(r)
					continue
				}
				shown = true
				ui.Column(c.Key(fieldRowKey(r.label.id))).Gap(colGap).Children(func() {
					r.underline(c)
					e := render(c, r.label) // holds a hidden label
					if r.key != 0 {
						registerMnemonic(r.key, e, r.w, func() { focusLater(r.w) })
					}
					if !r.label.hidden && r.label.place.align == 0 {
						e.AlignSelf(ui.Start)
					}
					render(c, r.w).NamedBy(e)
				})
			}
		})
	}
	tracks := []ui.Track{ui.FitContent(), ui.Fr(1)}
	if f.settings {
		tracks = []ui.Track{ui.Fr(1).MinContent(), ui.FitContent()}
	}
	return ui.Grid(c).ColumnTracks(tracks...).GapX(colGap).GapY(rowGap).Children(func() {
		// A cell's Align and VAlign are across and along the cell, not a
		// box's axes: build the cells as for a row, whose cross axis is vertical.
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		first, shown := true, false // no divider is due; a row or heading is up
		for i, r := range f.rows {
			for _, h := range f.headings(i) {
				e := render(c, h).ColumnSpan(2)
				if shown {
					e.Margin(sectionGap, 0, 0, 0)
				}
				first, shown = true, true
			}
			if r.w.base().hidden {
				f.hold(r)
				continue
			}
			if f.settings && !first {
				ui.Divider(c.Key(fieldRule(r.label.id))).ColumnSpan(2)
			}
			first, shown = false, true
			r.underline(c)
			e := render(c, r.label) // holds a hidden label
			if r.key != 0 {
				registerMnemonic(r.key, e, r.w, func() { focusLater(r.w) })
			}
			if r.label.hidden { // keep its cell, or ui flows the control into it
				ui.Box(c.Key(fieldHole(r.label.id)))
			} else {
				place(e, r.label, ui.Start)
				if r.label.place.valign == 0 {
					e.AlignSelf(ui.Center)
				}
			}
			def := ui.Stretch
			if f.settings {
				def = ui.End
			}
			ce := render(c, r.w).NamedBy(e)
			place(ce, r.w, def)
			// A control keeps its own height next to a wrapped label, and sits
			// at its middle, as the label does.
			if f.settings && r.w.base().place.valign == 0 && !fillsAcross(r.w) {
				ce.AlignSelf(ui.Center)
			}
		}
	})
}

// hold keeps the widgets of a row the form does not build, so that showing
// one tells the windows that show the form. Under mu.
func (f *FieldsWidget) hold(r fieldRow) {
	keeping.hold(r.w.base())
	keeping.hold(r.label.base())
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
