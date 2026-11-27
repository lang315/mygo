package main

import (
	"fmt"

	"github.com/egoist/mygo/ui/widget"
)

// product is a record of the Choices tab's LookUpEdit.
type product struct {
	ID         int
	Code, Name string
	Group      string
}

// productLookUp is a LookUpEdit of 10,000 products with its search panel:
// the drop-down's search box finds a product by its code, name or group,
// and only the rows in view are built. The label under it says the choice.
func productLookUp() widget.Control {
	groups := []string{"Tools", "Paint", "Garden", "Lighting"}
	items := make([]product, 10000)
	for i := range items {
		items[i] = product{i + 1, fmt.Sprintf("P%05d", i+1), fmt.Sprintf("Product %d", i+1), groups[i%len(groups)]}
	}
	said := widget.Label("No product")
	look := widget.LookUpEdit(
		func(p *product) int { return p.ID },
		func(p *product) string { return p.Name },
		widget.Column[product]{Title: "Code", Text: func(p *product) string { return p.Code }, Width: 70},
		widget.Column[product]{Title: "Name", Text: func(p *product) string { return p.Name }},
		widget.Column[product]{Title: "Group", Text: func(p *product) string { return p.Group }, Width: 80},
	).SetItems(items).SetSearchPanel(true).SetAccessibleName("Product")
	look.OnSelect(func(id int, ok bool) {
		if ok {
			said.SetText(fmt.Sprintf("Product %d chosen", id))
		}
	})
	return widget.VBox(4).Add(look, said)
}
