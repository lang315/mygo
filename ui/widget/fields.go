package widget

import "github.com/egoist/mygo/ui"

// FieldsWidget lays out a form: rows of a label and the control it names.
type FieldsWidget struct {
	common[FieldsWidget]
	rows           []fieldRow
	stacked        bool
	rowGap, colGap float32
	gapSet         bool
}

type fieldRow struct {
	label *LabelWidget
	w     Control
	key   rune // the label's mnemonic: Alt+key focuses w
	at    int  // its place in the label, in runes
}

// fieldRowKey keys the box of a stacked row, and fieldHole the cell of a
// two-column row whose label is hidden, by the label's id.
type (
	fieldRowKey int
	fieldHole   int
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

// Stacked puts each label above its control, over the form's width, as a
// VBox of a label and a control does, instead of in a column of their own.
func (f *FieldsWidget) Stacked() *FieldsWidget {
	f.change(func() { f.stacked = true })
	return f
}

// Gap sets the space between rows, and between the label column and the
// control column, in DIPs; stacked, between a label and its control. The
// default is the theme's: 2 units between rows and 3 between the columns,
// or 1 between a stacked label and its control.
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
	if f.stacked {
		return ui.Column(c).Gap(rowGap).Children(func() {
			// The rows are a column, whatever the box around the form.
			outer := axis
			axis = vertical
			defer func() { axis = outer }()
			for _, r := range f.rows {
				if r.w.base().hidden { // the row goes, its label with it
					f.hold(r)
					continue
				}
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
	return ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(colGap).GapY(rowGap).Children(func() {
		// A cell's Align and VAlign are across and along the cell, not a
		// box's axes: build the cells as for a row, whose cross axis is vertical.
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		for _, r := range f.rows {
			if r.w.base().hidden {
				f.hold(r)
				continue
			}
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
			place(render(c, r.w).NamedBy(e), r.w, ui.Stretch)
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
