package ui

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"sync"
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

// Rows built as the list lays out (their heights differ, so it builds them late)
// are marked like the others when they ask a missing element, and those that do
// not ask are kept.
func TestRowsBuiltLateAskingAMissingElementAreBuilt(t *testing.T) {
	for _, asks := range []bool{true, false} {
		name := "rows that do not ask"
		if asks {
			name = "rows that ask"
		}
		t.Run(name, func(t *testing.T) {
			p := newPair(t, 300, 400, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					var nothing Element
					Column(c).Fill().Children(func() {
						List(c, s, 200, func(i int) {
							if asks && nothing.Dismissed() {
								Textf(c, "never")
							}
							Row(c).Height(float32(15 + (i%4)*25)).Children(func() { Textf(c, "row %d", i) })
						}).Grow(1)
					})
				}
			})
			p.idle(5)
			built0, kept0 := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
			p.frame("idle")
			b, k := p.kept.rt.rowsBuilt-built0, p.kept.rt.rowsKept-kept0
			if asks && b == 0 || !asks && (b != 0 || k == 0) {
				t.Errorf("idle: %d rows built, %d kept (asks a missing element: %v)", b, k, asks)
			}
		})
	}
}

// Windows built at the same time, on goroutines (a test of Testers does), share
// the count of queries of missing elements: the one whose rows ask must be built,
// and the other must still draw what it would without it. The count is atomic:
// run it with -race.
func TestTestersAskingAMissingElementInParallelAreRight(t *testing.T) {
	mk := func(asks bool) func() func(c *Context) {
		return func() func(c *Context) {
			s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
			return func(c *Context) {
				var nothing Element
				Column(c).Fill().Children(func() {
					List(c, s, 60, func(i int) {
						if asks && nothing.Dismissed() {
							Textf(c, "never")
						}
						Textf(c, "row %d", i)
					}).Grow(1)
				})
			}
		}
	}
	var wg sync.WaitGroup
	for _, asks := range []bool{true, false} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kept, built := NewTester(mk(asks)(), 300, 400), NewTester(mk(asks)(), 300, 400)
			built.rt.noRetain, built.rt.noReplay = true, true
			for range 40 {
				kept.Frame()
				built.Frame()
				if a, b := kept.Image(), built.Image(); !bytes.Equal(a.Pix, b.Pix) {
					t.Errorf("asks=%v: the twin that keeps rows draws something else", asks)
					return
				}
			}
			if asks && kept.rt.rowsKept != 0 {
				t.Errorf("rows that ask a missing element were kept %d times", kept.rt.rowsKept)
			}
		}()
	}
	wg.Wait()
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

// The queries of Element that are written by hand (not generated, so
// TestEveryQueryOfAMissingElementNotesIt does not see them) note a missing
// element too: they find it through readNode, or call noteMissingElement.
func TestHandWrittenQueriesOfElementNoteAMissingElement(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	found := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "api_gen.go" {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Body == nil || !fd.Name.IsExported() {
				continue
			}
			if id, ok := fd.Recv.List[0].Type.(*ast.Ident); !ok || id.Name != "Element" {
				continue
			}
			r := fd.Type.Results
			if r == nil || len(r.List) == 0 {
				continue
			}
			if id, ok := r.List[0].Type.(*ast.Ident); ok && id.Name == "Element" {
				continue // a builder
			}
			found++
			notes := false
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					switch fn := c.Fun.(type) {
					case *ast.Ident:
						notes = notes || fn.Name == "noteMissingElement"
					case *ast.SelectorExpr:
						notes = notes || fn.Sel.Name == "readNode"
					}
				}
				return true
			})
			if !notes {
				t.Errorf("%s: Element.%s is written by hand and does not note a missing element (readNode)", f, fd.Name.Name)
			}
		}
	}
	if found < 5 {
		t.Errorf("found %d hand-written queries: the test looks at the wrong code", found)
	}
}

// ListRow.Selected asks the list, which is outside the row, whether it is built:
// the answer must not make every row built in every frame.
func TestRetainRowsShowingSelectedStayKept(t *testing.T) {
	p := newPair(t, 400, 400, func() func(c *Context) {
		sel := 1
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Selected: &sel}
		return func(c *Context) {
			List(c, s, 20).Rows(func(r ListRow) {
				Textf(c, "row %d sel=%v", r.Index, r.Selected())
			}).Grow(1)
		}
	})
	p.idle(3)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
		t.Errorf("idle: %d rows built, %d kept; want none built", b, k)
	}
}

// Valid of a Modal asked from rows: closed, open, closed again. The rows follow it.
func TestRetainRowsAskValidOfAModalThatOpensAndCloses(t *testing.T) {
	var opens []*bool
	p := newPair(t, 400, 400, func() func(c *Context) {
		open := new(bool)
		opens = append(opens, open)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				m := Modal(c, open, func() { Textf(c, "dialog") })
				List(c, s, 10, func(i int) {
					Textf(c, "row %d open=%v", i, m.Valid())
				}).Grow(1)
			})
		}
	})
	set := func(v bool) {
		for _, o := range opens {
			*o = v
		}
	}
	p.idle(3)
	set(true)
	p.idle(3)
	if !p.kept.HasText("row 0 open=true") {
		t.Errorf("open: the rows still say closed")
	}
	set(false)
	p.idle(3)
	if !p.kept.HasText("row 0 open=false") {
		t.Errorf("closed again: the rows still say open")
	}
}

// Valid of an element outside the row that stays there is a question of the
// element: the row is built while it asks (asksOutside).
func TestRetainRowsAskValidOfAnElementOutside(t *testing.T) {
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				head := Row(c).Height(40)
				List(c, s, 10, func(i int) {
					Textf(c, "row %d valid=%v", i, head.Valid())
				}).Grow(1)
			})
		}
	})
	p.idle(4)
	built := p.kept.rt.rowsBuilt
	p.frame("idle")
	if p.kept.rt.rowsBuilt == built {
		t.Errorf("rows that ask Valid of an outside element were kept")
	}
}
