package widget

import (
	"image"
	"image/color"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// ribbonRed is the color of Paste's icon, a PNG of one color, so that its
// ink is its box.
var ribbonRed = color.RGBA{R: 220, A: 255}

// ribbonKit is the ribbon of the cases, in VBox(ribbon, TextBox()) in a
// form 800 × 400.
type ribbonKit struct {
	r                                *RibbonWidget
	paste, table, cut, copy, painter *ToolButtonWidget
	bold, italic, underline          *ToolButtonWidget
	size                             *ComboBoxWidget
	body                             *TextBoxWidget
	form                             *Form
	pasted, bolded, cutRan           chan struct{}
}

func newRibbonKit(t *testing.T) *ribbonKit {
	t.Helper()
	k := &ribbonKit{pasted: make(chan struct{}, 4), bolded: make(chan struct{}, 4), cutRan: make(chan struct{}, 4)}
	k.paste = ToolButton("Paste").SetIconPNG(testPNG(t, ribbonRed)).OnClick(func() { k.pasted <- struct{}{} })
	k.table = ToolButton("Table").SetIcon(chevronDown)
	k.cut = ToolButton("Cut").SetIcon(chevronDown).OnClick(func() { k.cutRan <- struct{}{} })
	k.copy = ToolButton("Copy").SetIcon(chevronDown)
	k.painter = ToolButton("Painter").SetIcon(chevronDown)
	k.bold = ToolButton("Bold").SetToggle(true).OnChange(func() { k.bolded <- struct{}{} })
	k.italic = ToolButton("Italic").SetToggle(true)
	k.underline = ToolButton("Underline").SetToggle(true)
	k.size = ComboBox(Option{"11", "11"}, Option{"12", "12"}).SetValue("11").SetAccessibleName("Size")
	k.body = TextBox().SetAccessibleName("Body")
	k.r = Ribbon().AddTab("Home").
		AddGroup("Clipboard").AddLarge(k.paste).AddSmall(k.cut, k.copy, k.painter).
		AddGroup("Font").AddSmall(k.bold, k.italic, k.underline, k.size).
		AddTab("Insert").AddGroup("Tables").AddLarge(k.table)
	k.form = NewForm("Ribbon", 800, 400).SetContent(VBox(0).Add(k.r, k.body))
	return k
}

func (k *ribbonKit) tester(w int) *ui.Tester {
	tt := ui.NewTester(k.form.view, w, 400)
	tt.Frame()
	return tt
}

// rnode returns the first node labelled label, of any role.
func rnode(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Label == label {
			return n.Bounds
		}
	}
	t.Fatalf("no node labelled %q in %q", label, tt.Texts())
	return ui.Rect{}
}

// countNodes counts the nodes of role r labelled label.
func countNodes(tt *ui.Tester, r ui.Role, label string) int {
	n := 0
	for _, x := range tt.Nodes() {
		if x.Role == r && x.Label == label {
			n++
		}
	}
	return n
}

// ribbonRedInk returns the box of the pixels in r near ribbonRed: Paste's icon.
func ribbonRedInk(img *image.RGBA, r ui.Rect) (box ui.Rect, ok bool) {
	x0, y0, x1, y1 := math.MaxInt, math.MaxInt, -1, -1
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := img.RGBAAt(x, y)
			if c.R > 180 && c.G < 60 && c.B < 60 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: float32(x0), Y: float32(y0), W: float32(x1 - x0 + 1), H: float32(y1 - y0 + 1)}, true
}

func near1(a, b float32) bool { return math.Abs(float64(a-b)) <= 1.01 }

