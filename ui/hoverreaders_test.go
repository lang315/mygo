package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
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
var hoverReaders = map[string]string{
	// The ways to ask in a build: they note it.
	"input.go node.Hovered":         "notes the read (readsHover)",
	"input.go node.PointerPosition": "notes the read (readsHover)",
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
	"dragdrop.go node.Dragging":      "the source, which is acting",
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
	"retain.go listFrame.capture":     "the rule itself (the modifiers a row was built with)",
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
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, "input.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []struct{ name, notes string }{
		{"Hovered", "readsHover"}, {"PointerPosition", "readsHover"}, {"Modifiers", "readsMods"},
	} {
		noted := false
		for _, d := range af.Decls {
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
