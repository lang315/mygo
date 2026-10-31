package ui

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// comboRows is a list of rows, a Combobox in every third and a Button built in
// every frame in the others (so that a row kept is one of a combobox), each twin
// with a choice of its own that the app changes without a version.
type comboRows struct {
	choice [2][]string
	twin   int
}

var comboOptions = []string{"apple", "apricot", "banana", "cherry", "date"}

func newComboRows(n int) *comboRows {
	r := &comboRows{}
	for k := range 2 {
		r.choice[k] = make([]string, n)
		for i := range n {
			r.choice[k][i] = comboOptions[i%len(comboOptions)]
		}
	}
	return r
}

func (r *comboRows) scene(n int) func() func(c *Context) {
	return func() func(c *Context) {
		me := r.twin
		r.twin++
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d", i)
						switch i % 3 {
						case 0:
							Combobox(c, &r.choice[me][i], comboOptions).Label(fmt.Sprint("Choice ", i)).Width(180)
						case 1:
							Autocomplete(c, &r.choice[me][i], comboOptions).Width(180)
						default:
							c.build().neverKept()
							Button(c, "press")
						}
					})
				}).Grow(1)
			})
		}
	}
}

// The rows of a closed combobox are kept, as those of a text input: they are
// built when it opens and when it closes, and the choice the app changes without
// a version shows at once.
func TestRetainComboboxRowsAreKept(t *testing.T) {
	const n = 30
	r := newComboRows(n)
	p := newPair(t, 500, 400, r.scene(n))
	p.idle(4)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	// The rows of buttons are built in every frame (comboRows); the others are the
	// rows of the comboboxes and the autocompletes: two thirds of the rows.
	rows := p.kept.rt.rowsBuilt - built + p.kept.rt.rowsKept - kept
	if got := p.kept.rt.rowsKept - kept; got*3 < rows*2-3 || got == 0 {
		t.Fatalf("idle: %d of %d rows kept (the rows of the closed comboboxes are not)", got, rows)
	}
	for k := range 2 {
		r.choice[k][3] = "cherry"
	}
	p.frame("the app chooses")
	p.frame("the app chooses +1")
	if !containsText(shownTexts(p.kept), "cherry") {
		t.Errorf("a kept row does not show the choice the app set: %q", shownTexts(p.kept))
	}
}

func containsText(texts []string, s string) bool {
	for _, t := range texts {
		if t == s {
			return true
		}
	}
	return false
}