// 1. The tabs, the shown tab's groups with their titles under their items,
// and not the other tab's.
func TestRibbonShowsTabsAndTheShownTabsGroups(t *testing.T) {
	tt := newRibbonKit(t).tester(800)
	for _, s := range []string{"Home", "Insert"} {
		if countNodes(tt, ui.RoleTab, s) != 1 {
			t.Fatalf("no tab %q in %+v", s, tt.Texts())
		}
	}
	paste := nodeOf(t, tt, ui.RoleButton, "Paste").Bounds
	for _, g := range []string{"Clipboard", "Font"} {
		title := nodeOf(t, tt, ui.RoleText, g).Bounds
		if title.Y < paste.Y+paste.H-0.5 {
			t.Fatalf("the title %q at %v is not under the items (Paste ends at y %v)", g, title, paste.Y+paste.H)
		}
	}
	if tt.HasText("Tables") || tt.HasText("Table") {
		t.Fatalf("the Insert tab's group shows on Home: %q", tt.Texts())
	}
	// Without a skin the shown tab is filled as the groups row under it.
	home, clip := nodeOf(t, tt, ui.RoleTab, "Home").Bounds, nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds
	tab, row := tt.Image().RGBAAt(int(home.X+3), int(home.Y+home.H/2)), tt.Image().RGBAAt(790, int(clip.Y+clip.H/2))
	if tab != row || !near(row, ui.LightTheme().Background) {
		t.Fatalf("the shown tab is filled %v, the groups row %v: want both Background", tab, row)
	}
}

// 2. Paste is large, its 32 DIP icon above its text and centred; Cut, Copy
// and Painter stack in one column right of it.
func TestRibbonLargeButtonAndASmallColumn(t *testing.T) {
	tt := newRibbonKit(t).tester(800)
	paste := nodeOf(t, tt, ui.RoleButton, "Paste").Bounds
	icon, ok := ribbonRedInk(tt.Image(), paste)
	if !ok {
		t.Fatalf("no icon in Paste at %v", paste)
	}
	if !near1(icon.H, 32) || !near1(icon.W, 32) {
		t.Fatalf("Paste's icon is %v × %v, want 32 × 32", icon.W, icon.H)
	}
	if !near1(icon.X+icon.W/2, paste.X+paste.W/2) {
		t.Fatalf("the icon's centre x %v is not Paste's %v", icon.X+icon.W/2, paste.X+paste.W/2)
	}
	text := pasteText(t, tt, paste, icon)
	if text.Y < icon.Y+icon.H {
		t.Fatalf("Paste's text at y %v is not below its icon, which ends at %v", text.Y, icon.Y+icon.H)
	}
	if math.Abs(float64(text.X+text.W/2-(icon.X+icon.W/2))) > 1.5 {
		t.Fatalf("the text's ink centre x %v is not the icon's %v", text.X+text.W/2, icon.X+icon.W/2)
	}
	cut, cp, painter := nodeOf(t, tt, ui.RoleButton, "Cut").Bounds, nodeOf(t, tt, ui.RoleButton, "Copy").Bounds, nodeOf(t, tt, ui.RoleButton, "Painter").Bounds
	if !near1(cut.X, cp.X) || !near1(cp.X, painter.X) {
		t.Fatalf("Cut, Copy and Painter at x %v %v %v: not one column", cut.X, cp.X, painter.X)
	}
	if cut.X < paste.X+paste.W {
		t.Fatalf("Cut at x %v is not right of Paste, which ends at %v", cut.X, paste.X+paste.W)
	}
	if cp.Y < cut.Y+cut.H-0.5 || painter.Y < cp.Y+cp.H-0.5 {
		t.Fatalf("Cut %v, Copy %v, Painter %v: not each below the one before", cut, cp, painter)
	}
	if cut.H > paste.H/2 {
		t.Fatalf("Cut, a small button, is %v high, Paste %v: drawn large", cut.H, paste.H)
	}
}

