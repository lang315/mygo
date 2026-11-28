package widget_test

import "github.com/egoist/mygo/ui/widget"

// The FontEdit of docs/ui/widget/fontedit.md: a monospaced family for an
// editor.
func ExampleFontEdit() {
	var font *widget.FontEditWidget
	font = widget.FontEdit().SetFamily("Menlo").SetFixedPitchOnly(true).OnChange(func() {
		println(font.Family())
	})
}
