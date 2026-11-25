package widget

import (
	"fmt"
	"image"
	"image/color"
	"maps"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// tipKit is the ribbon of the cases in a form with no MenuBar, access keys
// on (macMenus false), 800 wide.
func tipKit(t *testing.T) (*ribbonKit, *ui.Tester) {
	t.Helper()
	withMac(t, false)
	k := newRibbonKit(t)
	return k, k.tester(800)
}

// typeTip presses k alone and lets the frames it asks for run.
func typeTip(tt *ui.Tester, k ui.Key) {
	tt.Key(0, k)
	tt.Frame()
}

// tipsRead returns the key tips assistive technology reads: the name of each
// element with a key shortcut, and the shortcut.
func tipsRead(tt *ui.Tester) map[string]string {
	m := map[string]string{}
	for _, a := range tt.AccessAll() {
		if a.KeyShortcuts != "" {
			m[a.Name] = a.KeyShortcuts
		}
	}
	return m
}

// tipBadges returns the boxes of the badges in img: the parts filled with
// fill at least minH high and half as wide, of which half or more is fill,
// that enclose something else, their letter (a glyph or a line of the text's
// color is thin, or open, as a tab title's "H" in a heavy font; a badge is a
// box with its letter in it).
func tipBadges(img *image.RGBA, fill ui.Color, minH int) []ui.Rect {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	part := make([]int32, w*h) // 1 + the part's index, 0 if not fill
	var out []ui.Rect
	var stack []int
	parts := int32(0)
	for start := range part {
		if part[start] != 0 || !near(img.RGBAAt(start%w, start/w), fill) {
			continue
		}
		parts++
		x0, y0, x1, y1, n := w, h, -1, -1, 0
		part[start] = parts
		stack = append(stack[:0], start)
		for len(stack) > 0 {
			i := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := i%w, i/w
			x0, y0, x1, y1, n = min(x0, x), min(y0, y), max(x1, x), max(y1, y), n+1
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					continue
				}
				j := ny*w + nx
				if part[j] == 0 && near(img.RGBAAt(nx, ny), fill) {
					part[j] = parts
					stack = append(stack, j)
				}
			}
		}
		bw, bh := x1-x0+1, y1-y0+1
		r := image.Rect(x0, y0, x1+1, y1+1)
		if bh >= minH && 2*bw >= minH && 2*n >= bw*bh && encloses(part, w, r, parts) {
			out = append(out, ui.Rect{X: float32(x0), Y: float32(y0), W: float32(bw), H: float32(bh)})
		}
	}
	return out
}

// encloses reports whether part p, whose box is r, closes around a pixel
// not its own: one that r's edges do not reach without crossing p.
func encloses(part []int32, w int, r image.Rectangle, p int32) bool {
	out := make([]bool, r.Dx()*r.Dy()) // reached from r's edges
	at := func(x, y int) int { return (y-r.Min.Y)*r.Dx() + x - r.Min.X }
	var stack []image.Point
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			edge := x == r.Min.X || y == r.Min.Y || x == r.Max.X-1 || y == r.Max.Y-1
			if edge && part[y*w+x] != p && !out[at(x, y)] {
				out[at(x, y)] = true
				stack = append(stack, image.Pt(x, y))
			}
		}
	}
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range []image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := q.Add(d)
			if n.In(r) && part[n.Y*w+n.X] != p && !out[at(n.X, n.Y)] {
				out[at(n.X, n.Y)] = true
				stack = append(stack, n)
			}
		}
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if part[y*w+x] != p && !out[at(x, y)] {
				return true
			}
		}
	}
	return false
}

// badgeMinH is the least height of a badge at text scale 1: the letter's
// line, 0.75 rem, and more.
var badgeMinH = int(ui.LightTheme().Rem(0.75) * 1.3)