// pasteText returns the ink of Paste's text: what differs from the button's
// fill under its icon.
func pasteText(t *testing.T, tt *ui.Tester, paste, icon ui.Rect) ui.Rect {
	t.Helper()
	img := tt.Image()
	bg := img.RGBAAt(int(paste.X+2), int(paste.Y+paste.H-2))
	differs := func(a, b uint8) bool { d := int(a) - int(b); return d > 40 || d < -40 }
	x0, y0, x1, y1 := math.MaxInt, math.MaxInt, -1, -1
	for y := int(icon.Y + icon.H + 1); y < int(paste.Y+paste.H-1); y++ {
		for x := int(paste.X + 1); x < int(paste.X+paste.W-1); x++ {
			if c := img.RGBAAt(x, y); differs(c.R, bg.R) || differs(c.G, bg.G) || differs(c.B, bg.B) {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		t.Fatalf("no text under Paste's icon in %v", paste)
	}
	return ui.Rect{X: float32(x0), Y: float32(y0), W: float32(x1 - x0 + 1), H: float32(y1 - y0 + 1)}
}

// 3. Font: Bold, Italic and Underline in one column, Size at the top of a
// second.
func TestRibbonSmallControlsFillColumnsOfThree(t *testing.T) {
	tt := newRibbonKit(t).tester(800)
	bold, italic, under, size := rnode(t, tt, "Bold"), rnode(t, tt, "Italic"), rnode(t, tt, "Underline"), rnode(t, tt, "Size")
	if !near1(bold.X, italic.X) || !near1(italic.X, under.X) {
		t.Fatalf("Bold, Italic and Underline at x %v %v %v: not one column", bold.X, italic.X, under.X)
	}
	if italic.Y < bold.Y+bold.H-0.5 || under.Y < italic.Y+italic.H-0.5 {
		t.Fatalf("Bold %v, Italic %v, Underline %v: not each below the one before", bold, italic, under)
	}
	if size.X < bold.X+bold.W-0.5 {
		t.Fatalf("Size at x %v is not in a column right of Bold's, which ends at %v", size.X, bold.X+bold.W)
	}
	if size.Y >= italic.Y {
		t.Fatalf("Size at y %v is not at the top of its column (Italic, the second row, at %v)", size.Y, italic.Y)
	}
}

// 4. A click runs Paste's OnClick once, toggles Bold and runs its OnChange
// once; a disabled Cut runs nothing.
func TestRibbonClicksRunTheirHandlersOnce(t *testing.T) {
	k := newRibbonKit(t)
	k.cut.SetDisabled(true)
	tt := k.tester(800)
	click(t, tt, "Paste")
	received(t, k.pasted)
	click(t, tt, "Bold")
	received(t, k.bolded)
	if !k.bold.Checked() {
		t.Fatal("clicking Bold did not check it")
	}
	_ = tt.Click("Cut") // disabled: it may not even be a target
	tt.Frame()
	Update(func() {}) // the handlers posted so far have run
	if n, m, c := len(k.pasted), len(k.bolded), len(k.cutRan); n != 0 || m != 0 || c != 0 {
		t.Fatalf("Paste ran %d more times, Bold's OnChange %d, the disabled Cut %d: want 0", n, m, c)
	}
}

// 5. Every group has one height, and Paste fills its group's, less the
// title: from the top of Cut, the first small row, to the bottom of
// Painter, the third, or more.
func TestRibbonGroupsShareOneHeight(t *testing.T) {
	tt := newRibbonKit(t).tester(800)
	clip, font := nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds, nodeOf(t, tt, ui.RoleGroup, "Font").Bounds
	if !near1(clip.H, font.H) || !near1(clip.Y, font.Y) {
		t.Fatalf("Clipboard %v and Font %v: not one height", clip, font)
	}
	paste, cut, painter := rnode(t, tt, "Paste"), rnode(t, tt, "Cut"), rnode(t, tt, "Painter")
	title := nodeOf(t, tt, ui.RoleText, "Clipboard").Bounds
	if paste.Y > cut.Y+0.5 || paste.Y+paste.H < painter.Y+painter.H-0.5 {
		t.Fatalf("Paste %v does not span Cut %v to Painter %v", paste, cut, painter)
	}
	if paste.Y+paste.H > title.Y+0.5 {
		t.Fatalf("Paste %v reaches into the title %v", paste, title)
	}
	if paste.H < clip.H-title.H-24 {
		t.Fatalf("Paste is %v high in a group %v high with a title %v high: it does not fill the group", paste.H, clip.H, title.H)
	}
}

// 6. SetTab shows the tab; a click on a tab shows it back.
func TestRibbonSetTabAndClickingATab(t *testing.T) {
	k := newRibbonKit(t)
	tt := k.tester(800)
	k.r.SetTab(1)
	tt.Frame()
	if !tt.HasText("Tables") || tt.HasText("Clipboard") {
		t.Fatalf("SetTab(1) shows %q", tt.Texts())
	}
	if k.r.Tab() != 1 {
		t.Fatalf("Tab() = %d, want 1", k.r.Tab())
	}
	click(t, tt, "Home")
	if !tt.HasText("Clipboard") || tt.HasText("Tables") {
		t.Fatalf("a click on Home shows %q", tt.Texts())
	}
	if k.r.Tab() != 0 {
		t.Fatalf("Tab() = %d after the click on Home, want 0", k.r.Tab())
	}
}

// Own case: switching tabs keeps the groups' height, and so the content's
// place, though Tables holds one large button and no small one.
func TestRibbonTabsKeepTheirHeight(t *testing.T) {
	k := newRibbonKit(t)
	tt := k.tester(800)
	clip, body := nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds, rnode(t, tt, "Body")
	k.r.SetTab(1)
	tt.Frame()
	tables, body1 := nodeOf(t, tt, ui.RoleGroup, "Tables").Bounds, rnode(t, tt, "Body")
	if !near1(tables.H, clip.H) || !near1(body1.Y, body.Y) {
		t.Fatalf("Tables %v, Clipboard %v; the content at y %v, then %v: the tabs differ in height", tables, clip, body.Y, body1.Y)
	}
}

// 7. Collapsed, the tabs alone show and the content moves up; a click on a
// tab shows it and expands; a double click collapses; Ctrl+F1 toggles with
// the focus in the content.
func TestRibbonCollapses(t *testing.T) {
	k := newRibbonKit(t)
	tt := k.tester(800)
	body, clip := rnode(t, tt, "Body"), nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds
	k.r.SetCollapsed(true)
	tt.Frame()
	if !k.r.Collapsed() {
		t.Fatal("Collapsed() is false after SetCollapsed(true)")
	}
	if tt.HasText("Clipboard") || tt.HasText("Paste") || !tt.HasText("Home") || !tt.HasText("Insert") {
		t.Fatalf("collapsed, it shows %q", tt.Texts())
	}
	home, body1 := nodeOf(t, tt, ui.RoleTab, "Home").Bounds, rnode(t, tt, "Body")
	if body1.Y > body.Y-clip.H || body1.Y < home.Y+home.H-0.5 {
		t.Fatalf("collapsed, the content is at y %v (was %v, the groups %v high, the tabs end at %v)", body1.Y, body.Y, clip.H, home.Y+home.H)
	}
	click(t, tt, "Insert")
	if !tt.HasText("Tables") || k.r.Collapsed() || k.r.Tab() != 1 {
		t.Fatalf("a click on Insert, collapsed: texts %q, Collapsed %v, Tab %d", tt.Texts(), k.r.Collapsed(), k.r.Tab())
	}
	if err := tt.Click("Home"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Home"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !k.r.Collapsed() || tt.HasText("Clipboard") {
		t.Fatalf("a double click on Home: Collapsed %v, texts %q", k.r.Collapsed(), tt.Texts())
	}
	click(t, tt, "Body")
	if !tt.Focused("Body") {
		t.Fatal("the text box did not take the focus")
	}
	tt.Key(ui.Ctrl, ui.KeyF1)
	tt.Frame()
	if k.r.Collapsed() || !tt.HasText("Clipboard") {
		t.Fatalf("Ctrl+F1 did not expand: Collapsed %v, texts %q", k.r.Collapsed(), tt.Texts())
	}
	tt.Key(ui.Ctrl, ui.KeyF1)
	tt.Frame()
	if !k.r.Collapsed() {
		t.Fatal("Ctrl+F1 again did not collapse")
	}
}

// Review Focus 5: collapsed, one click on a tab both selects and expands.
func TestRibbonCollapsedClickSelectsAndExpands(t *testing.T) {
	k := newRibbonKit(t)
	k.r.SetCollapsed(true)
	tt := k.tester(800)
	if err := tt.Click("Insert"); err != nil {
		t.Fatal(err)
	}
	if k.r.Tab() != 1 || k.r.Collapsed() {
		t.Fatalf("after one click on Insert: Tab %d, Collapsed %v", k.r.Tab(), k.r.Collapsed())
	}
	tt.Frame()
	if !tt.HasText("Tables") {
		t.Fatalf("the next frame shows %q", tt.Texts())
	}
	// The tab already shown expands too. The clock moves on: a second click
	// at once would be a double click.
	k.r.SetCollapsed(true)
	tt.Advance(time.Second)
	tt.Frame()
	click(t, tt, "Insert")
	if k.r.Collapsed() || !tt.HasText("Tables") {
		t.Fatalf("a click on the tab shown, collapsed: Collapsed %v, texts %q", k.r.Collapsed(), tt.Texts())
	}
}

// Own case: a double click on a tab of a collapsed ribbon leaves it
// expanded, as decided at the first click: the first click expands, and the
// second, the double, is not a collapse of what it just expanded.
func TestRibbonDoubleClickWhileCollapsedExpands(t *testing.T) {
	k := newRibbonKit(t)
	k.r.SetCollapsed(true)
	tt := k.tester(800)
	for range 2 {
		if err := tt.Click("Home"); err != nil {
			t.Fatal(err)
		}
	}
	tt.Frame()
	if k.r.Collapsed() || !tt.HasText("Clipboard") {
		t.Fatalf("a double click on Home, collapsed: Collapsed %v, texts %q", k.r.Collapsed(), tt.Texts())
	}
	// And a click after it is a click: it does nothing to an expanded ribbon.
	tt.Advance(time.Second)
	click(t, tt, "Home")
	if k.r.Collapsed() {
		t.Fatal("a single click after the double click collapsed the ribbon")
	}
}

// 8. Items before any group go to a group with no title, and a group before
// any tab to a tab with no title.
func TestRibbonAddsAGroupAndATabWithNoTitle(t *testing.T) {
	r := Ribbon().AddLarge(ToolButton("A").SetIcon(chevronDown)).AddSmall(ToolButton("B"))
	tt := tester(t, r)
	find(t, tt, "A")
	find(t, tt, "B")
	for _, s := range tt.Texts() {
		if s != "A" && s != "B" && s != "" {
			t.Fatalf("text %q in a ribbon of no title: %q", s, tt.Texts())
		}
	}
	a, b := rnode(t, tt, "A"), rnode(t, tt, "B")
	if b.X < a.X+a.W-0.5 {
		t.Fatalf("B %v is not right of A %v: not one group", b, a)
	}
	tt = tester(t, Ribbon().AddGroup("G").AddSmall(ToolButton("C")))
	find(t, tt, "C")
	if countNodes(tt, ui.RoleGroup, "G") != 1 || countNodes(tt, ui.RoleText, "G") != 1 {
		t.Fatalf("no group G titled G: %q", tt.Texts())
	}
}

// 9. AddLarge and AddSmall with nothing add nothing; SetTab out of range is
// as TabControl.SetPage: kept until a frame, then clamped.
func TestRibbonAddsNothingAndClampsTheTab(t *testing.T) {
	r := Ribbon().AddLarge().AddSmall()
	tt := tester(t, r)
	if n := countNodes(tt, ui.RoleTab, ""); n != 0 {
		t.Fatalf("AddLarge() and AddSmall() made %d tabs", n)
	}
	r.AddSmall(nil) // a nil control is no control: nothing to add, nothing to panic on
	tt.Frame()
	for _, i := range []int{5, -1} {
		k := newRibbonKit(t)
		nb := TabControl().AddPage("Home", Label("h")).AddPage("Insert", Label("i"))
		k.r.SetTab(i)
		nb.SetPage(i)
		if k.r.Tab() != nb.Page() {
			t.Fatalf("SetTab(%d): Tab() = %d before a frame, SetPage's Page() %d", i, k.r.Tab(), nb.Page())
		}
		k.tester(800)
		tester(t, nb)
		if k.r.Tab() != nb.Page() {
			t.Fatalf("SetTab(%d): Tab() = %d after a frame, SetPage's Page() %d", i, k.r.Tab(), nb.Page())
		}
	}
}

// 10. Groups are groups named by their titles, Paste a button named Paste;
// Focus focuses the tab shown, and Tab goes from it into the groups.
func TestRibbonAccess(t *testing.T) {
	k := newRibbonKit(t)
	tt := k.tester(800)
	nodeOf(t, tt, ui.RoleGroup, "Clipboard")
	nodeOf(t, tt, ui.RoleGroup, "Font")
	nodeOf(t, tt, ui.RoleButton, "Paste")
	k.r.Focus()
	tt.Frame()
	if n := nodeOf(t, tt, ui.RoleTab, "Home"); !n.Focused {
		t.Fatal("Focus did not focus the Home tab")
	}
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if !tt.Focused("Paste") {
		var on []string
		for _, n := range tt.Nodes() {
			if n.Focused {
				on = append(on, n.Label)
			}
		}
		t.Fatalf("Tab from the tabs went to %q, not Paste, the first control of the tab shown", on)
	}
}

// 11. A tool button after a ribbon, outside it, draws small: the flag that
// draws large is not left set.
func TestRibbonLeavesToolButtonsOutsideSmall(t *testing.T) {
	out := ToolButton("Out").SetIconPNG(testPNG(t, ribbonRed))
	s := ToolButton("S").SetIconPNG(testPNG(t, ribbonRed))
	r := Ribbon().AddTab("Home").AddGroup("G").AddLarge(ToolButton("L").SetIcon(chevronDown)).AddSmall(s)
	tt := tester(t, VBox(0).Add(r, ToolStrip(out)))
	for _, name := range []string{"S", "Out"} {
		b := nodeOf(t, tt, ui.RoleButton, name).Bounds
		icon, ok := ribbonRedInk(tt.Image(), b)
		if !ok {
			t.Fatalf("no icon in %s at %v", name, b)
		}
		if icon.H > 24 {
			t.Fatalf("%s's icon is %v high: drawn large after the ribbon's large item", name, icon.H)
		}
	}
}

// 12. Under each built-in skin, light and dark, the groups row fills with
// the skin's Chrome, Surface without one, and the group titles' text,
// TextMuted, reaches 4.5:1 on it.
func TestRibbonFillsWithChrome(t *testing.T) {
	check := func(t *testing.T, fill, muted ui.Color) {
		t.Helper()
		k := newRibbonKit(t)
		tt := k.tester(800)
		font := nodeOf(t, tt, ui.RoleGroup, "Font").Bounds
		if c := tt.Image().RGBAAt(790, int(font.Y+font.H/2)); !near(c, fill) {
			t.Fatalf("the groups row past the last group is %v, want %v", c, fill)
		}
		if r := ratio(muted, fill); r < 4.5 {
			t.Fatalf("the titles' TextMuted %v on %v is %.2f:1, want 4.5:1", muted, fill, r)
		}
	}
	t.Run("none", func(t *testing.T) { th := ui.LightTheme(); check(t, th.Background, th.TextMuted) })
	for _, name := range []string{"Calm", "Classic", "HighContrast"} {
		for _, dark := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "-light", true: "-dark"}[dark], func(t *testing.T) {
				useSkinNamed(t, name, false)
				tk := skinhook.Get(dark)
				k := newRibbonKit(t)
				tt := ui.NewTester(k.form.view, 800, 400)
				tt.SetDark(dark)
				tt.Frame()
				font := nodeOf(t, tt, ui.RoleGroup, "Font").Bounds
				if c := tt.Image().RGBAAt(790, int(font.Y+font.H/2)); !near(c, tk.Chrome) {
					t.Fatalf("the groups row past the last group is %v, want Chrome %v", c, tk.Chrome)
				}
				if r := ratio(tk.TextMuted, tk.Chrome); r < 4.5 {
					t.Fatalf("the titles' TextMuted on Chrome is %.2f:1, want 4.5:1", r)
				}
			})
		}
	}
}

