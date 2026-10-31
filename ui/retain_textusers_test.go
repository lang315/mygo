package ui

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// The other users of the single-line text input (a combobox, an autocomplete,
// a token field, a number input, an editable label, a color picker's hex field)
// in rows of a List with RowVersion, kept twin against built twin: their popups,
// edit modes, drafts and values the app changes must not leave a kept row stale.
func TestRetainRowsOfTextInputUsersRandomWalk(t *testing.T) {
	// "number" and "color" (NumberInput, ColorPicker) differ between the twins
	// on main as well, without any text input kept: a separate bug, not walked here.
	kinds := []string{"combobox", "autocomplete", "tokens", "number", "editable", "color"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			for seed := int64(1); seed <= walkSeeds(); seed++ {
				t.Run(fmt.Sprint(seed), func(t *testing.T) { textUsersWalk(t, kind, seed) })
			}
		})
	}
}

func textUsersWalk(t *testing.T, kind string, seed int64) {
	const n = 40
	opts := []string{"apple", "apricot", "banana", "cherry", "date"}
	var text [2][]string
	var num [2][]float64
	var toks [2][][]string
	var col [2][]Color
	for k := range 2 {
		text[k], num[k], toks[k], col[k] = make([]string, n), make([]float64, n), make([][]string, n), make([]Color, n)
		for i := range n {
			col[k][i] = RGB(uint8(i*5), 100, 200)
		}
	}
	twin := 0
	var sel [2]int
	p := newPair(t, 500, 400, func() func(c *Context) {
		me := twin
		twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }, Selected: &sel[me]}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d", i)
						if i%3 != 0 {
							Button(c, "press")
							return
						}
						switch kind {
						case "combobox":
							Combobox(c, &text[me][i], opts)
						case "autocomplete":
							Autocomplete(c, &text[me][i], opts)
						case "tokens":
							TokenField(c, &toks[me][i], opts)
						case "number":
							NumberInput(c, &num[me][i], 0, 100, 1)
						case "editable":
							EditableText(c, &text[me][i])
						default:
							ColorPicker(c, &col[me][i])
						}
					})
				}).Grow(1)
			})
		}
	})
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(3)
	rng := rand.New(rand.NewSource(seed))
	sys := newSystemTheme(uint64(seed))
	keys := []Key{KeyDown, KeyUp, KeyEnter, KeyEscape, KeyTab, KeyF2}
	for step := range 80 {
		x, y := float32(rng.Intn(520)-10), float32(rng.Intn(420)-10)
		var what string
		switch k := rng.Intn(13); k {
		case 0, 1, 2:
			what = fmt.Sprintf("click %v,%v", x, y)
			p.both(func(tt *Tester) { tt.ClickAt(x, y) })
		case 3, 4:
			s := []string{"a", "ap", "5", "x", "12"}[rng.Intn(5)]
			what = "type " + s
			p.both(func(tt *Tester) { tt.Type(s) })
		case 5, 6:
			key := keys[rng.Intn(len(keys))]
			what = fmt.Sprintf("key %v", key)
			p.both(func(tt *Tester) { tt.Key(0, key) })
		case 7:
			what = "double click"
			p.both(func(tt *Tester) { tt.ClickAt(x, y); tt.ClickAt(x, y) })
		case 8:
			what = fmt.Sprintf("move %v,%v", x, y)
			p.both(func(tt *Tester) { tt.Move(x, y) })
		case 9:
			dy := float32(rng.Intn(120) - 60)
			what = fmt.Sprintf("scroll %v", dy)
			p.both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
		case 10:
			what = "blur and focus"
			p.both(func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
		case 11:
			i := rng.Intn(n/3) * 3
			v := opts[rng.Intn(len(opts))]
			what = fmt.Sprintf("app sets %d to %s", i, v)
			nv, cv := float64(rng.Intn(100)), RGB(uint8(rng.Intn(256)), 10, 20) // drawn once: both twins get the same
			for k := range 2 {
				text[k][i], num[k][i] = v, nv
				toks[k][i] = []string{v}
				col[k][i] = cv
			}
		default:
			what = "idle"
		}
		what += sys.step(p.both)
		p.compareFocus(fmt.Sprintf("%d %s", step, what))
		p.compareFocus(fmt.Sprintf("%d %s +1", step, what))
	}
	t.Logf("%s seed %d: rows kept %d", kind, seed, p.kept.rt.rowsKept)
}