// wantTips checks that the tips read are want, by name, and that each has a
// badge of the light theme's Text centred on its element's bottom edge,
// within 1 DIP, and no other badge shows.
func wantTips(t *testing.T, tt *ui.Tester, want map[string]string) {
	t.Helper()
	wantTipsIn(t, tt, want, ui.LightTheme().Text, badgeMinH)
}

func wantTipsIn(t *testing.T, tt *ui.Tester, want map[string]string, fill ui.Color, minH int) []ui.Rect {
	t.Helper()
	if got := tipsRead(tt); !maps.Equal(got, want) {
		t.Fatalf("tips %v, want %v", got, want)
	}
	badges := tipBadges(tt.Image(), fill, minH)
	if len(badges) != len(want) {
		t.Fatalf("%d badges %v, want %d", len(badges), badges, len(want))
	}
	for _, a := range tt.AccessAll() {
		if a.KeyShortcuts == "" {
			continue
		}
		cx, cy := a.Bounds.X+a.Bounds.W/2, a.Bounds.Y+a.Bounds.H
		found := false
		for _, b := range badges {
			if near1(b.X+b.W/2, cx) && near1(b.Y+b.H/2, cy) {
				found = true
			}
		}
		if !found {
			t.Fatalf("no badge centred on %q's bottom edge (%v, %v): badges %v", a.Name, cx, cy, badges)
		}
	}
	return badges
}

var homeTips = map[string]string{"Paste": "P", "Cut": "C", "Copy": "O", "Painter": "A", "Bold": "B", "Italic": "I", "Underline": "U", "Size": "S"}
var tabTips = map[string]string{"Home": "H", "Insert": "I"}

// nothingRan checks that no handler of the kit ran.
func nothingRan(t *testing.T, k *ribbonKit) {
	t.Helper()
	Update(func() {}) // the handlers posted so far have run
	if n, m, c := len(k.pasted), len(k.bolded), len(k.cutRan); n != 0 || m != 0 || c != 0 {
		t.Fatalf("Paste ran %d times, Bold's OnChange %d, Cut %d: want 0", n, m, c)
	}
}

// 1. F10 shows "H" and "I" over the tabs, centred on their bottom edges;
// no command has one.
func TestRibbonTipsF10ShowsTabTips(t *testing.T) {
	_, tt := tipKit(t)
	wantTips(t, tt, map[string]string{})
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, tabTips)
	home, insert := nodeOf(t, tt, ui.RoleTab, "Home").Bounds, nodeOf(t, tt, ui.RoleTab, "Insert").Bounds
	if home.X >= insert.X {
		t.Fatalf("Home at %v is not before Insert at %v", home, insert)
	}
}

// 2. Alt pressed and let go alone (KeyMenuBar) does as F10.
func TestRibbonTipsAltAloneShowsTabTips(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyMenuBar)
	wantTips(t, tt, tabTips)
}

// 3. On tabs, H shows Home's commands' tips, and no tab's.
func TestRibbonTipsTabLetterShowsCommandTips(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	wantTips(t, tt, homeTips)
}

// 4. On commands, P runs Paste's OnClick once and ends the tips.
func TestRibbonTipsCommandLetterClicksAndEnds(t *testing.T) {
	k, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	typeTip(tt, ui.KeyP)
	received(t, k.pasted)
	nothingRan(t, k)
	wantTips(t, tt, map[string]string{})
}

// 5. On commands, B toggles Bold and runs its OnChange once.
func TestRibbonTipsToggleLetterTogglesOnce(t *testing.T) {
	k, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	typeTip(tt, ui.KeyB)
	received(t, k.bolded)
	nothingRan(t, k)
	if !k.bold.Checked() {
		t.Fatal("B did not check Bold")
	}
	wantTips(t, tt, map[string]string{})
}

// 6. On commands, S gives the Size combo the focus.
func TestRibbonTipsComboLetterFocuses(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	holdEvents(t)
	typeTip(tt, ui.KeyS)
	settleTips(t, tt) // the focus comes through the event goroutine
	if !tt.Focused("Size") {
		t.Fatal("S did not focus the Size combo")
	}
	wantTips(t, tt, map[string]string{})
}

