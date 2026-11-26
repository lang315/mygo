package widget_test

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The snippets of docs/ui/widget/rangetrackbar.md, which must compile.
func ExampleRangeTrackBar() {
	var price *widget.RangeTrackBarWidget
	shown := widget.Label("$0 to $500")
	price = widget.RangeTrackBar(0, 500).SetStep(10).SetValues(50, 200).OnChange(func() {
		lo, hi := price.Values()
		shown.SetText(fmt.Sprintf("$%.0f to $%.0f", lo, hi))
	})
}

// The snippets of docs/ui/widget/rating.md, which must compile.
func ExampleRating() {
	var stars *widget.RatingWidget
	stars = widget.Rating(5).SetValue(3).OnChange(func() { fmt.Println("rated", stars.Value()) })
	score := 4
	bound := widget.Bind(widget.Rating(5), func() int { return score }, func(v int) { score = v })
	_ = bound
}

// The snippets of docs/ui/widget/checkbutton.md, which must compile.
func ExampleCheckButton() {
	var boldIcon *ui.SVG
	var bold *widget.CheckButtonWidget
	bold = widget.CheckButton("Bold").SetIcon(boldIcon).OnChange(func() { fmt.Println("bold", bold.Checked()) })
	wrap := widget.CheckButton("Wrap lines").SetChecked(true)
	_ = wrap
}

// The snippets of docs/ui/widget/segmented.md, which must compile.
func ExampleSegmented() {
	var view *widget.SegmentedWidget
	view = widget.Segmented("List", "Grid").OnChange(func() { fmt.Println("view", view.Selected()) })
	period := widget.Segmented("Day", "Week", "Month").SetSelected(1).SetFillWidth(true)
	_ = period
}

// The snippets of docs/ui/widget/splitbutton.md, which must compile.
func ExampleSplitButton() {
	save := widget.SplitButton("Save",
		widget.MenuItem{Text: "Save As…", OnClick: func() { fmt.Println("save as") }},
		widget.MenuItem{Text: "Save All", OnClick: func() { fmt.Println("save all") }},
	).Suggested().OnClick(func() { fmt.Println("save") })
	_ = save
}
