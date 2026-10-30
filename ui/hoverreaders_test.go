package ui

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Every function of the package that reads what only the input says and a row
// built does not hold: the hover chain (rt.hover), the pointer's position
// (rt.pointerX, pointerY, pointerIn), the modifier keys held (rt.mods) or the drag (rt.drag), by the file and the
// function, and why a row that is kept while it is hovered does not depend on
// it. A row's callback that reads either to build something must note it
// (node.readsHover): Hovered and PointerPosition do, and they are the ways an
// app, and the widgets built on ui, can ask. A function that reads them and is
// not here fails TestHoverReadersAreKnown: decide whether it can run in the
// callback of a row, and if it can, note the read.
//
// Limit: the scan finds the functions that select these fields of a variable or
// field named rt, in the function itself. It does not follow calls (a function
// that reads through engine.isHovered or rowInteractive is not listed for it)
// nor an alias (r := rt; r.hover), so a reader that goes round both is not
// seen: ask for the hover in a build only through Hovered, PointerPosition and
// Modifiers, which note it.
var hoverReaders = map[string]string{
	// The ways to ask in a build: they note it.
	"input.go node.Hovered":         "notes the read (readsHover)",
	"input.go node.PointerPosition": "notes the read (readsHover)",
	"input.go context.readsMods":    "the note itself: it stores the modifier keys as the row reads them",
	"input.go context.Modifiers":    "notes the read (readsMods)",
	"input.go engine.modsChanged":   "an event",
	"dragdrop.go coreDragHover":     "notes the read (readsDrag): a row that asked is built while a drag lasts",
	"input.go node.Pressed":         "reads the position only while the element is pressed: a row with a press is built",
	// Option lists that move a highlight with the pointer: Hovered notes the read.
	"base.go selectParts.Item":       "asks Hovered, which notes the read",
	"combobox.go comboboxParts.Item": "asks Hovered, which notes the read",
	"popovermenu.go menuDraw.item":   "asks Hovered, which notes the read; a drawn menu is acting",
	// Read in an afterInput callback or while pressed, dragging or open: outside the row's callback, or a row built.
	"base.go coreSelectBase":                      "in an afterInput callback, when the select opens",
	"colorpicker.go coreColorPicker":              "in an afterInput callback, while pressed",
	"combobox.go comboboxBase":                    "in an afterInput callback",
	"indicators.go coreRangeSlider":               "while pressed",
	"value_input.go sliderInput":                  "while pressed",
	"editor.go editor.process":                    "while pressed and dragging",
	"cellselect.go listFrame.dragCells":           "a drag over the cells: the press is acting",
	"dragselect.go engine.scrollDragSelect":       "a drag-select: the press is acting",
	"dragselect.go listFrame.rowAtPointer":        "a drag-select: the press is acting",
	"list.go listFrame.reorder":                   "a drag of a row: the drag is acting",
	"gridview.go GridState.reorder":               "a drag of an item: the drag is acting",
	"dragdrop.go engine.dragEnd":                  "a drag: acting",
	"dragdrop.go engine.dragScroll":               "a drag: acting",
	"dragdrop.go engine.paintDrag":                "a drag, at paint",
	"data_drag.go engine.dataDragOver":            "an event",
	"menu.go engine.menuRelease":                  "an event; a menu is acting",
	"popovermenu.go engine.drawMenu":              "a menu drawn: acting",
	"popovermenu.go engine.openDrawnMenu":         "a menu asked by an event: acting",
	"textselection.go engine.scrollTextSelection": "a selection drag: the press is acting",
	"textselection.go engine.syncTextSelection":   "a selection drag: the press is acting",
	// The drag (rt.drag): its source and the element it is over are acting.
	"dragdrop.go coreDragOver":       "reads only the element it is over, which is acting",
	"dragdrop.go node.Drag":          "the source, which is acting",
	"dragdrop.go node.Dragging":      "notes the read (readsDrag): a source outside the row is not among the row's states",
	"dragdrop.go node.dragFrom":      "the source, which is acting",
	"dragdrop.go engine.dragCancel":  "an event",
	"dragdrop.go engine.dragMove":    "an event",
	"dragdrop.go engine.handPress":   "an event",
	"repaint.go engine.repaintParts": "no part is repainted while a drag lasts",
	"replay.go recordable":           "no row is replayed while a drag lasts",
	"retain.go listFrame.ok":         "the rule itself (a row that asked DragHover is not kept during a drag)",
	"runtime.go engine.close":        "the end of the window",
	"runtime.go engine.prune":        "after the frame",
	// Events, the cursor, the frame's own work and paint.
	"handler.go engine.deliver":       "an event",
	"input.go engine.event":           "an event",
	"input.go engine.pointerMove":     "an event",
	"input.go engine.pointerUp":       "an event",
	"input.go engine.pressMove":       "an event",
	"input.go engine.rehover":         "after the frame, from the hits",
	"input.go engine.scroll":          "an event",
	"input.go engine.scrollKey":       "an event",
	"input.go engine.setHover":        "after the frame, from the hits",
	"input.go engine.zoom":            "an event",
	"runtime.go engine.overScrollbar": "the cursor and the wheel",
	"runtime.go engine.runFrame":      "the frame's own work",
	"runtime.go engine.updateCursor":  "the cursor",
	"paint.go Painter.scrollbars":     "paint",
	"replay.go rowInteractive":        "paint: a hovered row paints anew",
	"retain.go listFrame.keep":        "the rule itself (the modifiers a row was built with)",
	"retain.go engine.isHovered":      "the rule itself",
	"retain.go engine.actingIDs":      "the rule itself",
	"tooltip.go engine.showTooltip":   "tooltips are acting",
	"webview.go engine.overWebView":   "a web view is not kept",
	"webview.go engine.webViewPress":  "an event",
	"inspector_details.go properties": "the inspector open: nothing is kept",
}