// 7. On tabs, I shows Insert, Table with "T".
func TestRibbonTipsOtherTab(t *testing.T) {
	k, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyI)
	if k.r.Tab() != 1 || !tt.HasText("Tables") {
		t.Fatalf("I: Tab %d, texts %q", k.r.Tab(), tt.Texts())
	}
	wantTips(t, tt, map[string]string{"Table": "T"})
}

// 8. Escape on commands goes back to the tabs' tips; on tabs it ends them.
func TestRibbonTipsEscapeGoesBackALevel(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	typeTip(tt, ui.KeyEscape)
	wantTips(t, tt, tabTips)
	typeTip(tt, ui.KeyEscape)
	wantTips(t, tt, map[string]string{})
}

// 9. F10 while on ends the tips, and so does a key that is no tip, which
// runs nothing.
func TestRibbonTipsF10AgainAndOtherKeyEnd(t *testing.T) {
	k, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, map[string]string{})
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyZ)
	wantTips(t, tt, map[string]string{})
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	typeTip(tt, ui.KeyZ)
	wantTips(t, tt, map[string]string{})
	nothingRan(t, k)
}

// 10. A press in the window ends the tips.
func TestRibbonTipsPressEnds(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	tt.Press(400, 350)
	tt.Release(400, 350)
	tt.Frame()
	wantTips(t, tt, map[string]string{})
}

// 11. Collapsed: F10, then H, expands the ribbon with Home's tips.
func TestRibbonTipsCollapsedExpands(t *testing.T) {
	k, tt := tipKit(t)
	k.r.SetCollapsed(true)
	tt.Frame()
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, tabTips)
	typeTip(tt, ui.KeyH)
	if k.r.Collapsed() {
		t.Fatal("H left the ribbon collapsed")
	}
	wantTips(t, tt, homeTips)
}

// 12. Alt+H with no tips shows Home with its commands' tips.
func TestRibbonTipsAltLetterGoesToCommands(t *testing.T) {
	k, tt := tipKit(t)
	k.r.SetTab(1)
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyH)
	tt.Frame()
	if k.r.Tab() != 0 {
		t.Fatalf("Alt+H: Tab %d, want 0", k.r.Tab())
	}
	wantTips(t, tt, homeTips)
}

// layoutOf returns every node's role, name and bounds.
func layoutOf(tt *ui.Tester) []string {
	var s []string
	for _, n := range tt.Nodes() {
		s = append(s, fmt.Sprintf("%v %q %v", n.Role, n.Label, n.Bounds))
	}
	return s
}

// 13. The tips change no element's bounds.
func TestRibbonTipsKeepTheLayout(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyEscape)
	off := layoutOf(tt) // the focus where the tips leave it
	typeTip(tt, ui.KeyF10)
	tabs := layoutOf(tt)
	typeTip(tt, ui.KeyH)
	cmds := layoutOf(tt)
	for name, on := range map[string][]string{"tabs": tabs, "commands": cmds} {
		if len(on) != len(off) {
			t.Fatalf("%d nodes with the %s' tips, %d without", len(on), name, len(off))
		}
		for i := range on {
			if on[i] != off[i] {
				t.Fatalf("with the %s' tips %s, without %s", name, on[i], off[i])
			}
		}
	}
}

// 14. With macOS's menus, F10 and Alt alone show no tip, nor does Alt+H.
func TestRibbonTipsOffOnMac(t *testing.T) {
	k := newRibbonKit(t)
	withMac(t, true)
	tt := k.tester(800)
	for _, x := range []ui.Key{ui.KeyF10, ui.KeyMenuBar} {
		typeTip(tt, x)
		wantTips(t, tt, map[string]string{})
	}
	tt.Key(ui.Alt, ui.KeyH)
	tt.Frame()
	wantTips(t, tt, map[string]string{})
}

