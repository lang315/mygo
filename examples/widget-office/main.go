// Widget-office puts the controls of an office app together in one window: a
// Ribbon, a NavBar, closable tabs of orders with a LookUpEdit, a TreeView in
// columns and toasts, over the orders of a small shop.
package main

import (
	"fmt"
	"log"
	"slices"

	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

// The window's size, in DIPs.
const winW, winH = 1100, 700

func main() {
	err := widget.Run("Office", func(w *widget.Form) {
		build(w)
		w.SetSize(winW, winH)
	})
	if err != nil {
		log.Fatal(err)
	}
}

// office is the window's controls and its open orders.
type office struct {
	form   *widget.Form
	ribbon *widget.RibbonWidget
	nav    *widget.NavBarWidget
	docs   *widget.TabControlWidget      // the order tabs
	pages  map[string]func(visible bool) // shows or hides the page of a NavBar item
	open   []*order                      // the orders open, in the order of their tabs
	next   int                           // the number of the last order made
}

// build fills w with the app: a ribbon over a navigation bar and the page
// of the item it selects.
func build(w *widget.Form) *office {
	o := &office{form: w}
	o.docs = widget.TabControl().SetClosable(true).OnPageClosing(o.closing)
	tree, skins := customersPage(), skinPage()
	o.pages = map[string]func(bool){
		"orders":    func(on bool) { o.docs.SetVisible(on) },
		"customers": func(on bool) { tree.SetVisible(on) },
		"skin":      func(on bool) { skins.SetVisible(on) },
	}

	o.nav = widget.NavBar().
		AddGroup("Sales").
		AddItem("orders", "Orders", icon(shapesOrders)).
		AddItem("customers", "Customers", icon(shapesCustomer)).
		AddGroup("Settings").
		AddItem("skin", "Skin", icon(shapesSkin)).
		SetSelected("orders").
		OnSelect(o.show)
	o.show("orders")

	o.ribbon = o.makeRibbon()
	right := widget.VBox(0)
	right.Pack(o.docs, true, true).Pack(tree, true, true).Pack(skins, true, true)
	body := widget.HBox(0).Add(o.nav).Pack(right, true, true)
	w.SetContent(widget.VBox(0).Add(o.ribbon).Pack(body, true, true))

	o.newOrder()
	return o
}

// show shows the page of the NavBar's item key and hides the others.
func (o *office) show(key string) {
	for k, set := range o.pages {
		set(k == key)
	}
}

func (o *office) makeRibbon() *widget.RibbonWidget {
	nav := widget.ToolButton("Navigation").SetIcon(icon(shapesMenu)).SetToggle(true).SetChecked(true)
	nav.OnChange(func() { o.nav.SetVisible(nav.Checked()) })
	return widget.Ribbon().
		AddTab("Home").
		AddGroup("Order").
		AddLarge(
			widget.ToolButton("New").SetIcon(icon(shapesNew)).OnClick(func() {
				o.nav.SetSelected("orders")
				o.show("orders")
				o.newOrder()
			}),
			widget.ToolButton("Save").SetIcon(icon(shapesSave)).OnClick(o.save),
		).
		AddSmall(widget.ToolButton("Close").SetIcon(icon(shapesClose)).OnClick(o.closeShown)).
		AddGroup("View").
		AddSmall(nav).
		AddTab("Help").
		AddGroup("About").
		AddLarge(widget.ToolButton("About").SetIcon(icon(shapesInfo)).OnClick(func() {
			o.form.ShowToast(widget.Toast{Title: "Office", Message: "An office app of MyGo's controls."})
		}))
}

// shown returns the order whose tab is shown, or nil with none.
func (o *office) shown() *order {
	if i := o.docs.Page(); i >= 0 && i < len(o.open) {
		return o.open[i]
	}
	return nil
}

// save saves the order shown, and tells the user how it went.
func (o *office) save() {
	ord := o.shown()
	if ord == nil {
		return
	}
	id, ok := ord.customer.Selected()
	if !ok {
		o.form.ShowToast(widget.Toast{Status: widget.StatusWarning, Title: "Choose a customer first"})
		return
	}
	ord.saved = ord.state()
	o.form.ShowToast(widget.Toast{Status: widget.StatusSuccess, Title: fmt.Sprintf("Order %d saved for %s", ord.n, nameOf(id))})
}

// closeShown closes the order shown, as Ctrl+F4 does: asking first when it
// changed.
func (o *office) closeShown() {
	i := o.docs.Page()
	if ord := o.shown(); ord != nil && o.closing(ord.page) {
		o.docs.RemovePage(i)
	}
}

// closing is the tabs' OnPageClosing: an order that changed since it was
// saved asks before it goes, and a closing one is forgotten.
func (o *office) closing(page widget.Control) bool {
	i := slices.IndexFunc(o.open, func(ord *order) bool { return ord.page == page })
	if i < 0 {
		return true
	}
	ord := o.open[i]
	if ord.dirty() && !widget.Confirm(o.form, widget.Question, fmt.Sprintf("Discard Order %d?", ord.n), "Its changes are lost.") {
		return false
	}
	o.open = slices.Delete(o.open, i, i+1)
	return true
}

// newOrder adds an order after the others and shows it.
func (o *office) newOrder() {
	o.next++
	ord := newOrder(o.next)
	o.open = append(o.open, ord)
	o.docs.AddPage(fmt.Sprintf("Order %d", ord.n), ord.page)
	o.docs.SetPage(len(o.open) - 1)
}

// nameOf returns the name of the customer with id.
func nameOf(id int) string {
	for _, c := range customers {
		if c.ID == id {
			return c.Name
		}
	}
	return ""
}

// customersPage is the customers in a tree of their regions, in columns.
func customersPage() *widget.TreeViewWidget[node, string] {
	tree := widget.TreeView(func(n *node) string { return n.Key }, func(n *node) string { return n.Name }).
		SetColumns("Customer",
			widget.Column[node]{Title: "City", Text: func(n *node) string { return n.City }, Width: 160},
			widget.Column[node]{Title: "Orders", Text: (*node).orders, Width: 90, Align: widget.End},
		).
		SetRoots(roots())
	for _, r := range regions {
		tree.SetChildren("region:"+r, kids(r))
	}
	return tree
}

// skinItem is a skin of the Skin page: nil is the plain look.
type skinItem struct {
	Name string
	Skin *skin.Skin
}

// skinPage lists the built-in skins; choosing one applies it.
func skinPage() *widget.ListBoxWidget[skinItem, string] {
	items := []skinItem{{Name: "Plain"}}
	for _, s := range skin.Builtin() {
		items = append(items, skinItem{Name: s.Name(), Skin: s})
	}
	list := widget.ListBox(func(s *skinItem) string { return s.Name }, func(s *skinItem) string { return s.Name }).SetItems(items)
	return list.OnSelect(func(names []string) {
		for _, s := range items {
			if len(names) == 1 && s.Name == names[0] {
				skin.Use(s.Skin)
			}
		}
	})
}
