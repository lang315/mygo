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

func keySel(sel *Selection[int]) []int {
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
		s.RowVersion = func(i int) uint64 { return 0 }
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
				for _, k := range keySel(&sel) {
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
