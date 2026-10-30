package ui

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"testing"
	"time"
)

// A pass keeps the key of a row for the places that ask again, until the app's
// code runs (a row's callback): these are the cases where the data changes in
// a row's build, and a key kept across it would be the wrong one.
type lkItem struct {
	id      int
	on      bool
	expired bool
}

func lkHash(tt *Tester) string {
	h := sha256.Sum256(tt.Image().Pix)
	return fmt.Sprintf("%x", h[:6])
}

func lkSel(sel *Selection[int]) []int {
	var ks []int
	for k := range 40 {
		if sel.Has(k) {
			ks = append(ks, k)
		}
	}
	return ks
}

// lkSetup builds a list of 12 items, each row a name, a Switch and a Del
// button; mode chooses the ListState's fields.
func lkSetup(mode string, items *[]lkItem, sel *Selection[int], cur *int, keyCalls *int) (*Tester, *ListState) {
	s := &ListState{Key: func(i int) any {
		*keyCalls++
		if i >= len(*items) {
			return -1 - i
		}
		return (*items)[i].id
	}}
	if mode != "noversion" {
		// A Switch's row is kept once its knob rests, so the data a row acts on
		// is versioned, as RowVersion asks: an item that is to expire changes it.
		s.RowVersion = func(i int) uint64 {
			if i < len(*items) && (*items)[i].expired {
				return 1
			}
			return 0
		}
	}
	s.Selection = sel
	if cur != nil {
		s.Selected = cur
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreList(c, s, len(*items), func(i int) {
				if i >= len(*items) {
					return
				}
				it := &(*items)[i]
				if it.expired {
					// The row drops its item as it builds, with no event.
					*items = slices.Delete(*items, i, i+1)
					return
				}
				coreRow(c).Children(func() {
					coreText(c, fmt.Sprint("item", it.id))
					coreSwitch(c, &it.on)
					if coreButton(c, fmt.Sprint("Del", it.id)).Clicked() {
						*items = slices.Delete(*items, i, i+1)
					}
				})
			}).Grow(1)
		})
	}, 400, 600)
	return tt, s
}

func lkItems() []lkItem {
	items := make([]lkItem, 12)
	for i := range items {
		items[i] = lkItem{id: i, on: i%2 == 0}
	}
	return items
}

// The picture 70 ms after a row's Del button deleted its item in the build
// (click), or after a row dropped its item as it built (expire), equals that of
// a list that never had the item: no Switch moves.
func TestListKeySwitchMidway(t *testing.T) {
	for _, how := range []string{"click", "expire"} {
		for _, mode := range []string{"version", "noversion"} {
			t.Run(how+"/"+mode, func(t *testing.T) {
				items := lkItems()
				var sel Selection[int]
				calls := 0
				tt, _ := lkSetup(mode, &items, &sel, nil, &calls)
				for range 3 {
					tt.Frame()
				}
				if err := tt.Click("item3"); err != nil {
					t.Fatal(err)
				}
				if err := tt.ClickWith(Shift, "item4"); err != nil {
					t.Fatal(err)
				}
				tt.Move(399, 599)
				for range 3 {
					tt.Frame()
				}
				if how == "click" {
					if err := tt.Click("Del2"); err != nil {
						t.Fatal(err)
					}
					tt.Move(399, 599)
				} else {
					tt.Update(func() { items[2].expired = true })
				}
				tt.Frame()
				tt.Advance(70 * time.Millisecond)
				tt.Frame()

				items2 := slices.Clone(items)
				var sel2 Selection[int]
				for _, k := range lkSel(&sel) {
					sel2.set(k, true)
				}
				calls2 := 0
				tt2, s2 := lkSetup(mode, &items2, &sel2, nil, &calls2)
				s2.lead = tt.rt.c.root.first.first.list.s.lead
				tt2.Move(399, 599)
				for range 3 {
					tt2.Frame()
				}
				tt2.Advance(70 * time.Millisecond)
				tt2.Frame()
				if d := sameFrames(tt, tt2, true); d != "" {
					t.Errorf("70 ms after: %s", d)
				}
				if tt.rt.animating {
					t.Errorf("70 ms after: still animating")
				}
			})
		}
	}
}

