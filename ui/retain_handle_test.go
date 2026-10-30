package ui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// The queries of a Handle name an element of the window that may not be built in
// this pass: a row that asks (here whether a field outside the list has the
// focus, and where it is) is built, and shows what a build shows.
func TestRetainRowsAskingAHandleAreBuilt(t *testing.T) {
	for _, tc := range []struct {
		name string
		ask  func(c *Context, h *Handle) string
	}{
		{"Focused", func(c *Context, h *Handle) string { return fmt.Sprint(h.Focused(c)) }},
		{"FocusWithin", func(c *Context, h *Handle) string { return fmt.Sprint(h.FocusWithin(c)) }},
		{"Bounds", func(c *Context, h *Handle) string { return fmt.Sprint(h.Bounds(c).H) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 400, 400, func() func(c *Context) {
				var text string
				var field Handle
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						TextInput(c, &text).Bind(&field)
						List(c, s, 30, func(i int) {
							Textf(c, "row %d %s", i, tc.ask(c, &field))
						}).Grow(1)
					})
				}
			})
			p.idle(3)
			built := p.kept.rt.rowsBuilt
			p.frame("idle")
			if p.kept.rt.rowsBuilt == built {
				t.Errorf("rows that ask a Handle were kept")
			}
			p.both(func(tt *Tester) { tt.ClickAt(50, 15) })
			p.frame("focus")
			p.frame("focus 2")
			p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
			p.frame("tab")
			p.frame("tab 2")
		})
	}
}

// OnShortcut of a Handle declares the action at each build, as Context.Shortcut
// registers its shortcut: a row that is linked declares none, so the shortcut
// would not run.
func TestRetainRowsDeclaringAHandleShortcutAreBuilt(t *testing.T) {
	var runs [2]int
	var field Handle
	p := newPair(t, 400, 400, func() func(c *Context) {
		var text string
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				TextInput(c, &text).Bind(&field)
				List(c, s, 10, func(i int) {
					if i == 0 {
						n := 0
						if c.rt.noRetain {
							n = 1
						}
						field.OnShortcut(c, 0, KeyF5, func() { runs[n]++ })
					}
					Textf(c, "row %d", i)
				}).Grow(1)
			})
		}
	})
	p.idle(4)
	p.both(func(tt *Tester) { tt.ClickAt(50, 15) }) // the field takes the focus
	p.idle(2)
	runs = [2]int{}
	p.both(func(tt *Tester) { tt.Key(0, KeyF5) })
	if runs[0] != runs[1] || runs[1] == 0 {
		t.Errorf("F5 ran %d times with kept rows, %d with every row built", runs[0], runs[1])
	}
}

// A row of the list inside a row shows ListFocused of the list around it: it
// compares its own list's focus when it is kept, so it is built while it asks of
// another.
func TestRetainRowsOfAnInnerListShowingTheOuterFocus(t *testing.T) {
	p := newPair(t, 400, 400, func() func(c *Context) {
		mk := func(n int) *ListState {
			sel := 0
			_ = n
			return &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }, Selected: &sel}
		}
		so, si := mk(4), mk(3)
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, so, 4).Rows(func(r ListRow) {
					if r.Index == 0 {
						Column(c).Height(120).Children(func() {
							List(c, si, 3).Rows(func(ir ListRow) {
								Textf(c, "inner %d outer=%v", ir.Index, r.ListFocused())
							}).Grow(1)
						})
						return
					}
					Textf(c, "outer %d", r.Index)
				}).Grow(1)
			})
		}
	})
	p.idle(4)
	p.both(func(tt *Tester) { tt.ClickAt(50, 140) }) // outer row 1: the outer list takes the focus
	p.compare("click on outer row 1")
	p.frame("after 1")
	p.frame("after 2")
}

// The methods of a Handle that read the window note it (readsHandle), or are in
// the list with their reason. (Context.Resolve finds the element through
// Element.Valid, which notes it.)
func TestHandleMethodsNoteTheRead(t *testing.T) {
	notNoted := map[string]string{
		"Focus":        "requests focus for the next build: an action, not a read",
		"RequestFocus": "as Focus",
		"CancelFocus":  "as Focus",
	}
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, "ref.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, d := range af.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Body == nil || !fd.Name.IsExported() {
			continue
		}
		recv := fd.Recv.List[0].Type
		if st, ok := recv.(*ast.StarExpr); ok {
			recv = st.X
		}
		if id, ok := recv.(*ast.Ident); !ok || id.Name != "Handle" {
			continue
		}
		found[fd.Name.Name] = true
		notes := false
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if se, ok := c.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "readsHandle" {
					notes = true
				}
			}
			return true
		})
		if _, ok := notNoted[fd.Name.Name]; !notes && !ok {
			t.Errorf("Handle.%s does not call readsHandle and is not in the list", fd.Name.Name)
		}
	}
	for name := range notNoted {
		if !found[name] {
			t.Errorf("%q is in the list but is not a method of Handle (any more)", name)
		}
	}
}