func hoverReadersFound(t *testing.T) map[string][]string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]bool{"hover": true, "pointerX": true, "pointerY": true, "pointerIn": true, "mods": true, "drag": true}
	found := map[string][]string{}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			name := fd.Name.Name
			if fd.Recv != nil && len(fd.Recv.List) > 0 {
				ty := fd.Recv.List[0].Type
				if s, ok := ty.(*ast.StarExpr); ok {
					ty = s.X
				}
				if ix, ok := ty.(*ast.IndexExpr); ok {
					ty = ix.X
				}
				if id, ok := ty.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				se, ok := n.(*ast.SelectorExpr)
				if !ok || !fields[se.Sel.Name] {
					return true
				}
				switch x := se.X.(type) {
				case *ast.Ident:
					if x.Name != "rt" {
						return true
					}
				case *ast.SelectorExpr:
					if x.Sel.Name != "rt" {
						return true
					}
				default:
					return true
				}
				found[f+" "+name] = append(found[f+" "+name], se.Sel.Name)
				return true
			})
		}
	}
	return found
}

func TestHoverReadersAreKnown(t *testing.T) {
	found := hoverReadersFound(t)
	var unknown, stale []string
	for k := range found {
		if _, ok := hoverReaders[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	for k := range hoverReaders {
		if _, ok := found[k]; !ok {
			stale = append(stale, k)
		}
	}
	slices.Sort(unknown)
	slices.Sort(stale)
	if len(unknown) > 0 {
		t.Errorf("functions that read the hover chain, the pointer or the modifier keys and are not in hoverReaders (can they run in the callback of a row? then note the read: readsHover, readsMods, readsDrag): %v", unknown)
	}
	if len(stale) > 0 {
		t.Errorf("hoverReaders lists functions that no longer read them: %v", stale)
	}
}

// Hovered, PointerPosition and Modifiers note the read: the call counts,
// whatever it returns.
func TestTheWaysToAskForTheInputNoteTheRead(t *testing.T) {
	// Each begins with a call that notes the read: of the hover and the
	// modifier keys (the row is built while hovered, or when they change), or
	// of an element outside the row (asksOutside, readsDrag).
	asks := []struct{ file, name, notes string }{
		{"input.go", "Hovered", "readsHover"}, {"input.go", "PointerPosition", "readsHover"},
		{"input.go", "Modifiers", "readsMods"}, {"dragdrop.go", "Dragging", "readsDrag"},
	}
	for _, n := range []string{"Pressed", "Clicked", "Clicks", "DoubleClicked", "RightClicked", "Focused", "FocusWithin", "Dragged", "DragVelocity", "DragEnded"} {
		asks = append(asks, struct{ file, name, notes string }{"input.go", n, "asksOutside"})
	}
	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	for _, w := range asks {
		if files[w.file] == nil {
			af, err := parser.ParseFile(fset, w.file, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			files[w.file] = af
		}
		noted := false
		for _, d := range files[w.file].Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Name.Name != w.name || fd.Recv == nil {
				continue
			}
			// The first statement, before any return.
			if len(fd.Body.List) > 0 {
				if es, ok := fd.Body.List[0].(*ast.ExprStmt); ok {
					if call, ok := es.X.(*ast.CallExpr); ok {
						if se, ok := call.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == w.notes {
							noted = true
						}
					}
				}
			}
		}
		if !noted {
			t.Errorf("%s does not begin with a call of %s", w.name, w.notes)
		}
	}
}

// elementQueriesNotNoted: the queries of Element that read the state of an
// element (e.st, the engine) and do not begin with a call that notes the read
// for the row being built (asksOutside, readsLayout, readsHover, readsDrag),
// and why a row that is kept does not need it. An element inside the row is
// covered by what keeps a row from being kept (state.pending, the acting ids,
// retainable); the call is for an element outside the row, and for the layout
// of the frame before. A query that is not here and does not note the read
// fails TestElementQueriesNoteTheRead: decide which kind it is.
var elementQueriesNotNoted = map[string]string{
	"element.go node.disabled":          "reads e.st.flags of the frame before, but every query that calls it notes the read first",
	"element.go node.hasState":          "only whether the element has a state: the engine cannot have given it another",
	"motion.go node.channel":            "finds the element's own animation for AnimateTo, Spring, Animate...: the row is built again because AnimateTo asks for frames (AnimationFrame bumps rt.bound), not because of the state; animating an outside element is not a read",
	"motion.go node.AnimateTo":          "drives the element's own value and asks for frames while it moves (AnimationFrame bumps rt.bound, which a list does not keep); animating an outside element from a row is not a read",
	"motion.go node.AnimateColor":       "as AnimateTo",
	"motion.go node.Jump":               "as AnimateTo",
	"motion.go node.Keyframes":          "as AnimateTo",
	"widgets.go node.Loop":              "as AnimateTo",
	"timeline.go node.Timeline":         "as AnimateTo: a row with a timeline is not kept",
	"rowclick.go node.RowDoubleClicked": "asks Hovered of the element, which notes the read",
}

// Limits of the guard below, by what it follows: calls to methods of *node by
// name, from the queries of Element. It does not see a function variable or a
// free function that takes the element, a method of *context that reads e.st, a
// method value (f := e.m), a note made on another receiver (e.parent.asksOutside()),
// a method of an embedded field, nor two types that share a method name (the
// set only widens). It skips the public methods that return *node (builders),
// 25 of them such as Root. An audit of ui/ found none of these forms that reads
// state today; a new one would have to be added to the guard.
func TestElementQueriesNoteTheRead(t *testing.T) {
	api, err := os.ReadFile("api_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	public := map[string]bool{}
	for _, m := range regexp.MustCompile(`func \(_handle Element\) (\w+)\(`).FindAllStringSubmatch(string(api), -1) {
		public[m[1]] = true
	}
	noted := map[string]bool{"asksOutside": true, "readsLayout": true, "readsHover": true, "readsDrag": true, "neverKept": true}
	// Fields of node that a widget or the engine sets from input as it builds,
	// and a query reads (the answer of the frame): reading one is reading state.
	stateFields := map[string]bool{"dismissed": true, "highlighted": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	// Every method of *node, by name, with its file.
	type method struct {
		file string
		fd   *ast.FuncDecl
	}
	methods := map[string][]method{}
	fset := token.NewFileSet()
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
			if !ok || fd.Recv == nil || fd.Body == nil || len(fd.Body.List) == 0 {
				continue
			}
			if st, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok && fmt.Sprint(st.X) == "node" {
				methods[fd.Name.Name] = append(methods[fd.Name.Name], method{f, fd})
			}
		}
	}
	builder := func(fd *ast.FuncDecl) bool {
		r := fd.Type.Results
		if r == nil || len(r.List) != 1 {
			return false
		}
		st, ok := r.List[0].Type.(*ast.StarExpr)
		return ok && fmt.Sprint(st.X) == "node"
	}
	// The queries of Element and every method of *node they call, to any depth
	// (by name: a name that another type shares only widens the set).
	reached := map[string]bool{}
	var visit func(name string)
	visit = func(name string) {
		if reached[name] {
			return
		}
		reached[name] = true
		for _, m := range methods[name] {
			ast.Inspect(m.fd.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					if se, ok := c.Fun.(*ast.SelectorExpr); ok && methods[se.Sel.Name] != nil {
						visit(se.Sel.Name)
					}
				}
				return true
			})
		}
	}
	for name := range public {
		for _, m := range methods[name] {
			if !builder(m.fd) {
				visit(name)
			}
		}
	}
	found := map[string]bool{}
	var missing []string
	for name := range reached {
		for _, m := range methods[name] {
			reads := false
			ast.Inspect(m.fd.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					reads = reads || n.Sel.Name == "st" || n.Sel.Name == "rt" || stateFields[n.Sel.Name]
				case *ast.CallExpr:
					if se, ok := n.Fun.(*ast.SelectorExpr); ok && se.Sel.Name == "hasState" {
						reads = true
					}
				}
				return true
			})
			if !reads {
				continue // it reads through the methods it calls, which are checked themselves
			}
			if es, ok := m.fd.Body.List[0].(*ast.ExprStmt); ok {
				if call, ok := es.X.(*ast.CallExpr); ok {
					if se, ok := call.Fun.(*ast.SelectorExpr); ok && noted[se.Sel.Name] {
						continue
					}
				}
			}
			key := m.file + " node." + name
			found[key] = true
			if _, ok := elementQueriesNotNoted[key]; !ok {
				missing = append(missing, key)
			}
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Errorf("methods that a query of Element reaches, that read state and do not begin by noting the read (asksOutside, readsLayout, neverKept...): %v", missing)
	}
	for key := range elementQueriesNotNoted {
		if !found[key] {
			t.Errorf("%q is listed in elementQueriesNotNoted but is not such a method (any more)", key)
		}
	}
}

