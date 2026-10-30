package ui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// The Modal is closed while the rows are first built (Modal returns no element:
// the rows ask Dismissed of nothing), then it opens, and Escape dismisses it:
// the rows that ask must see it, in the twin that keeps rows as in the other.
func TestRetainRowsAskDismissedOfAModalThatOpensLater(t *testing.T) {
	p := newPair(t, 400, 400, func() func(c *Context) {
		open := false
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				if Button(c, "open").Clicked() {
					open = true
				}
				m := Modal(c, &open, func() { Textf(c, "dialog") })
				List(c, s, 30, func(i int) {
					if m.Dismissed() {
						countSeen(c)
					}
					Textf(c, "row %d", i)
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	p.both(func(tt *Tester) { _ = tt.Click("open") })
	p.idle(3)
	seenKept.Store(0)
	seenBuilt.Store(0)
	p.both(func(tt *Tester) { tt.Key(0, KeyEscape) })
	p.compare("escape")
	p.frame("after")
	if seenKept.Load() != seenBuilt.Load() || seenBuilt.Load() == 0 {
		t.Errorf("rows that saw Dismissed: %d with kept rows, %d with every row built", seenKept.Load(), seenBuilt.Load())
	}
}

// The item of a select is made in the callback of its popup, which does not run
// while the popup is closed: the app's handle is Element{} until it opens, and
// the rows that ask Highlighted of it must follow the keys.
func TestRetainRowsAskHighlightedOfAnItemThatExistsLater(t *testing.T) {
	p := newPair(t, 400, 400, func() func(c *Context) {
		size := "M"
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		var l Element
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				sel := SelectBase(c, &size)
				sel.Trigger.Label("size").Padding(4).Children(func() { Textf(c, "Size: %s", size) })
				sel.Popup(func(panel Element) {
					for _, v := range []string{"S", "M", "L", "XL"} {
						it := sel.Item(v).Padding(4).Label("item " + v)
						if v == "L" {
							l = it
						}
						it.Children(func() { Textf(c, "%s", v) })
					}
				})
				List(c, s, 30, func(i int) {
					if l.Highlighted() {
						countSeen(c)
					}
					Textf(c, "row %d", i)
				}).Grow(1)
			})
		}
	})
	p.idle(3)
	p.both(func(tt *Tester) { _ = tt.Click("size") })
	p.idle(3)
	seenKept.Store(0)
	seenBuilt.Store(0)
	p.both(func(tt *Tester) { tt.Key(0, KeyDown) })
	p.compare("down")
	p.frame("after")
	if seenKept.Load() != seenBuilt.Load() {
		t.Errorf("rows that saw Highlighted: %d with kept rows, %d with every row built", seenKept.Load(), seenBuilt.Load())
	}
}

// buildingRow is the row whose callback runs, in every way a row is built, and
// nil outside one: a row inside a row puts the outer one back.
func TestBuildingRowIsSetAroundEveryRow(t *testing.T) {
	var outerSeen, innerSeen, afterInner, lateSeen, outside int
	var outerRB *rowBuild
	p := newPair(t, 300, 400, func() func(c *Context) {
		outer := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		inner := make([]*ListState, 40)
		for i := range inner {
			inner[i] = &ListState{Key: func(j int) any { return j }, RowVersion: func(int) uint64 { return 0 }}
		}
		late := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			if buildingRow != nil {
				outside++
			}
			Row(c).Fill().Children(func() {
				List(c, outer, 20, func(i int) {
					rb := buildingRow
					if rb == nil {
						return
					}
					outerSeen++
					outerRB = rb
					Column(c).Height(60).Children(func() {
						List(c, inner[i], 2, func(j int) {
							if buildingRow != nil && buildingRow != outerRB && buildingRow.outer == outerRB {
								innerSeen++
							}
						}).Height(40)
					})
					if buildingRow == outerRB {
						afterInner++ // the outer row is the one being built again
					}
				}).Width(150)
				// Rows of different heights, which the list builds as it lays out.
				List(c, late, 200, func(i int) {
					if buildingRow != nil {
						lateSeen++
					}
					Row(c).Height(float32(15 + (i%4)*25))
				}).Width(100)
			})
		}
	})
	p.idle(3)
	if outside != 0 {
		t.Errorf("buildingRow was set outside the rows %d times", outside)
	}
	if outerSeen == 0 || innerSeen == 0 || afterInner != outerSeen || lateSeen == 0 {
		t.Errorf("buildingRow: outer rows %d, inner rows %d, outer back after inner %d, late rows %d", outerSeen, innerSeen, afterInner, lateSeen)
	}
	if buildingRow != nil {
		t.Errorf("buildingRow is still set after the frames")
	}
	noteMissingElement() // outside a build: nothing to mark
}

// Every query of Element answers an element that is not there by noting it for
// the row being built: the generated code of api_gen.go, not a list here.
func TestEveryQueryOfAMissingElementNotesIt(t *testing.T) {
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, "api_gen.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	queries, noted := 0, 0
	for _, d := range af.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Body == nil || len(fd.Body.List) < 2 {
			continue
		}
		if id, ok := fd.Recv.List[0].Type.(*ast.Ident); !ok || id.Name != "Element" {
			continue
		}
		// _node := _handle.node(); if _node == nil { ...; return ... }
		asg, ok := fd.Body.List[0].(*ast.AssignStmt)
		if !ok || len(asg.Lhs) != 1 || fmt.Sprint(asg.Lhs[0]) != "_node" {
			continue
		}
		ifs, ok := fd.Body.List[1].(*ast.IfStmt)
		if !ok {
			continue
		}
		builder := false
		if r := fd.Type.Results; r != nil {
			for _, f := range r.List {
				if id, ok := f.Type.(*ast.Ident); ok && id.Name == "Element" {
					builder = true
				}
			}
		}
		if fd.Type.Results == nil || builder {
			continue
		}
		queries++
		found := false
		ast.Inspect(ifs.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "noteMissingElement" {
					found = true
				}
			}
			return true
		})
		if found {
			noted++
		} else {
			t.Errorf("Element.%s answers a missing element without noteMissingElement", fd.Name.Name)
		}
	}
	if queries < 40 {
		t.Errorf("found only %d queries in api_gen.go: the test looks at the wrong code", queries)
	}
	t.Logf("%d queries, %d note it", queries, noted)
}
