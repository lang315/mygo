package widget

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/internal/i18nlink"
	"github.com/egoist/mygo/internal/l10n"
)

// PanelText holds the words of a table's bottom panel: the formats of its
// counts, each with one %d (a text without % shows as it is, an empty one
// hides its count, and a % shown is written %%), and the text of its search
// button.
type PanelText struct {
	Rows, Selected, Checked string
	Search                  string
}

// BottomPanel shows under the table how many rows it has, how many of those
// shown are selected and checked (as Selected and Checked return them), and,
// with Filterable columns, a search: a column chosen in a combo, a keyword,
// and a button (or Enter) running it.
func (t *DataGridViewWidget[T, K]) BottomPanel() *DataGridViewWidget[T, K] {
	total, selected, checked := Label(""), Label(""), Label("")
	panel := HBox(12).Add(total, selected, checked)
	var opts []Option
	for _, cl := range t.cols { // cols never change
		if cl.Filterable {
			opts = append(opts, Option{Value: cl.Title, Text: cl.Title})
		}
	}
	var column *ComboBoxWidget
	var keyword *TextBoxWidget
	var button *ButtonWidget
	var search *BoxWidget
	if len(opts) > 0 {
		column = ComboBox(opts...).SetValue(opts[0].Value)
		keyword = TextBox().SetPlaceholder(l10n.TableKeyword()).OnEnter(t.searchPanel).SetSize(240, 0)
		button = Button("").OnClick(t.searchPanel) // counts sets its text
		search = HBox(12).Add(column, keyword, button)
		panel.PackEnd(search, false, false)
	}
	t.change(func() {
		t.panel, t.totalLabel, t.selectedLabel, t.checkedLabel = panel, total, selected, checked
		t.column, t.keyword, t.searchButton, t.searchBox = column, keyword, button, search
	})
	return t
}

// PanelText sets the words of the bottom panel, in the app's language by
// default.
func (t *DataGridViewWidget[T, K]) PanelText(pt PanelText) *DataGridViewWidget[T, K] {
	t.change(func() { t.text, t.textSet, t.words = pt, true, panelWords{} })
	return t
}

// ApplyPanel styles the bottom panel with styles, as Apply does a widget:
// its Background, and a TextSize or FontSize its counts and its search
// take. Call it after BottomPanel.
func (t *DataGridViewWidget[T, K]) ApplyPanel(styles ...Style) *DataGridViewWidget[T, K] {
	mu.Lock()
	p := t.panel
	mu.Unlock()
	if p != nil {
		p.Apply(styles...)
	}
	return t
}

// SetSearchWidth sets the width of the panel's keyword, in DIPs: 240 by
// default. Call it after BottomPanel.
func (t *DataGridViewWidget[T, K]) SetSearchWidth(w float32) *DataGridViewWidget[T, K] {
	mu.Lock()
	e := t.keyword
	mu.Unlock()
	if e != nil {
		e.SetSize(w, 0)
	}
	return t
}

// JoinSearch, when on, draws the panel's column, keyword and search button
// as one control: side by side, with no room and no rounded corners between
// them; off, as by default, undoes it. Call it after BottomPanel.
func (t *DataGridViewWidget[T, K]) JoinSearch(on bool) *DataGridViewWidget[T, K] {
	mu.Lock()
	box, column, keyword, button := t.searchBox, t.column, t.keyword, t.searchButton
	mu.Unlock()
	if box == nil {
		return t
	}
	box.change(func() {
		box.spacing = 12
		if on {
			box.spacing = 0
		}
	})
	// The three are the panel's own, styled by nothing else: Apply() undoes
	// what JoinSearch(true) applied, and only that.
	if on {
		column.Apply(Style{}.RadiusTopRight(0).RadiusBottomRight(0))
		keyword.Apply(Style{}.Radius(0))
		button.Apply(Style{}.RadiusTopLeft(0).RadiusBottomLeft(0))
	} else {
		column.Apply()
		keyword.Apply()
		button.Apply()
	}
	return t
}

// OnBeforeSearch runs fn with the column and the keyword, trimmed, of a
// search from the bottom panel, before it runs.
func (t *DataGridViewWidget[T, K]) OnBeforeSearch(fn func(column, keyword string)) *DataGridViewWidget[T, K] {
	t.change(func() { t.beforeSearch = fn })
	return t
}

// SetSearchPlaceholder sets what the panel's empty keyword shows; call it
// after BottomPanel.
func (t *DataGridViewWidget[T, K]) SetSearchPlaceholder(s string) *DataGridViewWidget[T, K] {
	mu.Lock()
	e := t.keyword
	mu.Unlock()
	if e != nil {
		e.SetPlaceholder(s)
	}
	return t
}

// searchPanel runs the panel's search, on the event goroutine.
func (t *DataGridViewWidget[T, K]) searchPanel() {
	mu.Lock()
	column, keyword, before := t.column, t.keyword, t.beforeSearch
	mu.Unlock()
	col, kw := column.Value(), strings.TrimSpace(keyword.Text())
	if before != nil {
		before(col, kw)
	}
	t.Search(col, kw)
}

