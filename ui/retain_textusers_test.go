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