// contextNotNoted: the methods of Context that do not begin by noting a read
// for the row being built (readsCtx, readsMods, neverKept), and why a row that
// is kept does not need it. A method of Context that is neither noted nor here
// fails TestContextMethodsNoteTheRead: decide which kind it is.
var contextNotNoted = map[string]string{
	"AddToast":          "an action, not a read: a row that raises a toast in every build is a bug the app sees at once",
	"After":             "asks for a build later: it bumps rt.bound, and a list does not keep such a row",
	"AnimationFrame":    "asks for frames: it bumps rt.bound, and a list does not keep such a row",
	"Announce":          "an action",
	"CloseToast":        "an action",
	"Direction":         "the direction of the parent, which a row is kept with (rec.dir)",
	"DisableAnimations": "an action",
	"Inspecting":        "the inspector open: no row is kept (listFrame.ok)",
	"MeasureText":       "a function of its arguments, the theme's fonts (themeGen) and the loaded ones",
	"OpenURL":           "an action",
	"OpenURLThen":       "an action",
	"Root":              "the window's root element, built before the rows",
	"RowSelected":       "the choice of the row, which a row is kept with (rec.chosen)",
	"SetTheme":          "an action, and the theme is compared when a row is kept (themeGen, rec.theme)",
	"SetZoom":           "an action",
	"Theme":             "compared when a row is kept (themeGen, rec.theme)",
	"Toast":             "an action",
	"ToastAction":       "an action",
	"WriteClipboard":    "an action",
}

