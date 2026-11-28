package widget_test

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of the search panel in docs/ui/widget/lookupedit.md, which must
// compile.
func ExampleLookUpEditWidget_SetSearchPanel() {
	type Product struct {
		ID         int
		Code, Name string
	}
	var products []Product
	product := widget.LookUpEdit(
		func(p *Product) int { return p.ID },
		func(p *Product) string { return p.Name },
		widget.Column[Product]{Title: "Code", Text: func(p *Product) string { return p.Code }, Width: 70},
		widget.Column[Product]{Title: "Name", Text: func(p *Product) string { return p.Name }},
	).SetItems(products).SetSearchPanel(true) // a search box over the records, in the drop-down
	product.OnSelect(func(id int, ok bool) { fmt.Println(id, ok) })
	_ = widget.Fields().Row("&Product", product)
}
