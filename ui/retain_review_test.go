package ui

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// The tests of this file are the cases the whole-branch review found: each
// failed before its fix.

// statesUnder returns the states of e and the elements under it.
func statesUnder(e *node) (out []*state) {
	if e.st != nil {
		out = append(out, e.st)
	}
	for c := e.first; c != nil; c = c.next {
		out = append(out, statesUnder(c)...)
	}
	return
}

// danglingStates counts the states of the rows a list linked that the engine
// does not hold (prune recycled them).
func danglingStates(rt *engine, list *ListState) (bad int) {
	for _, r := range list.frame.rows {
		for _, st := range statesUnder(r.e) {
			if rt.states[st.id] != st {
				bad++
			}
		}
	}
	return
}

// A list built in the first pass of a frame and not in the last (a handler
// after it hid it) had its states pruned: when it shows again its rows are
// built, not linked with states that were recycled.
func TestRetainAListLeftOutOfTheLastPassBuildsItsRowsAgain(t *testing.T) {
	vers := make([]uint64, 20)
	list := ListState{
		Key:        func(i int) any { return i },
		RowVersion: func(i int) uint64 { return vers[i] },
	}
	omit, flip := false, false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Children(func() {
			if !omit {
				coreList(c, &list, 20, func(i int) {
					coreText(c, fmt.Sprint("row", i))
				}).Grow(1)
			}
			if flip && c.rt.pass == 0 {
				// As a handler built after the list does: it hides the list, and
				// the engine builds again.
				flip, omit = false, true
				c.rt.consumed = true
			}
		})
	}, 400, 600)
	for range 4 {
		tt.Frame()
	}
	flip = true
	tt.Frame()
	omit = false
	tt.Frame()
	if bad := danglingStates(tt.rt, &list); bad > 0 {
		t.Errorf("%d states of the rows linked are not the engine's (prune recycled them)", bad)
	}
}

// Rows linked in the first pass of a frame that the last pass scrolled out of
// view had their states pruned too.
func TestRetainRowsScrolledOutInTheLastPassAreBuiltWhenTheyComeBack(t *testing.T) {
	const n = 1000
	vers := make([]uint64, n)
	list := ListState{
		Key:        func(i int) any { return i },
		RowVersion: func(i int) uint64 { return vers[i] },
	}
	jump := -1
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Children(func() {
			coreList(c, &list, n, func(i int) {
				coreText(c, fmt.Sprint("row", i))
			}).Grow(1)
			if jump >= 0 && c.rt.pass == 0 {
				list.ScrollTo(jump, Start)
				jump = -1
				c.rt.consumed = true
			}
		})
	}, 400, 300)
	for range 4 {
		tt.Frame()
	}
	jump = 900
	tt.Frame()
	jump = 0
	tt.Frame()
	if bad := danglingStates(tt.rt, &list); bad > 0 {
		t.Errorf("%d states of the rows linked are not the engine's (prune recycled them)", bad)
	}
}

// The key that renames a row of a Table, and the one that opens it, go to a
// row whose EditableText was kept as to one that was built.
func TestRetainTheKeyThatRenamesARowRenamesIt(t *testing.T) {
	for _, keep := range []bool{false, true} {
		names := []string{"notes.txt", "photo.jpeg", "README"}
		vers := make([]uint64, len(names))
		sel, opened := -1, -1
		s := ListState{Selected: &sel, Key: func(i int) any { return names[i] }, RowVersion: func(i int) uint64 { return vers[i] }}
		cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80}}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Padding(10).Children(func() {
				if coreTable(c, &s, cols, len(names), func(row, col int) {
					if col == 0 {
						if coreEditableText(c, &names[row]).Changed() {
							vers[row]++
						}
					} else {
						coreText(c, "1 KB")
					}
				}).Grow(1).Submitted() {
					opened = sel
				}
			})
		}, 400, 300)
		tt.rt.noRetain = !keep
		tt.Click("photo.jpeg")
		tt.Move(390, 290) // off the rows
		for range 3 {
			tt.Frame()
		}
		if runtime.GOOS == "darwin" {
			tt.Key(0, KeyEnter)
		} else {
			tt.Key(0, KeyF2)
		}
		tt.Type("beach")
		tt.Key(0, KeyEnter)
		if names[1] != "beach.jpeg" || opened != -1 {
			t.Errorf("keep=%v: the key that renames left %q and opened row %d", keep, names[1], opened)
		}
	}
}