// 15. With a MenuBar in the form, F10 goes to the bar, and the ribbon shows
// no tip.
func TestRibbonTipsLeaveF10ToAMenuBar(t *testing.T) {
	withMac(t, false)
	k := newRibbonKit(t)
	bar := MenuBar(MenuItem{Text: "&File", Children: []MenuItem{{Text: "Open"}}})
	k.form.SetContent(VBox(0).Add(bar, k.r, k.body))
	tt := k.tester(800)
	typeTip(tt, ui.KeyF10)
	mu.Lock()
	inMode := k.form.modeBar == bar
	mu.Unlock()
	if !inMode {
		t.Fatal("F10 did not take the keyboard to the menu bar")
	}
	wantTips(t, tt, map[string]string{})
}

// Own case: with the system's menu bar (SetMenuBar), F10 is the system's.
func TestRibbonTipsLeaveF10ToTheSystemMenu(t *testing.T) {
	withMac(t, false)
	k := newRibbonKit(t)
	k.form.SetMenuBar(MenuItem{Text: "&File", Children: []MenuItem{{Text: "Open"}}})
	tt := k.tester(800)
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, map[string]string{})
}

// ribbonBoxesForm is the kit's controls in boxes under a ui.TabsKeyed strip
// (ribbonRef), in a form as the kit's.
func ribbonBoxesForm(t *testing.T) *Form {
	q := newRibbonKit(t)
	ref := &ribbonRef{keys: []any{1, 2}, labels: []string{"Home", "Insert"},
		body: HBox(0).Add(HBox(0).Add(q.paste, VBox(0).Add(q.cut, q.copy, q.painter)),
			HBox(0).Add(VBox(0).Add(q.bold, q.italic, q.underline), VBox(0).Add(q.size)))}
	ref.init(ref)
	return NewForm("Ribbon", 800, 400).SetContent(VBox(0).Add(ref, q.body))
}

// restAllocs is what a frame of tt allocates at rest.
func restAllocs(tt *ui.Tester) float64 {
	for range 10 {
		tt.Frame()
	}
	return allocrun.Least(30, tt.Frame)
}

// 16. The kit's form allocates in a frame, with no tips, 3 fewer than the
// same controls in boxes, as on main (14 and 17 on darwin at c22a9fc4a);
// with the tabs' or the commands' tips at rest, 2 more at most, plus what
// ui's Element.KeyShortcuts allocates for each badge, formatting its key.
// Measured on darwin: 14 with no tips, 24 with the tabs' (2 badges), 54 with
// Home's commands' (8 badges).
func TestRibbonTipsAllocations(t *testing.T) {
	_, tt := tipKit(t)
	boxes := ui.NewTester(ribbonBoxesForm(t).view, 800, 400)
	off, ref := restAllocs(tt), restAllocs(boxes)
	typeTip(tt, ui.KeyF10)
	tabs := restAllocs(tt)
	typeTip(tt, ui.KeyH)
	cmds := restAllocs(tt)
	t.Logf("allocations in a frame: no tips %v, tabs' %v, commands' %v (darwin: 14, 24, 54); the boxes %v", off, tabs, cmds, ref)
	if off > ref-3 {
		t.Fatalf("with no tips the form allocates %v, the boxes %v: want 3 fewer, as on main", off, ref)
	}
	for _, c := range []struct {
		name   string
		got    float64
		badges int
	}{{"tabs'", tabs, len(tabTips)}, {"commands'", cmds, len(homeTips)}} {
		if want := off + 2 + keyShortcutsAllocs*float64(c.badges); c.got > want {
			t.Fatalf("the %s tips allocate %v, none %v: want %v at most (2, and %d for each of %d badges)", c.name, c.got, off, want, keyShortcutsAllocs, c.badges)
		}
	}
}

// keyShortcutsAllocs is what ui's Element.KeyShortcuts allocates in a call,
// formatting its key (ui/access.go, accelerator, shortcutText).
const keyShortcutsAllocs = 5