// Search shows only the rows whose column titled column holds keyword,
// as the column's Filter decides; a keyword of spaces, or a column not
// Filterable, shows them all. The bottom panel shows the search too.
func (t *DataGridViewWidget[T, K]) Search(column, keyword string) *DataGridViewWidget[T, K] {
	keyword = strings.TrimSpace(keyword)
	mu.Lock()
	combo, entry := t.column, t.keyword
	mu.Unlock()
	if combo != nil {
		combo.SetValue(column)
		entry.SetText(keyword)
	}
	t.change(func() {
		var before []K
		t.selSeen = nil // the shown selection is taken anew under the new search
		watch := t.onSelChanged != nil && t.chosen.value.Len() > 0
		if watch {
			before = t.shown(&t.chosen.value)
		}
		t.filter = t.keep(column, keyword)
		t.view.SetFilter(t.filter)
		if watch && !sameKeys(before, t.shown(&t.chosen.value)) {
			t.selectionChanged()
		}
	})
	return t
}

// sameKeys reports whether a and b, keys each once, hold the same keys.
func sameKeys[K comparable](a, b []K) bool {
	if len(a) != len(b) {
		return false
	}
	in := make(map[K]struct{}, len(a))
	for _, k := range a {
		in[k] = struct{}{}
	}
	for _, k := range b {
		if _, ok := in[k]; !ok {
			return false
		}
	}
	return true
}

// keep returns the filter of a search, nil for none. Under mu.
func (t *DataGridViewWidget[T, K]) keep(column, keyword string) func(*T) bool {
	i := slices.IndexFunc(t.cols, func(cl Column[T]) bool { return cl.Filterable && cl.Title == column })
	if keyword == "" || i < 0 {
		return nil
	}
	if f := t.cols[i].Filter; f != nil {
		return func(r *T) bool { return f(r, keyword) }
	}
	text, kw := t.cols[i].Text, strings.ToLower(keyword)
	return func(r *T) bool { return strings.Contains(strings.ToLower(text(r)), kw) }
}

// panelWords are the panel's texts and what they were worked out from.
type panelWords struct {
	ok                    bool
	rows, selected, check int
	locale                string
	catalogs              uint64
	set                   bool
	total, sel, chk, find string
}

// counts sets the panel's counts, and its words. They are worked out again
// only when a count, the language or PanelText changed: through the catalog
// each costs allocations, in every frame. Under mu, while building.
func (t *DataGridViewWidget[T, K]) counts() {
	rows := len(t.src.rows)
	selected, checked := t.panelCounts()
	locale, catalogs := "", uint64(0)
	if !t.textSet {
		locale, catalogs = i18n.Locale(), i18nlink.Version()
	}
	w := &t.words
	if !w.ok || w.rows != rows || w.selected != selected || w.check != checked || w.locale != locale || w.catalogs != catalogs || w.set != t.textSet {
		*w = panelWords{ok: true, rows: rows, selected: selected, check: checked, locale: locale, catalogs: catalogs, set: t.textSet}
		if t.textSet {
			w.find = t.text.Search
			w.total, w.sel, w.chk = countText(t.text.Rows, rows), countText(t.text.Selected, selected), countText(t.text.Checked, checked)
		} else {
			w.find = l10n.TableSearch()
			w.total, w.sel, w.chk = l10n.TableRows(rows), l10n.TableSelected(selected), l10n.TableChecked(checked)
		}
	}
	search, total, sel, chk := w.find, w.total, w.sel, w.chk
	if t.searchButton != nil {
		t.searchButton.text = search
	}
	t.totalLabel.text, t.selectedLabel.text, t.checkedLabel.text = total, sel, chk
	t.totalLabel.hidden = t.textSet && t.text.Rows == ""
	t.selectedLabel.hidden = t.textSet && t.text.Selected == ""
	t.checkedLabel.hidden = !t.checks || t.textSet && t.text.Checked == ""
}

// countText is a count of n in format; a format with no verb, but %% at
// most, is a text, with %% for %.
func countText(format string, n int) string {
	if !strings.Contains(strings.ReplaceAll(format, "%%", ""), "%") {
		return strings.ReplaceAll(format, "%%", "%")
	}
	return fmt.Sprintf(format, n)
}

// shownCounts are the counts of the rows selected and checked that a search
// shows, as of a version of the view, a change of the source (which may put
// other rows in the places the view shows) and a version of the keys.
type shownCounts struct {
	ok                     bool
	version, seq, sel, chk uint64
	selected, checked      int
}

// panelCounts returns how many rows shown are selected and checked. Under a
// search it counts them only when the view, the source or the keys changed:
// counts runs in every frame. Under mu, while building.
func (t *DataGridViewWidget[T, K]) panelCounts() (selected, checked int) {
	if t.filter == nil {
		return t.chosen.value.Len(), t.checked.value.Len()
	}
	c := &t.shownN
	if !c.ok || c.version != t.view.Version() || c.seq != t.src.seq || c.sel != t.chosen.gen || c.chk != t.checked.gen {
		*c = shownCounts{true, t.view.Version(), t.src.seq, t.chosen.gen, t.checked.gen,
			t.shownCount(&t.chosen.value), t.shownCount(&t.checked.value)}
	}
	return c.selected, c.checked
}
