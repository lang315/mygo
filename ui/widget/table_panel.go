package widget

import (
	"fmt"
	"slices"
	"strings"
)

// PanelText holds the words of a table's bottom panel: the formats of its
// counts, each with one %d, and the text of its search button.
type PanelText struct {
	Rows, Selected, Checked string
	Search                  string
}

var englishPanel = PanelText{Rows: "Rows: %d", Selected: "Selected: %d", Checked: "Checked: %d", Search: "Search"}

// BottomPanel shows under the table how many rows it has, are selected
// and are checked, and, with Filterable columns, a search: a column
// chosen in a combo, a keyword, and a button (or Enter) running it.
func (t *TableWidget[T, K]) BottomPanel() *TableWidget[T, K] {
	mu.Lock()
	text := t.text
	mu.Unlock()
	total, selected, checked := Label(""), Label(""), Label("")
	panel := HBox(12).Add(total, selected, checked)
	var opts []Option
	for _, cl := range t.cols { // cols never change
		if cl.Filterable {
			opts = append(opts, Option{Value: cl.Title, Text: cl.Title})
		}
	}
	var column *ComboWidget
	var keyword *EntryWidget
	var button *ButtonWidget
	if len(opts) > 0 {
		column = Combo(opts...).SetValue(opts[0].Value)
		keyword = Entry().SetPlaceholder("Keyword").OnEnter(t.searchPanel).SetSize(240, 0)
		button = Button(text.Search).OnClick(t.searchPanel)
		panel.PackEnd(button, false, false).PackEnd(keyword, false, false).PackEnd(column, false, false)
	}
	change(func() {
		t.panel, t.totalLabel, t.selectedLabel, t.checkedLabel = panel, total, selected, checked
		t.column, t.keyword, t.searchButton = column, keyword, button
	})
	return t
}

// PanelText sets the words of the bottom panel, in English by default.
func (t *TableWidget[T, K]) PanelText(pt PanelText) *TableWidget[T, K] {
	mu.Lock()
	t.text = pt
	b := t.searchButton
	mu.Unlock()
	if b != nil {
		b.SetText(pt.Search)
	}
	change(func() {})
	return t
}

// OnBeforeSearch runs fn with the column and the keyword, trimmed, of a
// search from the bottom panel, before it runs.
func (t *TableWidget[T, K]) OnBeforeSearch(fn func(column, keyword string)) *TableWidget[T, K] {
	change(func() { t.beforeSearch = fn })
	return t
}

// SetSearchPlaceholder sets what the panel's empty keyword shows; call it
// after BottomPanel.
func (t *TableWidget[T, K]) SetSearchPlaceholder(s string) *TableWidget[T, K] {
	mu.Lock()
	e := t.keyword
	mu.Unlock()
	if e != nil {
		e.SetPlaceholder(s)
	}
	return t
}

// searchPanel runs the panel's search, on the event goroutine.
func (t *TableWidget[T, K]) searchPanel() {
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
func (t *TableWidget[T, K]) Search(column, keyword string) *TableWidget[T, K] {
	keyword = strings.TrimSpace(keyword)
	mu.Lock()
	combo, entry := t.column, t.keyword
	mu.Unlock()
	if combo != nil {
		combo.SetValue(column)
		entry.SetText(keyword)
	}
	change(func() { t.view.SetFilter(t.keep(column, keyword)) })
	return t
}

// keep returns the filter of a search, nil for none. Under mu.
func (t *TableWidget[T, K]) keep(column, keyword string) func(*T) bool {
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

// counts sets the panel's counts. Under mu, while building.
func (t *TableWidget[T, K]) counts() {
	t.totalLabel.text = fmt.Sprintf(t.text.Rows, len(t.rows))
	t.selectedLabel.text = fmt.Sprintf(t.text.Selected, t.chosen.value.Len())
	t.checkedLabel.text = fmt.Sprintf(t.text.Checked, t.checked.value.Len())
	t.checkedLabel.hidden = !t.checks
}