// 17. AddTab and AddSmall from another goroutine while the tips show: the
// letters follow on a following frame (run it under -race).
func TestRibbonTipsChangesFromAnotherGoroutine(t *testing.T) {
	k, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		k.r.AddTab("Late")
		k.r.AddSmall(ToolButton("Button"))
	}()
	for range 20 {
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	wantTips(t, tt, map[string]string{"Home": "H", "Insert": "I", "Late": "L"})
	typeTip(tt, ui.KeyL)
	wantTips(t, tt, map[string]string{"Button": "B"})
}

// Access: Paste has its tip as a key shortcut while its badge shows, and
// none with the tips off.
func TestRibbonTipsAccess(t *testing.T) {
	_, tt := tipKit(t)
	if a, _ := tt.Access("Paste"); a.KeyShortcuts != "" {
		t.Fatalf("with no tips Paste has the key shortcut %q", a.KeyShortcuts)
	}
	typeTip(tt, ui.KeyF10)
	if a, _ := tt.Access("Paste"); a.KeyShortcuts != "" {
		t.Fatalf("with the tabs' tips Paste has the key shortcut %q", a.KeyShortcuts)
	}
	typeTip(tt, ui.KeyH)
	if a, _ := tt.Access("Paste"); a.KeyShortcuts != "P" {
		t.Fatalf("with its badge Paste has the key shortcut %q, want P", a.KeyShortcuts)
	}
	typeTip(tt, ui.KeyEscape)
	typeTip(tt, ui.KeyEscape)
	if a, _ := tt.Access("Paste"); a.KeyShortcuts != "" {
		t.Fatalf("after the tips Paste has the key shortcut %q", a.KeyShortcuts)
	}
}

// tipsForm shows r over a text box in a form 800 × 400, access keys on.
func tipsForm(t *testing.T, r *RibbonWidget) *ui.Tester {
	t.Helper()
	withMac(t, false)
	tt := ui.NewTester(NewForm("Ribbon", 800, 400).SetContent(VBox(0).Add(r, TextBox().SetAccessibleName("Body"))).view, 800, 400)
	tt.Frame()
	return tt
}

// Review Focus 1: two tabs of one first letter: H, then E.
func TestRibbonTipsSameFirstLetter(t *testing.T) {
	tt := tipsForm(t, Ribbon().AddTab("Home").AddGroup("G").AddSmall(ToolButton("A")).AddTab("Help").AddGroup("G").AddSmall(ToolButton("B")))
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, map[string]string{"Home": "H", "Help": "E"})
}

// Review Focus 2: a Vietnamese title's tip is its letter without its accent.
func TestRibbonTipsFoldAccents(t *testing.T) {
	tt := tipsForm(t, Ribbon().AddTab("Định dạng").AddGroup("G").AddSmall(ToolButton("Ẩn")))
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, map[string]string{"Định dạng": "D"})
	typeTip(tt, ui.KeyD)
	wantTips(t, tt, map[string]string{"Ẩn": "A"})
}

// Review Focus 3: in dark mode under each built-in skin the badge is the
// skin's Text, and its letter, the skin's Background, reaches 4.5:1 on it.
func TestRibbonTipsContrast(t *testing.T) {
	for _, name := range []string{"Calm", "Classic", "HighContrast"} {
		t.Run(name, func(t *testing.T) {
			useSkinNamed(t, name, false)
			tk := skinhook.Get(true)
			withMac(t, false)
			k := newRibbonKit(t)
			tt := ui.NewTester(k.form.view, 800, 400)
			tt.SetDark(true)
			tt.Frame()
			typeTip(tt, ui.KeyF10)
			wantTipsIn(t, tt, tabTips, tk.Text, badgeMinH)
			if r := ratio(tk.Background, tk.Text); r < 4.5 {
				t.Fatalf("the letter's Background on the badge's Text is %.2f:1, want 4.5:1", r)
			}
		})
	}
}