// A random walk over rows of comboboxes: clicks that open and close them, typing
// that filters, Down, Up, Enter and Escape in the options, a click on an option,
// the focus leaving, the app choosing without a version, scrolling, the window's
// focus and the system's theme.
func TestRetainComboboxRowsRandomWalk(t *testing.T) {
	for seed := int64(1); seed <= raceFewer(walkSeeds(), 3); seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			const n = 30
			r := newComboRows(n)
			p := newPair(t, 500, 400, r.scene(n))
			p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
			p.idle(3)
			rng := rand.New(rand.NewSource(seed))
			sys := newSystemTheme(uint64(seed))
			words := []string{"a", "ap", "ban", "x", ""}
			keys := []Key{KeyDown, KeyDown, KeyUp, KeyEnter, KeyEscape, KeyTab}
			for step := range int(raceFewer(80, 50)) {
				x, y := float32(rng.Intn(520)-10), float32(rng.Intn(420)-10)
				var what string
				switch rng.Intn(13) {
				case 11:
					// The label of a combobox names its field and a click on it goes to
					// the input; a click on the chevron opens the options.
					i := rng.Intn(n/3) * 3
					if b, ok := p.kept.Find(fmt.Sprint("Choice ", i)); ok {
						cx, cy := b.X+b.W/2, b.Y+b.H/2
						what = fmt.Sprintf("click the label of %d", i)
						if rng.Intn(2) == 0 {
							cx, what = b.X+b.W-10, fmt.Sprintf("click the chevron of %d", i)
						}
						p.both(func(tt *Tester) { tt.ClickAt(cx, cy) })
					} else {
						what = "label not shown"
					}
				case 0, 1, 2:
					what = fmt.Sprintf("click %v,%v", x, y)
					p.both(func(tt *Tester) { tt.ClickAt(x, y) })
				case 3, 4:
					w := words[rng.Intn(len(words))]
					what = "type " + w
					p.both(func(tt *Tester) { tt.Type(w) })
				case 5, 6:
					k := keys[rng.Intn(len(keys))]
					what = fmt.Sprint("key ", k)
					p.both(func(tt *Tester) { tt.Key(0, k) })
				case 7:
					i := rng.Intn(n/3) * 3
					v := comboOptions[rng.Intn(len(comboOptions))]
					what = fmt.Sprintf("app chooses %d %s", i, v)
					for k := range 2 {
						r.choice[k][i] = v
					}
				case 8:
					what = fmt.Sprintf("move %v,%v", x, y)
					p.both(func(tt *Tester) { tt.Move(x, y) })
				case 9:
					dy := float32(rng.Intn(120) - 60)
					what = fmt.Sprintf("scroll %v", dy)
					p.both(func(tt *Tester) { tt.Scroll(x, y, 0, dy) })
				case 12:
					what = "tab"
					p.both(func(tt *Tester) { tt.Key(0, KeyTab) })
				default:
					what = "blur and focus"
					p.both(func(tt *Tester) { tt.SetFocused(false); tt.SetFocused(true) })
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

// A keepLinks element's links are made to the copies of the row, and one to an
// element outside the row makes the row built.
func TestRetainRowsLinkInsideAndOutside(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outside bool
	}{{"inside", false}, {"outside", true}} {
		t.Run(tc.name, func(t *testing.T) {
			var header *node
			p := newPair(t, 300, 300, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						header = Box(c).Size(10, 10).node()
						List(c, s, 8, func(i int) {
							Row(c).Padding(4).Children(func() {
								in := Box(c).Size(20, 20)
								f := Row(c).Size(40, 20)
								f.node().keepLinks, in.node().keepLinks = true, true
								f.node().focusTo, in.node().nameFrom = in.node(), f.node()
								if tc.outside {
									in.node().nameFrom = header
								}
							})
						}).Grow(1)
					})
				}
			})
			p.idle(4)
			kept := p.kept.rt.rowsKept
			p.frame("idle")
			got := p.kept.rt.rowsKept - kept
			if tc.outside && got != 0 {
				t.Errorf("%d rows kept, with a link to an element outside them", got)
			}
			if !tc.outside {
				if got == 0 {
					t.Fatalf("no row kept")
				}
				var check func(n *node, root *node)
				check = func(n, row *node) {
					if n.listRow {
						row = n
					}
					if n.focusTo != nil && !within(row, n.focusTo) {
						t.Errorf("a link of a kept row points outside its copy")
					}
					if n.nameFrom != nil && !within(row, n.nameFrom) {
						t.Errorf("a link of a kept row points outside its copy")
					}
					for c := n.first; c != nil; c = c.next {
						check(c, row)
					}
				}
				check(p.kept.rt.c.root, nil)
			}
		})
	}
}

// within reports whether o is e or inside it.
func within(e, o *node) bool {
	if e == nil {
		return false
	}
	if e == o {
		return true
	}
	for c := e.first; c != nil; c = c.next {
		if within(c, o) {
			return true
		}
	}
	return false
}

// The other users of field (SearchField, TokenField) and the number input link
// their elements too, and are not among those verified: their rows are built in
// every frame.
func TestRetainRowsOfOtherFieldsAreBuilt(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  func(c *Context, i int)
	}{
		{"SearchField", func(c *Context, i int) { SearchField(c, new(string)) }},
		{"TokenField", func(c *Context, i int) { TokenField(c, new([]string), comboOptions) }},
		{"NumberInput", func(c *Context, i int) { NumberInput(c, new(float64), 0, 10, 1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vals := make([]*string, 20)
			toks := make([][]string, 20)
			nums := make([]float64, 20)
			for i := range vals {
				vals[i] = new(string)
			}
			p := newPair(t, 400, 300, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
				return func(c *Context) {
					Column(c).Fill().Children(func() {
						List(c, s, 20, func(i int) {
							Row(c).Padding(4).Children(func() {
								switch tc.name {
								case "SearchField":
									SearchField(c, vals[i])
								case "TokenField":
									TokenField(c, &toks[i], comboOptions)
								default:
									NumberInput(c, &nums[i], 0, 10, 1)
								}
							})
						}).Grow(1)
					})
				}
			})
			p.idle(4)
			kept := p.kept.rt.rowsKept
			p.frame("idle")
			if got := p.kept.rt.rowsKept - kept; got != 0 {
				t.Errorf("%d rows of %s were kept", got, tc.name)
			}
		})
	}
}