// A cell that paints by its Element is valid until the next pass: a row with
// such a closure is built, as it was written.
func TestRetainARowThatDrawsByItsElementIsNeverKept(t *testing.T) {
	vers := make([]uint64, 5)
	list := ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return vers[i] }}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreList(c, &list, 5, func(i int) {
				el := wrapElement(coreBox(c).Size(20, 20))
				el.Draw(func(p *Painter, r Rect) {
					if el.Hovered() {
						p.Fill(r, Color{R: 1, A: 1}, 0)
					}
				})
			}).Grow(1)
		})
	}, 400, 600)
	for range 4 {
		tt.Frame()
	}
	if tt.rt.rowsKept != 0 {
		t.Errorf("%d rows that draw by their Element were kept", tt.rt.rowsKept)
	}
}

// The item chosen from a row's context menu runs for that row when the row is
// built, which it is once the choice waits for it.
func TestRetainAContextMenuChoiceOnAKeptRowRuns(t *testing.T) {
	for _, keep := range []bool{false, true} {
		vers := make([]uint64, 10)
		deleted := -1
		list := ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return vers[i] }}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				coreList(c, &list, 10, func(i int) {
					coreText(c, fmt.Sprint("row", i)).ContextMenu(func(m *Menu) {
						if m.Item("Delete").Chosen() {
							deleted = i
						}
					})
				}).Grow(1)
			})
		}, 400, 600)
		tt.rt.noRetain = !keep
		for range 3 {
			tt.Frame()
		}
		if err := tt.RightClick("row3"); err != nil {
			t.Fatal(err)
		}
		tt.Move(399, 599)
		tt.Frame()
		if err := tt.ChooseMenuItem("Delete"); err != nil {
			t.Fatal(err)
		}
		if deleted != 3 {
			t.Errorf("keep=%v: the item chosen deleted %d", keep, deleted)
		}
	}
}

// A row that asks for frames (AnimationFrame) or for a build later (After) is
// built when they come.
func TestRetainARowThatAsksForFramesIsBuiltWhenTheyCome(t *testing.T) {
	for _, after := range []bool{false, true} {
		vers := make([]uint64, 5)
		builds := 0
		list := ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 { return vers[i] }}
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Fill().Children(func() {
				coreList(c, &list, 5, func(i int) {
					if i == 0 {
						builds++
						coreText(c, fmt.Sprint(c.Now().UnixMilli()))
						if after {
							c.After(100 * time.Millisecond)
						} else {
							c.AnimationFrame()
						}
						return
					}
					coreText(c, fmt.Sprint("row", i))
				}).Grow(1)
			})
		}, 400, 600)
		tt.Frame()
		b := builds
		tt.Advance(time.Second)
		if builds-b < 2 {
			t.Errorf("after=%v: the row built %d times in a second", after, builds-b)
		}
	}
}

// RowVersion set to nil again leaves the outline as it was without one.
func TestRetainAnOutlineWhoseRowVersionIsNilAgainBuilds(t *testing.T) {
	var s OutlineState[int]
	s.RowVersion = func(int) uint64 { return 0 }
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreOutline(c, &s, []int{0, 1}, func(k int) []int { return nil }, func(k int) {
				coreText(c, fmt.Sprint("item", k))
			}).Grow(1)
		})
	}, 400, 600)
	tt.Frame()
	s.RowVersion = nil
	tt.Frame()
	tt.Frame()
}

// What a list does with its rows besides their data (a choice that can be
// made) is part of what a kept row was built with.
func TestRetainAListThatGetsAChoiceBuildsItsRows(t *testing.T) {
	vers := make([]uint64, 20)
	var list ListState
	list.Key = func(i int) any { return i }
	list.RowVersion = func(i int) uint64 { return vers[i] }
	sel := -1
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Children(func() {
			coreList(c, &list, 20, func(i int) {
				coreText(c, fmt.Sprint("row", i))
			}).Grow(1)
		})
	}, 400, 600)
	for range 4 {
		tt.Frame()
	}
	list.Selected = &sel
	b := tt.rt.rowsBuilt
	tt.Frame()
	if tt.rt.rowsBuilt == b {
		t.Errorf("the list got a choice and no row was built")
	}
}