// Review Focus 4: at text scale 1.5 the badges grow with the text and stay
// centred.
func TestRibbonTipsGrowWithTheTextScale(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	small := tipBadges(tt.Image(), ui.LightTheme().Text, badgeMinH)
	withMac(t, false)
	k := newRibbonKit(t)
	big := ui.NewTester(k.form.view, 1000, 500)
	big.SetPreferences(ui.Preferences{TextScale: 1.5})
	big.Frame()
	typeTip(big, ui.KeyF10)
	large := wantTipsIn(t, big, tabTips, ui.LightTheme().Text, badgeMinH)
	if len(small) == 0 || large[0].H < 1.4*small[0].H {
		t.Fatalf("badges %v at text scale 1.5, %v at 1: they did not grow", large, small)
	}
}

// tipBadges takes a box around its letter and not a glyph of the same color
// as heavy as Linux's font draws a tab title's "H" at text scale 1.5, which
// fills more than half its box but encloses nothing.
func TestRibbonTipBadgesSkipGlyphs(t *testing.T) {
	fill := ui.LightTheme().Text
	img := image.NewRGBA(image.Rect(0, 0, 60, 30))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	paint := func(x0, y0, x1, y1 int, c color.RGBA) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	ink := color.RGBA{fill.R, fill.G, fill.B, 255}
	paint(2, 2, 6, 17, ink) // "H", 13 × 15: its stems and bar
	paint(11, 2, 15, 17, ink)
	paint(6, 8, 11, 11, ink)
	paint(30, 2, 50, 22, ink) // a badge, 20 × 20, and its letter "I"
	paint(39, 7, 41, 17, color.RGBA{255, 255, 255, 255})
	got := tipBadges(img, fill, badgeMinH)
	if want := (ui.Rect{X: 30, Y: 2, W: 20, H: 20}); len(got) != 1 || got[0] != want {
		t.Fatalf("badges %v, want [%v]", got, want)
	}
}

// Review Focus 5: a disabled command has a badge; its letter runs nothing
// and ends the tips.
func TestRibbonTipsDisabledCommandEnds(t *testing.T) {
	k, tt := tipKit(t)
	k.cut.SetDisabled(true)
	tt.Frame()
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	wantTips(t, tt, homeTips)
	typeTip(tt, ui.KeyC)
	nothingRan(t, k)
	wantTips(t, tt, map[string]string{})
}

// Own case (orchestrator): a drop-down button's tip gives it the focus and
// ends the tips; Down then opens its menu.
func TestRibbonTipsDropDownTakesTheFocus(t *testing.T) {
	tt := tipsForm(t, Ribbon().AddTab("Home").AddGroup("G").AddSmall(DropDownButton("Fonts", MenuItem{Text: "Arial"}, MenuItem{Text: "Serif"})))
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	wantTips(t, tt, map[string]string{"Fonts": "F"})
	holdEvents(t)
	typeTip(tt, ui.KeyF)
	settleTips(t, tt)
	if !tt.Focused("Fonts") {
		t.Fatal("F did not focus Fonts")
	}
	wantTips(t, tt, map[string]string{})
	typeTip(tt, ui.KeyDown)
	if m := tt.Menu(); len(m) != 2 || m[0] != "Arial" {
		t.Fatalf("Down on Fonts shows the menu %q", m)
	}
}

// eventGate holds the event goroutine while set (holdEvents).
var eventGate chan struct{}

// holdEvents keeps the event goroutine busy until settleTips, as a loaded
// machine may (CI's Windows ARM64 did): what the tips post, as focusLater's
// focus, waits, and a test must wait for it, not for a frame.
func holdEvents(t *testing.T) {
	gate, held := make(chan struct{}), make(chan struct{})
	Post(func() { close(held); <-gate })
	<-held
	eventGate = gate
	t.Cleanup(releaseEvents)
}

func releaseEvents() {
	if eventGate != nil {
		close(eventGate)
		eventGate = nil
	}
}