// ribbonRef is case 13's reference: a ui.TabsKeyed strip over a body.
type ribbonRef struct {
	common[ribbonRef]
	sel    int
	keys   []any
	labels []string
	body   Control
}

func (x *ribbonRef) build(c *ui.Context) ui.Element {
	return ui.Column(c).Children(func() {
		ui.TabsKeyed(c, &x.sel, x.keys, x.labels)
		render(c, x.body)
	})
}

// 13. A ribbon's frame at rest allocates no more than the same controls in
// a VBox of HBoxes under a ui.TabsKeyed strip.
func TestRibbonFrameAllocatesNoMoreThanBoxes(t *testing.T) {
	k := newRibbonKit(t)
	ribbon := ui.NewTester(content(VBox(0).Add(k.r, k.body)), 800, 400)
	q := newRibbonKit(t)
	ref := &ribbonRef{keys: []any{1, 2}, labels: []string{"Home", "Insert"},
		body: HBox(0).Add(HBox(0).Add(q.paste, VBox(0).Add(q.cut, q.copy, q.painter)),
			HBox(0).Add(VBox(0).Add(q.bold, q.italic, q.underline), VBox(0).Add(q.size)))}
	ref.init(ref)
	boxes := ui.NewTester(content(VBox(0).Add(ref, q.body)), 800, 400)
	for range 10 {
		ribbon.Frame()
		boxes.Frame()
	}
	got, want := allocrun.Least(30, ribbon.Frame), allocrun.Least(30, boxes.Frame)
	if got > want {
		t.Fatalf("a ribbon's frame allocates %v, the same controls in boxes %v", got, want)
	}
}