// A row drops its expired item as it builds (no event: the frame's only pass,
// whose rows are kept), and the app adds an item at the top before the next
// frame: the rows after the dropped one are back at the indexes they were
// kept at, with the same key and version. They must show their own items.
func TestListKeyKeptRowAfterExpireAndPrepend(t *testing.T) {
	for _, mode := range []string{"selection", "selected", "none"} {
		t.Run(mode, func(t *testing.T) {
			items := lkItems()
			next := 100
			var sel Selection[int]
			cur := -1
			s := &ListState{
				Key: func(i int) any {
					if i >= len(items) {
						return -1 - i
					}
					return items[i].id
				},
				RowVersion: func(i int) uint64 { return 0 },
			}
			switch mode {
			case "selection":
				s.Selection = &sel
			case "selected":
				s.Selected = &cur
			}
			tt := coreNewTester(func(c *context) {
				coreColumn(c).Fill().Children(func() {
					coreList(c, s, len(items), func(i int) {
						if i >= len(items) {
							return
						}
						if items[i].expired {
							items = slices.Delete(items, i, i+1)
							return
						}
						coreText(c, fmt.Sprint("item", items[i].id))
					}).Grow(1)
				})
			}, 400, 600)
			for range 3 {
				tt.Frame()
			}
			switch mode {
			case "selection":
				if err := tt.Click("item3"); err != nil {
					t.Fatal(err)
				}
				if err := tt.ClickWith(Shift, "item4"); err != nil {
					t.Fatal(err)
				}
			case "selected":
				if err := tt.Click("item4"); err != nil {
					t.Fatal(err)
				}
			}
			tt.Move(399, 599)
			for range 3 {
				tt.Frame()
			}
			dump := func(what string) {
				var b []string
				for _, r := range s.frame.rows {
					txt := ""
					var walk func(e *node)
					walk = func(e *node) {
						if e.text != "" {
							txt += e.text
						}
						for c := e.first; c != nil; c = c.next {
							walk(c)
						}
					}
					walk(r.e)
					b = append(b, fmt.Sprintf("%d:%v=%s", r.i, r.key, txt))
				}
				t.Logf("%s: kept=%d rows %v", what, tt.rt.rowsKept, b)
			}
			dump("before")
			// The row of item 2 is built (the rows moved), and drops it.
			tt.Update(func() {
				items[2].expired = true
				items = slices.Insert(items, 0, lkItem{id: next})
			})
			tt.Frame()
			dump("expire+prepend")
			tt.Update(func() { items = slices.Insert(items, 0, lkItem{id: next + 1}) })
			tt.Frame()
			dump("prepend again")
			tt.Frame()
			dump("next")
			want := []string{}
			for _, it := range items {
				want = append(want, fmt.Sprint("item", it.id))
			}
			got := []string{}
			for _, s := range tt.Texts() {
				if len(s) > 4 && s[:4] == "item" {
					got = append(got, s)
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("rows show %v,\nwant %v (kept %d)", got, want, tt.rt.rowsKept)
			}
		})
	}
}

// One ListState shown in two windows whose frames have the same number: an
// item deleted between their frames.
func TestListKeyTwoWindows(t *testing.T) {
	items := lkItems()
	var sel Selection[int]
	sel.set(3, true)
	s := &ListState{
		Key: func(i int) any {
			if i >= len(items) {
				return -1 - i
			}
			return items[i].id
		},
		Selection: &sel,
	}
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreList(c, s, len(items), func(i int) { coreText(c, fmt.Sprint("item", items[i].id)) }).Grow(1)
		})
	}
	a := coreNewTester(view, 400, 600)
	b := coreNewTester(view, 400, 600)
	for range 3 {
		a.Frame()
		b.Frame()
	}
	a.Frame()
	items = slices.Delete(items, 0, 1) // between frames, as an Update of the app does
	b.Frame()
	if a.rt.frame != b.rt.frame {
		t.Logf("frames %d and %d", a.rt.frame, b.rt.frame)
	}
	for _, r := range s.frame.rows {
		txt := ""
		for c := r.e; c != nil && txt == ""; c = c.first {
			txt = c.text
		}
		if want := fmt.Sprint("item", r.key); txt != want {
			t.Errorf("window b: row %d has key %v and shows %s", r.i, r.key, txt)
		}
	}
}

// The choice around a row reads its neighbours' keys as the pass asked them
// (peek), which may be from before a row's callback changed the data: the
// corners of a chosen row can be wrong for the frame of the change, and the
// frame after must show what a list built from the new data shows, with no row
// linked with the choice of the old one.
func TestListKeyPeekHealsInTheNextFrame(t *testing.T) {
	for _, mode := range []string{"version", "version+selected"} {
		t.Run(mode, func(t *testing.T) {
			items := lkItems()
			var sel Selection[int]
			var cur *int
			if mode == "version+selected" {
				v := -1
				cur = &v
			}
			calls := 0
			tt, s := lkSetup(mode, &items, &sel, cur, &calls)
			for range 3 {
				tt.Frame()
			}
			// One chosen row below the row that expires: it moves up by one, and
			// the neighbours' keys read before the row above built are of the
			// rows it had.
			if err := tt.Click("item3"); err != nil {
				t.Fatal(err)
			}
			tt.Move(399, 599)
			for range 3 {
				tt.Frame()
			}
			f := &s.frame
			// What the rows record of the choice around them is what the list
			// reads now (outside a pass, where nothing is kept).
			recorded := func(when string) {
				t.Helper()
				for _, r := range f.rows {
					if want := f.chosenAround(r.i); r.chosen != want && !f.isHeader(r.i) {
						t.Errorf("%s: row %d (%v) is recorded with the choice %v around it, the list reads %v", when, r.i, r.key, r.chosen, want)
					}
				}
				for _, st := range tt.rt.states {
					if rec := st.krec; rec != nil && rec.root != nil {
						if want := f.chosenAround(rec.root.rowIndex); rec.chosen != want && !f.isHeader(rec.root.rowIndex) {
							t.Errorf("%s: the kept record of row %d has the choice %v around it, the list reads %v", when, rec.root.rowIndex, rec.chosen, want)
						}
					}
				}
			}
			tt.Update(func() { items[2].expired = true })
			tt.Frame() // the frame of the change: no event, one pass
			tt.Frame() // the next
			recorded("the frame after a change made in a build")
			// And an item added at the top between frames: every key moves, and
			// the answers kept in the frame before must not be read.
			tt.Update(func() { items = slices.Insert(items, 0, lkItem{id: 100}) })
			tt.Frame()
			recorded("the frame after a change between frames")
			tt.Advance(time.Second)
			tt.Frame()

			items2 := slices.Clone(items)
			var sel2 Selection[int]
			for _, k := range lkSel(&sel) {
				sel2.set(k, true)
			}
			var cur2 *int
			if cur != nil {
				v := *cur
				cur2 = &v
			}
			calls2 := 0
			tt2, s2 := lkSetup(mode, &items2, &sel2, cur2, &calls2)
			s2.lead = s.lead
			tt2.Move(399, 599)
			tt2.Advance(time.Second)
			for range 3 {
				tt2.Frame()
			}
			if d := sameFrames(tt, tt2, true); d != "" {
				t.Errorf("the frame after the changes differs from a list built from the new data: %s", d)
			}
		})
	}
}