// settleTips lets the event goroutine go and runs what was posted and a
// frame, until a round posts nothing, as widget.Tester.Settle does for a
// ui.Tester.
func settleTips(t *testing.T, tt *ui.Tester) {
	t.Helper()
	releaseEvents()
	for range 100 {
		before := postCount()
		done := make(chan struct{})
		Post(func() { close(done) })
		<-done
		tt.Frame()
		if postCount() == before+1 {
			return
		}
	}
	t.Fatal("still posting after 100 rounds")
}

// typeLate types the letter of k as a keyboard does, the key going down, its
// text coming, the key going up, with what the press posted run before the
// text comes: the latest a text can come.
func typeLate(tt *ui.Tester, k ui.Key, text string) {
	tt.KeyDown(k)
	Update(func() {})
	tt.Frame()
	tt.Type(text)
	tt.KeyUp(k)
	tt.Frame()
}

// Own case: the tips take the focus from a text box, so the letters typed
// reach them and not the box, and give it back as they end, once the
// letter that ended them is up: its text, coming after the press, does not
// reach the box.
func TestRibbonTipsTakeTheFocusFromATextBox(t *testing.T) {
	k, tt := tipKit(t)
	click(t, tt, "Body")
	typeTip(tt, ui.KeyF10)
	typeLate(tt, ui.KeyH, "h")
	typeLate(tt, ui.KeyP, "p")
	received(t, k.pasted)
	nothingRan(t, k)
	settleTips(t, tt)
	if s := k.body.Text(); s != "" {
		t.Fatalf("the text box took %q", s)
	}
	if !tt.Focused("Body") {
		t.Fatal("the focus did not go back to the text box")
	}
	// Escape gives it back too.
	typeTip(tt, ui.KeyF10)
	holdEvents(t)
	typeTip(tt, ui.KeyEscape)
	settleTips(t, tt)
	if !tt.Focused("Body") {
		t.Fatal("Escape did not give the focus back to the text box")
	}
}

// Own case: Alt+H on a collapsed ribbon expands it with Home's tips.
func TestRibbonTipsAltLetterOnACollapsedRibbon(t *testing.T) {
	k, tt := tipKit(t)
	k.r.SetCollapsed(true)
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyH)
	tt.Frame()
	if k.r.Collapsed() {
		t.Fatal("Alt+H left the ribbon collapsed")
	}
	wantTips(t, tt, homeTips)
}

// Own case (bound): of 36 commands the 36th has no tip.
func TestRibbonTipsThirtySixthCommandHasNone(t *testing.T) {
	r := Ribbon().AddTab("Home").AddGroup("G")
	for i := range 36 {
		r.AddSmall(ToolButton(fmt.Sprintf("#%d", i)))
	}
	withMac(t, false)
	tt := ui.NewTester(NewForm("Ribbon", 3000, 400).SetContent(r).view, 3000, 400)
	tt.Frame()
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	got := tipsRead(tt)
	if len(got) != 35 || got["#35"] != "" {
		t.Fatalf("%d tips, #35's %q: want 35, none on #35", len(got), got["#35"])
	}
}

// Own case: the window going inactive ends the tips.
func TestRibbonTipsEndWhenTheWindowGoesInactive(t *testing.T) {
	_, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	tt.SetWindowActive(false)
	tt.Frame()
	tt.SetWindowActive(true)
	tt.Frame()
	wantTips(t, tt, map[string]string{})
}

// Own case: a ribbon outside a form (widget.View) shows no tips.
func TestRibbonTipsNoneOutsideAForm(t *testing.T) {
	withMac(t, false)
	tt := tester(t, newRibbonKit(t).r)
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, map[string]string{})
}

// Own case: a hidden ribbon shows no tips, and F10 does not turn them on
// for later.
func TestRibbonTipsNoneForAHiddenRibbon(t *testing.T) {
	k, tt := tipKit(t)
	k.r.SetVisible(false)
	tt.Frame()
	typeTip(tt, ui.KeyF10)
	k.r.SetVisible(true)
	tt.Frame()
	wantTips(t, tt, map[string]string{})
}
