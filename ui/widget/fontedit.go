package widget

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// FontEditWidget is a field that picks a font family from a drop-down of
// ui.FontFamilies, each name drawn in its own family, with a search box
// whose typing filters them: the families that start with what is typed
// first, then those that hold it (DevExpress's FontEdit).
type FontEditWidget struct {
	common[FontEditWidget]
	family   string // under mu
	fixed    bool
	onChange func()
	families func() []ui.FontFamily // ui.FontFamilies, read at the first open
	all      []ui.FontFamily        // what families gave, sorted; nil until then
	// What only the drawing thread touches, under mu while building.
	open      bool // ui's Popover writes it while the view builds
	seenHides int  // the hides the last build saw: a hide closes the drop-down
	query     string
	matched   bool   // shown holds the matches of seenQuery
	seenQuery string // the query shown matches
	shown     []int  // the indexes in all of the rows
	hl        int    // the index in shown of the row highlighted, -1 for none
	list      ui.ListState
	c         *ui.Context // the context of the last build, for the rows ui lays out after it
	in        ui.Element  // the search box of the last build, whose active descendant the highlighted row is
	// The functions handed to ui, made once.
	children func()
	text     func()
	panel    func(panel ui.Element)
	row      func(i int)
}

// FontEdit makes an empty font family field. The families are read when
// its drop-down first opens.
func FontEdit() *FontEditWidget {
	f := &FontEditWidget{families: ui.FontFamilies, hl: -1}
	f.children = f.buildField
	f.text = f.buildText
	f.panel = f.buildPopup
	f.row = f.buildRow
	f.init(f)
	f.lookAs(skinhook.KindField)
	return f
}

// Family returns the family chosen.
func (f *FontEditWidget) Family() string { mu.Lock(); defer mu.Unlock(); return f.family }

// SetFamily sets the family, without running OnChange. A family the system
// lacks keeps its name, drawn in the default face.
func (f *FontEditWidget) SetFamily(name string) *FontEditWidget {
	f.change(func() { f.family = name })
	return f
}

// SetFixedPitchOnly lists only monospaced families, as FontDialog's
// SetFixedPitchOnly. It does not change the family chosen.
func (f *FontEditWidget) SetFixedPitchOnly(on bool) *FontEditWidget {
	f.change(func() { f.fixed, f.matched = on, false })
	return f
}

// OnChange runs fn, on the event goroutine, after the user picks a family
// other than the one chosen. A setter does not run it.
func (f *FontEditWidget) OnChange(fn func()) *FontEditWidget {
	f.change(func() { f.onChange = fn })
	return f
}

// fontEditRows is how many rows the drop-down shows at most.
const fontEditRows = 10

func (f *FontEditWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	f.c = c
	frame := ui.Row(c).Radius(t.Radius).Border(1, t.Outline()).Background(t.FieldFill()).
		AlignItems(ui.Stretch).MinWidth(t.Space(35))
	if h := t.Metrics.ControlHeight; h > 0 {
		frame.MinHeight(h)
	}
	frame.Children(f.children)
	switch {
	case f.inner.Focused():
		frame.Border(1, t.Accent)
	case frame.Hovered():
		frame.Border(1, t.Outline().Mix(t.Text, 0.25))
	}
	if f.errMsg != "" {
		f.post = errorMarkRoom // the chevron stops before the mark
	}
	return frame
}

// buildField builds the field: the family, which a click, Enter, Space,
// Down or Alt+Down opens the drop-down from. Under mu, building.
func (f *FontEditWidget) buildField() {
	c, t := f.c, f.c.Theme()
	if f.seenHides != f.hides { // out of sight since the last build: it closes
		f.open, f.seenHides = false, f.hides
	}
	field := ui.ButtonBase(c).Role(ui.RoleComboBox).Value(f.family).Grow(1).
		Padding(t.Space(1.5), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.SpaceBetween)
	f.inner = field
	field.Children(f.text)
	opened := false
	if field.Clicked() {
		f.open = !f.open // a press on the field is not outside the drop-down: it toggles
		opened = f.open
	} else if !f.open && (field.Shortcut(ui.Alt, ui.KeyDown) || field.Shortcut(0, ui.KeyDown)) {
		f.open, opened = true, true
	}
	if opened {
		f.query, f.hl, f.matched, f.list = "", -1, false, ui.ListState{}
	}
	field.Expanded(f.open)
	if f.open {
		ui.PopoverBase(c, field, &f.open, f.panel)
	}
}

// buildText builds what the field holds: the family, in the default face.
// Under mu, building.
func (f *FontEditWidget) buildText() {
	c, t := f.c, f.c.Theme()
	ui.Text(c, f.family).SingleLine().AccessHidden(true) // the field's value
	ui.Icon(c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted)
}