// 14. Changes from another goroutine while frames run show on a following
// frame (run it under -race).
func TestRibbonChangesFromAnotherGoroutine(t *testing.T) {
	r := Ribbon().AddTab("Home").AddGroup("G").AddSmall(ToolButton("A"))
	tt := tester(t, VBox(0).Add(r, Label("body")))
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.AddTab("Late")
		r.AddGroup("LateGroup")
		r.AddSmall(ToolButton("LateButton"))
		r.SetCollapsed(true)
		r.SetTab(1)
		r.SetCollapsed(false)
	}()
	for range 20 {
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	if r.Tab() != 1 || r.Collapsed() || !tt.HasText("LateButton") || !tt.HasText("Late") {
		t.Fatalf("Tab %d, Collapsed %v, texts %q", r.Tab(), r.Collapsed(), tt.Texts())
	}
}

// Review Focus 1: at text scale 1.5 the large icon is 48 DIPs, the groups
// keep one height, and three small rows still fit.
func TestRibbonGrowsWithTheTextScale(t *testing.T) {
	k := newRibbonKit(t)
	tt := ui.NewTester(k.form.view, 1000, 500)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	paste := nodeOf(t, tt, ui.RoleButton, "Paste").Bounds
	icon, ok := ribbonRedInk(tt.Image(), paste)
	if !ok || !near1(icon.H, 48) {
		t.Fatalf("at text scale 1.5, Paste's icon is %v high, want 48", icon.H)
	}
	clip, font := nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds, nodeOf(t, tt, ui.RoleGroup, "Font").Bounds
	if !near1(clip.H, font.H) {
		t.Fatalf("Clipboard %v and Font %v high: not one height", clip.H, font.H)
	}
	under, title := rnode(t, tt, "Underline"), nodeOf(t, tt, ui.RoleText, "Font").Bounds
	if under.Y+under.H > title.Y+0.5 {
		t.Fatalf("Underline, the third row, ends at %v, into the title at %v", under.Y+under.H, title.Y)
	}
	painter := rnode(t, tt, "Painter")
	if painter.Y+painter.H > paste.Y+paste.H+0.5 {
		t.Fatalf("Painter ends at %v, below Paste at %v: three rows do not fit", painter.Y+painter.H, paste.Y+paste.H)
	}
}

