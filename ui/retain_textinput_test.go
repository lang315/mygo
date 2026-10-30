package ui

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// textRows: a list of rows, a TextInput in every third and a Button in the others,
// whose options (ReadOnly, MaxLength, Password) the app changes and versions, and
// whose text each twin keeps for itself.
type textRows struct {
	vals [2][]string
	opts [2]textOpts
	bump [2][]uint64
	twin int
}

type textOpts struct {
	readOnly, password bool
	maxLen             int
	gen                uint64
}

func newTextRows(n int) *textRows {
	r := &textRows{}
	for k := range 2 {
		r.vals[k], r.bump[k] = make([]string, n), make([]uint64, n)
	}
	return r
}

func (r *textRows) set(f func(o *textOpts)) {
	for k := range 2 {
		f(&r.opts[k])
		r.opts[k].gen++
	}
}

func (r *textRows) scene(n int) func() func(c *Context) {
	return func() func(c *Context) {
		me := r.twin
		r.twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(i int) uint64 {
			return r.opts[me].gen<<16 + r.bump[me][i]
		}}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d", i)
						if i%3 == 0 {
							o := r.opts[me]
							in := TextInput(c, &r.vals[me][i]).Width(140).Placeholder(fmt.Sprint("type ", i))
							if o.readOnly {
								in.ReadOnly(true)
							}
							if o.maxLen > 0 {
								in.MaxLength(o.maxLen)
							}
							if o.password {
								in.Password()
							}
						} else {
							Button(c, "press")
						}
					})
				}).Grow(1)
			})
		}
	}
}

// What the options of a text input were when its row was last built is what the
// user saw, and the input he makes (typing, a paste, an input method) applies by
// them: a row kept for many frames in between must not change that. The options
// change by a version, the row is kept for frames, and then the input arrives.
func TestRetainTextInputRowsApplyInputByWhatWasShown(t *testing.T) {
	const n = 30
	r := newTextRows(n)
	p := newPair(t, 500, 400, r.scene(n))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(4)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept: the test shows nothing")
	}
	// The rows of text inputs are kept too: nothing is built in an idle frame.
	built := p.kept.rt.rowsBuilt
	p.frame("idle")
	if b := p.kept.rt.rowsBuilt - built; b != 0 {
		t.Fatalf("idle: %d rows built (the rows of the text inputs are not kept)", b)
	}
	typeIn := func(what, text string) {
		p.both(func(tt *Tester) { tt.ClickAt(100, 14) }) // the first input (row 0)
		p.compareFocus(what + ": focus")
		p.both(func(tt *Tester) { tt.Type(text) })
		p.compareFocus(what + ": typed")
		p.both(func(tt *Tester) { tt.Command("selectAll") })
		p.both(func(tt *Tester) { tt.Type("") })
		p.both(func(tt *Tester) { tt.ClickAt(300, 100) }) // leave it
		p.compareFocus(what + ": left")
		p.idle(3)
	}
	// 1. MaxLength changes by a version; many frames; typing is cut at it.
	r.set(func(o *textOpts) { o.maxLen = 3 })
	p.idle(6)
	typeIn("max 3", "abcdef")
	// 2. A paste applies by it too.
	p.both(func(tt *Tester) { tt.SetClipboard("xyzxyz") })
	p.both(func(tt *Tester) { tt.ClickAt(100, 14) })
	p.both(func(tt *Tester) { tt.Command("selectAll") })
	p.both(func(tt *Tester) { tt.Command("paste") })
	p.compareFocus("paste at max 3")
	p.both(func(tt *Tester) { tt.ClickAt(300, 100) })
	p.idle(3)
	// 3. Password: the text shows as dots once the version says so.
	r.set(func(o *textOpts) { o.password = true })
	p.idle(6)
	typeIn("password", "hi")
	// 4. ReadOnly: nothing is typed or pasted, and a composition is not taken.
	r.set(func(o *textOpts) { o.readOnly = true; o.password = false; o.maxLen = 0 })
	p.idle(6)
	typeIn("read only", "no")
	p.both(func(tt *Tester) { tt.ClickAt(100, 14) })
	p.both(func(tt *Tester) { tt.Command("paste") })
	p.compareFocus("paste read only")
	p.both(func(tt *Tester) { tt.ClickAt(300, 100) })
	p.idle(3)
	// 5. An input method composes in an input that was kept, and commits.
	r.set(func(o *textOpts) { o.readOnly = false })
	p.idle(6)
	p.both(func(tt *Tester) { tt.ClickAt(100, 14) })
	p.both(func(tt *Tester) { tt.Compose("ni", 2) })
	p.compareFocus("composing")
	p.both(func(tt *Tester) { tt.Type("你") })
	p.compareFocus("committed")
}