// buildPopup builds the drop-down: the search box and a list of the
// matching families, which builds only those in view. Under mu, building.
func (f *FontEditWidget) buildPopup(panel ui.Element) {
	c, t := f.c, f.c.Theme()
	comboPanel(c, panel)
	panel.Gap(t.Space(1)).Width(t.Space(60)).MinWidth(f.inner.Bounds().W) // at least as wide as the field, as ui's combo box at least as wide as the field, as ui's combo box
	if f.all == nil {
		f.all = sortFamilies(f.families())
	}
	in := ui.TextInput(c, &f.query).AutoFocus().Label(l10n.Search()).Placeholder(l10n.Search())
	f.in = in
	if !f.matched || f.query != f.seenQuery {
		f.shown = f.matching(f.query)
		f.matched, f.seenQuery, f.hl = true, f.query, -1
		switch {
		case f.query != "" && len(f.shown) > 0:
			f.hl = 0 // Enter picks the first match
		case f.query == "":
			for i, k := range f.shown {
				if strings.EqualFold(f.all[k].Name, f.family) { // without regard to case, as the text system does // without regard to case, as the text system does
					f.hl = i
				}
			}
		}
		f.list.ScrollTo(max(f.hl, 0), ui.Start)
	}
	n := len(f.shown)
	if in.Shortcut(0, ui.KeyDown) && f.hl < n-1 {
		f.hl++
		f.list.ScrollIntoView(f.hl)
	}
	if in.Shortcut(0, ui.KeyUp) && f.hl > 0 {
		f.hl--
		f.list.ScrollIntoView(f.hl)
	}
	if in.Submitted() && f.hl >= 0 { // hl < n: the matches reset it
		f.pick(f.shown[f.hl])
		return
	}
	if n == 0 {
		ui.Text(c, l10n.NoResults()).TextColor(t.TextMuted).Padding(t.Space(1.5), t.Space(2.5))
		return
	}
	rh := t.Metrics.RowHeight
	if rh <= 0 {
		rh = t.Space(7)
	}
	ui.List(c, &f.list, n, f.row).Height(float32(min(n, fontEditRows)) * rh)
}

// buildRow builds the row of the list at i, the family's name in its own
// family: while the list builds, or when ui lays out rows it did not build
// then.
func (f *FontEditWidget) buildRow(i int) {
	if lockOutsideView() {
		defer mu.Unlock()
	}
	c, t := f.c, f.c.Theme()
	name := f.all[f.shown[i]].Name // shown changes only while building, which gives ui its length
	row := ui.Row(c).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius)
	if h := t.Metrics.RowHeight; h > 0 {
		row.MinHeight(h).PaddingY(0).AlignItems(ui.Center)
	}
	if i == f.hl {
		f.in.ActiveDescendant(row) // assistive technology reads the row the arrows are on
	}
	if i == f.hl || row.Hovered() {
		bg, fg := highlightOf(t)
		row.Background(bg).TextColor(fg)
	}
	if row.Clicked() {
		f.pick(f.shown[i])
	}
	row.Children(func() { ui.Text(c, name).Font(name).SingleLine() })
}

// matching returns the indexes in all of the families listed (monospaced
// ones alone with fixed pitch only) that start with q, case ignored, then
// of those that hold it; all of them for an empty q. Under mu.
func (f *FontEditWidget) matching(q string) []int {
	out := f.shown[:0]
	for i, fam := range f.all {
		if (!f.fixed || fam.Monospaced) && hasFoldedPrefix(fam.Name, q) {
			out = append(out, i)
		}
	}
	if q == "" {
		return out
	}
	lq := strings.ToLower(q)
	for i, fam := range f.all {
		if (!f.fixed || fam.Monospaced) && !hasFoldedPrefix(fam.Name, q) && strings.Contains(strings.ToLower(fam.Name), lq) {
			out = append(out, i)
		}
	}
	return out
}

// pick chooses the family of index i in all, as the user did, closes the
// drop-down, and runs OnChange and the binding's write if that changed the
// family. Under mu, building.
func (f *FontEditWidget) pick(i int) {
	f.open = false
	name := f.all[i].Name
	if name == f.family {
		return
	}
	f.family = name
	h, write := f.onChange, f.bound.write
	if h == nil && write == nil {
		return
	}
	Post(func() {
		if h != nil {
			h()
		}
		if write != nil {
			write(name)
		}
	})
}

// sortFamilies sorts fs as ui.FontFamilies does: the registered families
// first, then the rest, each without regard to case.
func sortFamilies(fs []ui.FontFamily) []ui.FontFamily {
	fs = slices.Clone(fs)
	slices.SortStableFunc(fs, func(a, b ui.FontFamily) int {
		if a.Registered != b.Registered {
			if a.Registered {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	if fs == nil {
		fs = []ui.FontFamily{} // read: not read again
	}
	return fs
}

func (f *FontEditWidget) bindValue() string  { return f.Family() }
func (f *FontEditWidget) bindApply(v string) { f.SetFamily(v) }