func TestContextMethodsNoteTheRead(t *testing.T) {
	api, err := os.ReadFile("api_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	public := map[string]bool{}
	for _, m := range regexp.MustCompile(`func \(_handle \*Context\) (\w+)\(`).FindAllStringSubmatch(string(api), -1) {
		public[m[1]] = true
	}
	noted := map[string]bool{"readsCtx": true, "readsMods": true, "neverKept": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	found := map[string]bool{}
	var missing []string
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
			if !ok || fd.Recv == nil || fd.Body == nil || !public[fd.Name.Name] {
				continue
			}
			if st, ok := fd.Recv.List[0].Type.(*ast.StarExpr); !ok || fmt.Sprint(st.X) != "context" {
				continue
			}
			if r := fd.Type.Results; r != nil && len(r.List) == 1 {
				if st, ok := r.List[0].Type.(*ast.StarExpr); ok && fmt.Sprint(st.X) == "context" {
					continue // a builder
				}
			}
			found[fd.Name.Name] = true
			if len(fd.Body.List) > 0 {
				if es, ok := fd.Body.List[0].(*ast.ExprStmt); ok {
					if call, ok := es.X.(*ast.CallExpr); ok {
						if se, ok := call.Fun.(*ast.SelectorExpr); ok && noted[se.Sel.Name] {
							continue
						}
					}
				}
			}
			if _, ok := contextNotNoted[fd.Name.Name]; !ok {
				missing = append(missing, fd.Name.Name)
			}
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Errorf("methods of Context that do not begin by noting the read (readsCtx, readsMods, neverKept): %v", missing)
	}
	for name := range contextNotNoted {
		if !found[name] {
			t.Errorf("%q is listed in contextNotNoted but is not such a method (any more)", name)
		}
	}
}
