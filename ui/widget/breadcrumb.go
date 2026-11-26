package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
)

// BreadCrumbWidget is a path of items separated by chevrons, as
// DevExpress's BreadCrumbEdit for navigation: a click on an item but the
// last chooses it, and the last is where the path is, which assistive
// technology reads as the current page. It is ui.Breadcrumbs underneath.
type BreadCrumbWidget struct {
	common[BreadCrumbWidget]
	items    []string
	chosen   int // ui's, which only the drawing thread touches
	onSelect func(i int)
	pick     func() // picked, made once
}

// BreadCrumb makes a path of items, the first first.
func BreadCrumb(items ...string) *BreadCrumbWidget {
	b := &BreadCrumbWidget{items: slices.Clone(items), chosen: -1}
	b.init(b)
	return b
}

// SetItems replaces the items.
func (b *BreadCrumbWidget) SetItems(items ...string) *BreadCrumbWidget {
	items = slices.Clone(items) // ui reads the slice while it draws
	b.change(func() { b.items = items })
	return b
}

// OnSelect runs fn, on the event goroutine, with the index of the item the
// user clicked, each time, the same item again too. The app usually trims
// its path to [:i+1].
func (b *BreadCrumbWidget) OnSelect(fn func(i int)) *BreadCrumbWidget {
	b.change(func() { b.onSelect = fn })
	return b
}

// picked is ui's OnChange callback, made once.
func (b *BreadCrumbWidget) picked() {
	i := b.chosen
	mu.Lock()
	fn := b.onSelect
	mu.Unlock()
	if fn != nil {
		Post(func() { fn(i) })
	}
}

func (b *BreadCrumbWidget) build(c *ui.Context) ui.Element {
	if b.pick == nil {
		b.pick = b.picked
	}
	return ui.Breadcrumbs(c, b.items, &b.chosen).OnChange(b.pick)
}