// shownTexts returns the text of every single-line input in the last frame.
func shownTexts(tt *Tester) (out []string) {
	var walk func(n *node)
	walk = func(n *node) {
		if n.st != nil && n.st.editor != nil && n.kind == 3 {
			out = append(out, n.st.editor.buf.string())
		}
		for c := n.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(tt.rt.c.root)
	return out
}

// App code changes the value a control is bound to without a version bump,
// as it may when the control's own state is what the row watches: the kept row
// must show what a built one shows. Each user of the single-line input gets its
// value to the row by its own path; this fails if that path is not noted.
func TestRetainRowsOfTextInputUsersFollowTheApp(t *testing.T) {
	for _, kind := range []string{"textinput", "textinputbase", "autocomplete", "comboboxbase", "combobox", "tokens", "number", "editable", "color"} {
		t.Run(kind, func(t *testing.T) {
			const n = 12
			var text [2][]string
			var num [2][]float64
			var toks [2][][]string
			var col [2][]Color
			var sel [2]int
			for k := range 2 {
				text[k], num[k], toks[k], col[k] = make([]string, n), make([]float64, n), make([][]string, n), make([]Color, n)
			}
			opts := []string{"apple", "apricot", "banana"}
			twin := 0
			p := newPair(t, 500, 400, func() func(c *Context) {
				me := twin
				twin++
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }, Selected: &sel[me]}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						List(c, s, n, func(i int) {
							Row(c).Padding(4).Children(func() {
								switch kind {
								case "textinput":
									TextInput(c, &text[me][i]).Width(140)
								case "textinputbase":
									TextInputBase(c, &text[me][i]).Width(140)
								case "comboboxbase":
									ComboboxBase(c, &text[me][i]).Input.Width(140)
								case "combobox":
									Combobox(c, &text[me][i], opts)
								case "autocomplete":
									Autocomplete(c, &text[me][i], opts)
								case "tokens":
									TokenField(c, &toks[me][i], opts)
								case "number":
									NumberInput(c, &num[me][i], 0, 100, 1)
								case "editable":
									EditableText(c, &text[me][i])
								default:
									ColorPicker(c, &col[me][i])
								}
							})
						}).Grow(1)
					})
				}
			})
			p.idle(4)
			kept := p.kept.rt.rowsKept
			for k := range 2 {
				text[k][2], num[k][2], toks[k][2], col[k][2] = "apricot", 42, []string{"banana"}, RGB(200, 30, 90)
			}
			for range 3 {
				p.frame("app changed row 2")
			}
			var rows func(n *node, f func(r *node))
			rows = func(n *node, f func(r *node)) {
				if n.listRow {
					f(n)
					return
				}
				for c := n.first; c != nil; c = c.next {
					rows(c, f)
				}
			}
			why := ""
			rows(p.kept.rt.c.root, func(r *node) {
				if w := unretainable(r); w != "" {
					why = w
				}
			})
			// The path that keeps the row from going stale. The combobox, the token
			// field and the number input bind a state of their own, which a kept row
			// would not see follow the app: they are never kept (retainable: focusTo,
			// with the field, popover and others behind it for the first two). The
			// color picker is not either (a paint function of its own, and behind it a
			// motion that is not animateRest, state.pending), the editable text uses a
			// Handle, and the TextInput, TextInputBase, ComboboxBase and Autocomplete are
			// kept and watch the app's string itself (readsString).
			wantWhy := map[string]string{"combobox": "focusTo", "tokens": "focusTo", "number": "focusTo", "color": "paintFn"}
			if want, never := wantWhy[kind]; never {
				if why != want || p.kept.rt.rowsKept != 0 {
					t.Fatalf("rows kept %d, held back by %q: want none, held back by %q", p.kept.rt.rowsKept, why, want)
				}
			} else if kind == "editable" {
				if p.kept.rt.rowsKept != 0 {
					t.Fatalf("%d rows kept: an editable text uses a Handle, which builds its row", p.kept.rt.rowsKept)
				}
			} else {
				if p.kept.rt.rowsKept <= kept {
					t.Fatalf("rows kept %d -> %d: the rows of a %s are kept", kept, p.kept.rt.rowsKept, kind)
				}
				if got := shownTexts(p.kept); got[2] != "apricot" {
					t.Fatalf("a kept row shows %q after the app set apricot", got[2])
				}
			}
		})
	}
}

// unretainable names the first thing that keeps e, or an element under it, from
// being kept (retainable), or "" if none does.
func unretainable(e *node) string {
	for _, c := range []struct {
		name string
		is   bool
	}{
		{"popover", e.popover != nil}, {"placeOn", e.placeOn != nil}, {"nameFrom", e.nameFrom != nil},
		{"focusTo", e.focusTo != nil}, {"activeDescendant", e.activeDescendant != nil}, {"field", e.field != nil},
		{"form", e.form != nil}, {"overflow", e.overflow != nil}, {"inputFn", e.inputFn != nil},
		{"paintAfterFn", e.paintAfterFn != nil}, {"paintFn", e.paintFn != nil && !e.paintPure}, {"tip", e.tip},
		{"effect", e.effect != nil}, {"hidden", e.hidden}, {"copyData", e.copyData != nil},
	} {
		if c.is {
			return c.name
		}
	}
	for c := e.first; c != nil; c = c.next {
		if w := unretainable(c); w != "" {
			return w
		}
	}
	return ""
}