// firstTextField returns the id of the first text field in the tree.
func firstTextField(t *testing.T, tt *Tester) uint64 {
	t.Helper()
	for _, nd := range tt.h.access.Nodes {
		if nd.Role == platform.RoleTextField {
			return nd.ID
		}
	}
	t.Fatal("no text field")
	return 0
}

// Assistive technology sets the value of a text input, or selects text in it,
// while the input is not focused and its row is kept: the value goes in (the focus
// goes to the input and its edits wait for the build: the state is pending) and
// the selection shows, as in the twin that builds every row.
func TestRetainTextInputRowsFollowAssistiveTechnology(t *testing.T) {
	const n = 30
	r := newTextRows(n)
	for k := range 2 {
		r.vals[k][0] = "Ada Lovelace"
	}
	p := newPair(t, 500, 400, r.scene(n))
	p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
	p.idle(5)
	if p.kept.rt.rowsKept == 0 {
		t.Fatal("no row kept")
	}
	built := p.kept.rt.rowsBuilt
	p.frame("idle")
	if p.kept.rt.rowsBuilt != built {
		t.Fatalf("idle: %d rows built", p.kept.rt.rowsBuilt-built)
	}
	p.both(func(tt *Tester) {
		tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: firstTextField(t, tt), Action: platform.AccessSelectText, From: 0, To: 3})
	})
	p.compareFocus("select text")
	p.idle(2)
	p.both(func(tt *Tester) {
		tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: firstTextField(t, tt), Action: platform.AccessSetValue, Text: "Grace"})
	})
	p.compareFocus("set value")
	p.frame("set value +1")
	p.idle(3)
	// Another input, three rows down, which was never focused.
	p.both(func(tt *Tester) {
		n := 0
		for _, nd := range tt.h.access.Nodes {
			if nd.Role == platform.RoleTextField {
				if n++; n == 2 {
					tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: nd.ID, Action: platform.AccessSetValue, Text: "Alan"})
					break
				}
			}
		}
	})
	p.compareFocus("set value of another")
	p.idle(3)
}