// Review Focus 2: in a form 400 wide the groups row clips at the right and
// nothing overlaps; a long large text wraps to two lines at most, no wider
// than twice the icon.
func TestRibbonClipsInANarrowForm(t *testing.T) {
	k := newRibbonKit(t)
	wide := k.tester(800)
	cut0 := rnode(t, wide, "Cut")
	tt := k.tester(400)
	cut, body := rnode(t, tt, "Cut"), rnode(t, tt, "Body")
	if cut != cut0 {
		t.Fatalf("Cut at %v in a narrow form, %v in a wide one: the groups shrank", cut, cut0)
	}
	clip, font := nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds, nodeOf(t, tt, ui.RoleGroup, "Font").Bounds
	if font.X < clip.X+clip.W-0.5 || body.Y < clip.Y+clip.H-0.5 {
		t.Fatalf("Clipboard %v, Font %v, the content %v: they overlap", clip, font, body)
	}
	if font.X+font.W > 400.5 {
		t.Fatalf("Font %v reaches past the form's 400", font)
	}
	// The long text against an icon alone, 32 wide, in the same padding.
	const text = "Insert a very long table of contents here"
	long, only := ToolButton(text).SetIconPNG(testPNG(t, ribbonRed)), ToolButton("").SetIcon(chevronDown).SetTooltip("Only")
	tt = tester(t, Ribbon().AddLarge(long, only))
	b, o := rnode(t, tt, text), rnode(t, tt, "Only")
	if b.W-o.W > 32+0.5 {
		t.Fatalf("the long large button is %v wide, the icon-only one %v: its text is wider than twice the icon", b.W, o.W)
	}
	icon, _ := ribbonRedInk(tt.Image(), b)
	ink := pasteText(t, tt, b, icon)
	if line := ui.LightTheme().Rem(1) * 1.6; ink.H > 2*line {
		t.Fatalf("the long text's ink is %v high, over two lines of %v", ink.H, line)
	}
	if ink.H < ui.LightTheme().Rem(1)*1.2 {
		t.Fatalf("the long text's ink is %v high: it did not wrap", ink.H)
	}
}

