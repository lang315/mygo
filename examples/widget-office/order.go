package main

import "github.com/egoist/mygo/ui/widget"

// order is a document of the Orders page: a customer and notes.
type order struct {
	n        int
	customer *widget.LookUpEditWidget[customer, int]
	notes    *widget.MultilineTextBoxWidget
	page     widget.Control
	saved    fields // what was last saved
}

// fields are what the user edits in an order.
type fields struct {
	customer int // 0 for none
	notes    string
}

func newOrder(n int) *order {
	o := &order{n: n}
	o.customer = widget.LookUpEdit(
		func(c *customer) int { return c.ID },
		func(c *customer) string { return c.Name },
		widget.Column[customer]{Title: "Code", Text: func(c *customer) string { return c.Code }, Width: 60},
		widget.Column[customer]{Title: "Name", Text: func(c *customer) string { return c.Name }},
		widget.Column[customer]{Title: "City", Text: func(c *customer) string { return c.City }, Width: 100},
	).SetItems(customers)
	o.notes = widget.MultilineTextBox()
	o.page = widget.Fields().Row("&Customer", o.customer).Row("&Notes", o.notes).Wide()
	return o
}

// state returns what the user has edited so far.
func (o *order) state() fields {
	id, _ := o.customer.Selected()
	return fields{id, o.notes.Text()}
}

// dirty reports whether the order changed since it was saved.
func (o *order) dirty() bool { return o.state() != o.saved }
