package widget

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// DomainUpDownWidget chooses one of a list of texts with a pair of arrows,
// as WinForms' DomainUpDown does: Up goes to the item before, Down to the
// next one, and with Wrap the ends lead to each other. The arrows are a
// ui.Stepper.
type DomainUpDownWidget struct {
	common[DomainUpDownWidget]
	items []string
	wrap  bool
	// pos is the stepper's value: n-1-index, so that Up, which raises it,
	// goes to the item before. With Wrap it may be -1 or n for a moment,
	// the step past an end, which the next commit turns into the other end.
	pos      mirror[float64]
	onChange func()
}

// DomainUpDown makes a chooser of items, at the first.
func DomainUpDown(items ...string) *DomainUpDownWidget {
	d := &DomainUpDownWidget{items: slices.Clone(items)}
	d.setIndex(0)
	d.init(d)
	return d
}

// setIndex selects item i, clamped; -1 where there are no items. Under mu.
func (d *DomainUpDownWidget) setIndex(i int) {
	n := len(d.items)
	if n == 0 {
		d.pos.set(0)
		return
	}
	d.pos.set(float64(n - 1 - min(max(i, 0), n-1)))
}

// index returns the selected item's index, -1 for none. Under mu.
func (d *DomainUpDownWidget) index() int {
	n := len(d.items)
	if n == 0 {
		return -1
	}
	return n - 1 - int(math.Round(min(max(d.pos.value, 0), float64(n-1))))
}

// wrapped turns a step past an end into the other end, and keeps any other
// value in range. Under mu.
func (d *DomainUpDownWidget) wrapped(s float64) float64 {
	n := float64(len(d.items))
	switch {
	case d.wrap && s < 0:
		return n - 1
	case d.wrap && s > n-1:
		return 0
	}
	return min(max(s, 0), max(n-1, 0))
}

// SelectedIndex returns the index of the selected item, -1 where there are
// no items; SelectedItem its text, "" for none.
func (d *DomainUpDownWidget) SelectedIndex() int {
	mu.Lock()
	defer mu.Unlock()
	return d.index()
}

func (d *DomainUpDownWidget) SelectedItem() string {
	mu.Lock()
	defer mu.Unlock()
	if i := d.index(); i >= 0 {
		return d.items[i]
	}
	return ""
}

// SetSelectedIndex selects item i, or the nearest end if i is out of range,
// without running OnChange.
func (d *DomainUpDownWidget) SetSelectedIndex(i int) *DomainUpDownWidget {
	d.change(func() { d.setIndex(i) })
	return d
}

// SetSelectedItem selects the item with this text, without running
// OnChange. A text that is not an item changes nothing.
func (d *DomainUpDownWidget) SetSelectedItem(s string) *DomainUpDownWidget {
	d.change(func() {
		if i := slices.Index(d.items, s); i >= 0 {
			d.setIndex(i)
		}
	})
	return d
}

// SetItems replaces the items. The selection stays at its index if the item
// there is the same, else it follows the selected item if that is among
// them, else the first is selected.
func (d *DomainUpDownWidget) SetItems(items ...string) *DomainUpDownWidget {
	d.change(func() {
		i, cur := d.index(), ""
		if i >= 0 {
			cur = d.items[i]
		}
		d.items = slices.Clone(items)
		if i >= 0 && i < len(d.items) && d.items[i] == cur {
			d.setIndex(i) // the item at its index is the same
			return
		}
		d.setIndex(max(slices.Index(d.items, cur), 0))
	})
	return d
}

// SetWrap makes Up at the first item lead to the last, and Down at the last
// to the first.
func (d *DomainUpDownWidget) SetWrap(wrap bool) *DomainUpDownWidget {
	d.change(func() { d.wrap = wrap })
	return d
}

// OnChange runs fn, on the event goroutine, after the user chose another
// item. A setter does not run it.
func (d *DomainUpDownWidget) OnChange(fn func()) *DomainUpDownWidget {
	d.change(func() { d.onChange = fn })
	return d
}

func (d *DomainUpDownWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	p, _ := d.pos.bind()
	n := float64(len(d.items))
	lo, hi := 0.0, max(n-1, 0)
	if d.wrap {
		lo, hi = -1, n
	}
	text := ""
	if i := d.index(); i >= 0 {
		text = d.items[i]
	}
	return ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).MinWidth(t.Space(20)).Padding(t.Space(1.5), t.Space(2)).Border(1, t.Border).Radius(t.Radius).Children(func() {
			ui.Text(c, text)
		})
		ui.Stepper(c, p, lo, hi, 1).Disabled(n == 0).Value(text).OnChange(d.edited())
	})
}

// edited returns ui's OnChange callback: it takes what the user chose, turns
// a step past an end into the other end, and runs OnChange and the binding's
// write on the event goroutine. Under mu, while building.
func (d *DomainUpDownWidget) edited() func() {
	return func() {
		mu.Lock()
		old := d.index()
		s := d.wrapped(d.pos.shown)
		d.pos.value, d.pos.shown, d.pos.dirty = s, s, false
		if d.index() == old {
			mu.Unlock()
			return
		}
		item := ""
		if i := d.index(); i >= 0 {
			item = d.items[i]
		}
		h, write := d.onChange, d.bound.write
		mu.Unlock()
		Post(func() {
			if h != nil {
				h()
			}
			if write != nil {
				write(item)
			}
		})
	}
}

func (d *DomainUpDownWidget) bindValue() string  { return d.SelectedItem() }
func (d *DomainUpDownWidget) bindApply(v string) { d.SetSelectedItem(v) }