// Review Focus 3: a DropDownButton among the small controls opens its menu.
func TestRibbonDropDownOpensItsMenu(t *testing.T) {
	r := Ribbon().AddTab("Home").AddGroup("G").AddSmall(DropDownButton("Fonts", MenuItem{Text: "Arial"}, MenuItem{Text: "Serif"}))
	tt := tester(t, r)
	click(t, tt, "Fonts")
	if m := tt.Menu(); len(m) != 2 || m[0] != "Arial" {
		t.Fatalf("menu %q", m)
	}
}

// Review Focus 4: a control the ribbon holds is not added again, and one in
// a ribbon and a ToolStrip of one form shows in both without a panic.
func TestRibbonRefusesAControlItHolds(t *testing.T) {
	k := newRibbonKit(t)
	k.r.AddSmall(k.cut, k.size).AddLarge(k.paste)
	x := ToolButton("Twice")
	k.r.AddSmall(x, x)
	k.form.ToolStrip(ToolStrip(k.copy))
	tt := k.tester(800)
	for _, s := range []string{"Cut", "Paste"} {
		if n := countNodes(tt, ui.RoleButton, s); n != 1 {
			t.Fatalf("%d buttons %q, want 1", n, s)
		}
	}
	if n := countNodes(tt, ui.RoleButton, "Copy"); n != 2 {
		t.Fatalf("%d buttons Copy, want 2: the ribbon's and the tool strip's", n)
	}
	k.r.SetTab(1) // Twice went to the last group, Tables
	tt.Frame()
	if n := countNodes(tt, ui.RoleButton, "Twice"); n != 1 {
		t.Fatalf("%d buttons Twice, want 1", n)
	}
	if n := countNodes(tt, ui.RoleButton, "Table"); n != 1 {
		t.Fatalf("%d buttons Table, want 1", n)
	}
}

// Own case: a ribbon with no tab builds nothing and does not panic.
func TestRibbonWithNoTab(t *testing.T) {
	tt := tester(t, VBox(0).Add(Ribbon(), Label("body")))
	if texts := tt.Texts(); len(texts) != 1 || texts[0] != "body" {
		t.Fatalf("texts %q", texts)
	}
}