// A random walk over rows of text inputs and buttons: clicks into and out of the
// inputs, Tab, typing, editing keys, selection, the edit commands, undo and redo,
// an input method composing, committing and cancelling, the app changing a text
// (not versioned: the row records the *string) and the options (versioned),
// assistive technology, the window's focus, scrolling.
func TestRetainTextInputRowsRandomWalk(t *testing.T) {
	for seed := int64(1); seed <= walkSeeds(); seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			const n = 40
			r := newTextRows(n)
			p := newPair(t, 500, 400, r.scene(n))
			p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
			p.idle(3)
			rng := rand.New(rand.NewSource(seed))
			sys := newSystemTheme(uint64(seed))
			words := []string{"a", "bc", "Ada", "你好", "x y", ""}
			keys := []Key{KeyBackspace, KeyDelete, KeyLeft, KeyRight, KeyHome, KeyEnd, KeyEscape, KeyTab}
			cmds := []string{"selectAll", "copy", "cut", "paste", "undo", "redo"}
			for step := range 90 {
				x, y := float32(rng.Intn(520)-10), float32(rng.Intn(420)-10)
				var what string
				switch k := rng.Intn(18); k {
				case 0, 1, 2:
					what = fmt.Sprintf("click %v,%v", x, y)
					p.both(func(tt *Tester) { tt.ClickAt(x, y) })
				case 3, 4:
					w := words[rng.Intn(len(words))]
					what = "type " + w
					p.both(func(tt *Tester) { tt.Type(w) })
				case 5, 6:
					key := keys[rng.Intn(len(keys))]
					m := Modifiers(0)
					if rng.Intn(3) == 0 {
						m = Shift
					}
					what = fmt.Sprintf("key %v %v", m, key)
					p.both(func(tt *Tester) { tt.Key(m, key) })
				case 7:
					c := cmds[rng.Intn(len(cmds))]
					what = "command " + c
					p.both(func(tt *Tester) { tt.Command(c) })
				case 8:
					w := words[rng.Intn(len(words))]
					what = "compose " + w
					p.both(func(tt *Tester) { tt.Compose(w, len([]rune(w))) })
				case 9:
					what = fmt.Sprintf("move %v,%v", x, y)
					p.both(func(tt *Tester) { tt.Move(x, y) })
				case 10:
					dy := float32(rng.Intn(120) - 60)
					what = fmt.Sprintf("scroll %v", dy)
					p.both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
				case 11:
					what = "blur and focus"
					p.both(func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
				case 12:
					i := rng.Intn(n/3) * 3
					v := words[rng.Intn(len(words))]
					what = fmt.Sprintf("app sets %d to %q", i, v)
					for k := range 2 {
						r.vals[k][i] = v
					}
				case 13:
					what = "options"
					m := rng.Intn(4)
					r.set(func(o *textOpts) { o.maxLen = 0; o.password = m == 1; o.readOnly = m == 2; o.maxLen = m * 2 })
				case 14:
					i := rng.Intn(n)
					what = fmt.Sprintf("bump %d", i)
					for k := range 2 {
						r.bump[k][i]++
					}
				case 15:
					what = "a11y select text"
					p.both(func(tt *Tester) {
						if id := firstTextFieldOrZero(tt); id != 0 {
							tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: id, Action: platform.AccessSelectText, From: 0, To: 2})
						}
					})
				case 16:
					what = "a11y set value"
					p.both(func(tt *Tester) {
						if id := firstTextFieldOrZero(tt); id != 0 {
							tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: id, Action: platform.AccessSetValue, Text: "Zed"})
						}
					})
				default:
					what = "tab"
					p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
				}
				p.both(func(tt *Tester) { tt.Frame() })
				what += sys.step(p.both)
				p.compareFocus(fmt.Sprintf("%d %s", step, what))
				p.frame(fmt.Sprintf("%d %s +1", step, what))
			}
			if p.kept.rt.rowsKept == 0 {
				t.Fatalf("no row kept: the walk tests nothing")
			}
		})
	}
}

func firstTextFieldOrZero(tt *Tester) uint64 {
	for _, nd := range tt.h.access.Nodes {
		if nd.Role == platform.RoleTextField {
			return nd.ID
		}
	}
	return 0
}

// A text area scrolls, lays out and reveals its caret, and a document editor and an
// app's own text-input client carry state the row does not hold: their rows are
// built in every frame, as before.
func TestRetainTextAreaAndClientRowsAreBuilt(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  func(c *Context, v *string)
	}{
		{"TextArea", func(c *Context, v *string) { TextArea(c, v).Width(140) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vals := make([]string, 30)
			p := newPair(t, 400, 300, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						List(c, s, 30, func(i int) {
							Row(c).Padding(4).Children(func() { tc.row(c, &vals[i]) })
						}).Grow(1)
					})
				}
			})
			p.idle(4)
			built := p.kept.rt.rowsBuilt
			p.frame("idle")
			if p.kept.rt.rowsBuilt == built {
				t.Errorf("rows of a %s were kept", tc.name)
			}
		})
	}
}
