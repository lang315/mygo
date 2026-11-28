package widget

import (
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// The menu items' English text, as internal/l10n's en.json has them.
const (
	addQAT    = "Add to Quick Access Toolbar"
	removeQAT = "Remove from Quick Access Toolbar"
	belowQAT  = "Show below the ribbon"
)

// qatKit is the ribbon kit with Undo and Redo, icon buttons that are no
// command of the ribbon, to put in the Quick Access Toolbar, and a channel
// for OnQuickAccessChange.
type qatKit struct {
	*ribbonKit
	undo, redo         *ToolButtonWidget
	undone, redone     chan struct{}
	changes            chan struct{}
	undoIcon, redoIcon *ui.SVG
}

func newQATKit(t *testing.T) *qatKit {
	t.Helper()
	k := &qatKit{ribbonKit: newRibbonKit(t), undone: make(chan struct{}, 4), redone: make(chan struct{}, 4), changes: make(chan struct{}, 8)}
	k.undo = ToolButton("Undo").SetIcon(chevronDown).OnClick(func() { k.undone <- struct{}{} })
	k.redo = ToolButton("Redo").SetIcon(chevronDown).OnClick(func() { k.redone <- struct{}{} })
	k.r.OnQuickAccessChange(func() { k.changes <- struct{}{} })
	return k
}

// allBounds returns the bounds of the nodes of role r labelled label, in tree
// order.
func allBounds(tt *ui.Tester, r ui.Role, label string) []ui.Rect {
	var out []ui.Rect
	for _, n := range tt.Nodes() {
		if n.Role == r && n.Label == label {
			out = append(out, n.Bounds)
		}
	}
	return out
}

// qatButton returns the button (or toggle) labelled label that is above the
// tab Home: the one in the Quick Access Toolbar.
func qatButton(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	home := nodeOf(t, tt, ui.RoleTab, "Home").Bounds
	for _, n := range tt.Nodes() {
		if b := n.Bounds; n.Label == label && n.Role != ui.RoleText && b.Y+b.H <= home.Y+0.5 {
			return b
		}
	}
	t.Fatalf("no button %q above the tabs (at %v): %q", label, home, layoutOf(tt))
	return ui.Rect{}
}

// noChange checks that OnQuickAccessChange did not run (again).
func noChange(t *testing.T, k *qatKit) {
	t.Helper()
	Update(func() {})
	if n := len(k.changes); n != 0 {
		t.Fatalf("OnQuickAccessChange ran %d more times, want 0", n)
	}
}

func centre(r ui.Rect) (float32, float32) { return r.X + r.W/2, r.Y + r.H/2 }

// R1. Two QAT buttons in a row above the tabs, Undo first; with
// SetQuickAccessBelow(true), below the groups. A click runs each button's
// OnClick once.
func TestRibbonQATShowsAboveTabs(t *testing.T) {
	k := newQATKit(t)
	k.r.AddQuickAccess(k.undo, k.redo)
	tt := k.tester(800)
	undo, redo := qatButton(t, tt, "Undo"), qatButton(t, tt, "Redo")
	if !near1(undo.Y, redo.Y) || redo.X < undo.X+undo.W-0.5 {
		t.Fatalf("Undo %v and Redo %v: not one row, Undo first", undo, redo)
	}
	if home := nodeOf(t, tt, ui.RoleTab, "Home").Bounds; undo.H > home.H+0.5 {
		t.Fatalf("Undo is %v high, the tab %v: not a small icon", undo.H, home.H)
	}
	click(t, tt, "Undo")
	received(t, k.undone)
	click(t, tt, "Redo")
	received(t, k.redone)
	Update(func() {})
	if len(k.undone) != 0 || len(k.redone) != 0 {
		t.Fatalf("Undo ran %d more times, Redo %d", len(k.undone), len(k.redone))
	}
	k.r.SetQuickAccessBelow(true)
	tt.Frame()
	clip := nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds
	undo, redo = rnode(t, tt, "Undo"), rnode(t, tt, "Redo")
	if undo.Y < clip.Y+clip.H-0.5 || !near1(undo.Y, redo.Y) || redo.X < undo.X+undo.W-0.5 {
		t.Fatalf("below: Undo %v, Redo %v, the groups end at %v", undo, redo, clip.Y+clip.H)
	}
	if body := rnode(t, tt, "Body"); body.Y < undo.Y+undo.H-0.5 {
		t.Fatalf("below: the content at %v is not under the QAT %v", body, undo)
	}
	noChange(t, k)
}

// R2. A right-click on a large command offers "Add…": it shows in the QAT,
// and OnQuickAccessChange runs once. A right-click on it in the QAT offers
// "Remove…" and "Show below…"; Remove takes it out, and the callback runs
// once more.
func TestRibbonQATAddRemoveByMenu(t *testing.T) {
	k := newQATKit(t)
	tt := k.tester(800)
	if err := tt.RightClick("Paste"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{addQAT}) {
		t.Fatalf("Paste's menu %q, want %q", m, addQAT)
	}
	if err := tt.ChooseMenuItem(addQAT); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	received(t, k.changes)
	noChange(t, k)
	if n := countNodes(tt, ui.RoleButton, "Paste"); n != 2 {
		t.Fatalf("%d buttons Paste after Add, want 2: the group's and the QAT's", n)
	}
	if got := k.r.QuickAccess(); !slices.Equal(got, []string{"Paste"}) {
		t.Fatalf("QuickAccess() = %q", got)
	}
	tt.RightClickAt(centre(qatButton(t, tt, "Paste")))
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{removeQAT, belowQAT}) {
		t.Fatalf("the QAT button's menu %q, want %q", m, []string{removeQAT, belowQAT})
	}
	if err := tt.ChooseMenuItem(removeQAT); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	received(t, k.changes)
	noChange(t, k)
	if n := countNodes(tt, ui.RoleButton, "Paste"); n != 1 {
		t.Fatalf("%d buttons Paste after Remove, want 1", n)
	}
	if got := k.r.QuickAccess(); len(got) != 0 {
		t.Fatalf("QuickAccess() = %q after Remove", got)
	}
	// A small command offers it too.
	if err := tt.RightClick("Cut"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{addQAT}) {
		t.Fatalf("Cut's menu %q, want %q", m, addQAT)
	}
	tt.CloseMenu()
	// A control that is no tool button, the Size combo, offers none.
	tt.RightClickAt(centre(rnode(t, tt, "Size")))
	tt.Frame()
	if m := tt.Menu(); slices.Contains(m, addQAT) {
		t.Fatalf("the Size combo's menu %q offers %q", m, addQAT)
	}
	tt.CloseMenu()
}

// Own case: a ribbon with no OnQuickAccessChange adds from the menu all the
// same.
func TestRibbonQATAddWithNoCallback(t *testing.T) {
	k := newRibbonKit(t)
	k.r.AddQuickAccess()
	tt := k.tester(800)
	if err := tt.RightClick("Paste"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.ChooseMenuItem(addQAT); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	Update(func() {})
	if got := k.r.QuickAccess(); !slices.Equal(got, []string{"Paste"}) {
		t.Fatalf("QuickAccess() = %q", got)
	}
}

// Own case: in a shown window, the QAT button's "Show below the ribbon" is a
// check item, checked while the QAT is below.
func TestRibbonQATShowBelowIsChecked(t *testing.T) {
	for _, below := range []bool{false, true} {
		save := ToolButton("Save").SetIcon(chevronDown)
		r := Ribbon().AddTab("Home").AddGroup("G").AddSmall(ToolButton("A")).AddQuickAccess(save).SetQuickAccessBelow(below)
		_, _ = showFrame(t, VBox(0).Add(r, Label("body")))
		fw := lastFake(t)
		settleFake(fw)
		var at ui.Rect
		for _, n := range accessNodes(fw) {
			if n.Label == "Save" {
				at = ui.Rect{X: float32(n.Bounds.X), Y: float32(n.Bounds.Y), W: float32(n.Bounds.W), H: float32(n.Bounds.H)}
			}
		}
		before := len(fb.Popups())
		x, y := centre(at)
		mygo.RunOnMain(func() {
			s := fw.FakeSurface()
			s.Send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y), Button: 1})
			s.Send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y), Button: 1})
			s.Frame()
		})
		mygo.RunOnMain(func() {}) // the menu, posted
		popups := fb.Popups()
		if len(popups) != before+1 {
			t.Fatalf("below %v: %d menus shown, want one more than %d", below, len(popups), before)
		}
		items := popups[len(popups)-1].Menu.Items
		if len(items) != 2 || items[1].Label != belowQAT || items[1].Type != platform.MenuItemCheckbox || items[1].Checked != below {
			t.Fatalf("below %v: items %+v, want %q checked %v", below, items, belowQAT, below)
		}
	}
}

// Own case: a hidden QAT button takes no room and no tip; the next keeps its
// digit.
func TestRibbonQATHiddenButton(t *testing.T) {
	withMac(t, false)
	k := newQATKit(t)
	k.r.AddQuickAccess(k.undo, k.redo)
	k.undo.SetVisible(false)
	tt := k.tester(800)
	if n := countNodes(tt, ui.RoleButton, "Undo"); n != 0 {
		t.Fatalf("%d buttons Undo, hidden", n)
	}
	typeTip(tt, ui.KeyF10)
	want := maps.Clone(tabTips)
	want["Redo"] = "2"
	wantTips(t, tt, want)
	typeTip(tt, ui.Key1) // no tip: it ends the tips, and the hidden Undo does not run
	Update(func() {})
	if len(k.undone) != 0 {
		t.Fatal("1 ran the hidden Undo")
	}
	wantTips(t, tt, map[string]string{})
}

// Own case: "Add…" on a command already in the QAT is disabled; "Show below"
// moves the QAT below the ribbon, checked, and runs no OnQuickAccessChange.
func TestRibbonQATShowBelowByMenu(t *testing.T) {
	k := newQATKit(t)
	k.r.AddQuickAccess(k.paste)
	tt := k.tester(800)
	pastes := allBounds(tt, ui.RoleButton, "Paste")
	if len(pastes) != 2 {
		t.Fatalf("%d buttons Paste, want 2", len(pastes))
	}
	tt.RightClickAt(centre(pastes[1])) // the group's, after the QAT's
	tt.Frame()
	if m := tt.Menu(); !slices.Equal(m, []string{addQAT}) {
		t.Fatalf("Paste's menu %q, want %q", m, addQAT)
	}
	if err := tt.ChooseMenuItem(addQAT); err == nil {
		t.Fatal("Add on Paste, already in the QAT, was not disabled")
	}
	tt.CloseMenu()
	tt.RightClickAt(centre(qatButton(t, tt, "Paste")))
	tt.Frame()
	if err := tt.ChooseMenuItem(belowQAT); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	clip := nodeOf(t, tt, ui.RoleGroup, "Clipboard").Bounds
	pastes = allBounds(tt, ui.RoleButton, "Paste")
	if len(pastes) != 2 || pastes[1].Y < clip.Y+clip.H-0.5 {
		t.Fatalf("after Show below, Paste at %v, the groups end at %v", pastes, clip.Y+clip.H)
	}
	// Chosen again, it goes back above.
	tt.RightClickAt(centre(pastes[1]))
	tt.Frame()
	if err := tt.ChooseMenuItem(belowQAT); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	qatButton(t, tt, "Paste")
	noChange(t, k)
}

// R3. SetQuickAccess("Paste", "Nope", "Copy") shows Paste then Copy;
// QuickAccess() returns ["Paste" "Copy"]; the setter runs no callback. A
// button only in the QAT is found by name too.
func TestRibbonQATNames(t *testing.T) {
	k := newQATKit(t)
	k.r.SetQuickAccess("Paste", "Nope", "Copy")
	tt := k.tester(800)
	paste, cp := qatButton(t, tt, "Paste"), qatButton(t, tt, "Copy")
	if cp.X < paste.X+paste.W-0.5 || !near1(paste.Y, cp.Y) {
		t.Fatalf("Paste %v, Copy %v: not Paste then Copy in one row", paste, cp)
	}
	if got := k.r.QuickAccess(); !slices.Equal(got, []string{"Paste", "Copy"}) {
		t.Fatalf("QuickAccess() = %q, want [Paste Copy]", got)
	}
	k.r.AddQuickAccess(k.undo)
	k.r.SetQuickAccess("Undo", "Copy", "Copy")
	if got := k.r.QuickAccess(); !slices.Equal(got, []string{"Undo", "Copy"}) {
		t.Fatalf("QuickAccess() = %q, want [Undo Copy]", got)
	}
	k.r.SetQuickAccess()
	tt.Frame()
	if got := k.r.QuickAccess(); len(got) != 0 || countNodes(tt, ui.RoleButton, "Copy") != 1 {
		t.Fatalf("SetQuickAccess(): QuickAccess() = %q, %d buttons Copy", got, countNodes(tt, ui.RoleButton, "Copy"))
	}
	noChange(t, k)
}

// Own case: AddQuickAccess skips nil and a button the QAT holds.
func TestRibbonQATAddsOnce(t *testing.T) {
	k := newQATKit(t)
	k.r.AddQuickAccess(nil, k.undo, k.undo).AddQuickAccess(k.undo)
	if got := k.r.QuickAccess(); !slices.Equal(got, []string{"Undo"}) {
		t.Fatalf("QuickAccess() = %q, want [Undo]", got)
	}
	tt := k.tester(800)
	if n := countNodes(tt, ui.RoleButton, "Undo"); n != 1 {
		t.Fatalf("%d buttons Undo", n)
	}
}

// Own case: a toggle in the QAT and in its group: one click checks it and
// runs OnChange once.
func TestRibbonQATToggleRunsOnce(t *testing.T) {
	k := newQATKit(t)
	k.r.AddQuickAccess(k.bold)
	tt := k.tester(800)
	tt.ClickAt(centre(qatButton(t, tt, "Bold")))
	tt.Frame()
	received(t, k.bolded)
	nothingRan(t, k.ribbonKit)
	if !k.bold.Checked() {
		t.Fatal("the QAT's Bold did not check it")
	}
}

// R4. Alt shows 1 and 2 on the QAT, beside the tabs' letters, and pressing 2
// runs the second button alone.
func TestRibbonQATKeyTips(t *testing.T) {
	withMac(t, false)
	k := newQATKit(t)
	k.r.AddQuickAccess(k.undo, k.redo)
	tt := k.tester(800)
	typeTip(tt, ui.KeyMenuBar)
	want := maps.Clone(tabTips)
	want["Undo"], want["Redo"] = "1", "2"
	wantTips(t, tt, want)
	typeTip(tt, ui.Key2)
	received(t, k.redone)
	Update(func() {})
	if len(k.undone) != 0 || len(k.redone) != 0 {
		t.Fatalf("Undo ran %d times, Redo %d more", len(k.undone), len(k.redone))
	}
	wantTips(t, tt, map[string]string{})
}

// Own case: the first nine QAT buttons have the tips 1 to 9, the tenth none,
// and the tabs then take no digit.
func TestRibbonQATTipsStopAtNine(t *testing.T) {
	withMac(t, false)
	k := newQATKit(t)
	for _, name := range []string{"Q1", "Q2", "Q3", "Q4", "Q5", "Q6", "Q7", "Q8", "Q9", "Q10"} {
		k.r.AddQuickAccess(ToolButton(name).SetIcon(chevronDown))
	}
	tt := k.tester(1200)
	typeTip(tt, ui.KeyF10)
	got := tipsRead(tt)
	want := maps.Clone(tabTips)
	for i, d := range "123456789" {
		want["Q"+string(rune('1'+i))] = string(d)
	}
	if !maps.Equal(got, want) {
		t.Fatalf("tips %v, want %v", got, want)
	}
}

// Own case: a tab titled with a digit does not take a QAT's digit.
func TestRibbonQATDigitsAreReserved(t *testing.T) {
	withMac(t, false)
	k := newQATKit(t)
	k.r.AddTab("1A") // with H and I taken by Home and Insert, it would take 1
	k.r.AddQuickAccess(k.undo)
	tt := k.tester(800)
	typeTip(tt, ui.KeyF10)
	if got := tipsRead(tt); got["Undo"] != "1" || got["1A"] != "A" {
		t.Fatalf("tips %v: want Undo 1 and 1A A", got)
	}
}

// contextKit is the kit with a contextual tab, Format of Picture Tools,
// hidden.
func contextKit(t *testing.T) *qatKit {
	t.Helper()
	k := newQATKit(t)
	k.r.AddContextualTab("Picture Tools", "Format").AddGroup("Arrange").AddSmall(ToolButton("Rotate"))
	return k
}

// R5. A category's tab is hidden until SetCategoryVisible; its band shows the
// category's name above the tab, in the accent series; hiding the category
// while its tab is shown shows tab 0.
func TestRibbonContextualTabs(t *testing.T) {
	k := contextKit(t)
	tt := k.tester(800)
	if countNodes(tt, ui.RoleTab, "Format") != 0 || tt.HasText("Picture Tools") {
		t.Fatalf("the hidden category shows: %q", tt.Texts())
	}
	k.r.SetCategoryVisible("Picture Tools", true)
	tt.Frame()
	tab := nodeOf(t, tt, ui.RoleTab, "Format").Bounds
	band := nodeOf(t, tt, ui.RoleText, "Picture Tools").Bounds
	if band.Y+band.H > tab.Y+0.5 || band.X > tab.X+tab.W || band.X+band.W < tab.X {
		t.Fatalf("the band %v is not above the tab %v", band, tab)
	}
	home := nodeOf(t, tt, ui.RoleTab, "Home").Bounds
	if !near1(home.Y+home.H, tab.Y+tab.H) {
		t.Fatalf("Home ends at %v, Format at %v: the tabs are not on one line", home.Y+home.H, tab.Y+tab.H)
	}
	if c := tt.Image().RGBAAt(int(tab.X+1), int(band.Y+1)); !near(c, plainChartColors[0]) {
		t.Fatalf("the band is %v, want the first series color %v", c, plainChartColors[0])
	}
	click(t, tt, "Format")
	if k.r.Tab() != 2 || !tt.HasText("Rotate") {
		t.Fatalf("a click on Format: Tab %d, texts %q", k.r.Tab(), tt.Texts())
	}
	k.r.SetCategoryVisible("Picture Tools", false)
	tt.Frame()
	if k.r.Tab() != 0 || !tt.HasText("Clipboard") || countNodes(tt, ui.RoleTab, "Format") != 0 {
		t.Fatalf("hiding the category shown: Tab %d, texts %q", k.r.Tab(), tt.Texts())
	}
	// Hiding a category whose tab is not shown keeps the tab shown.
	k.r.SetCategoryVisible("Picture Tools", true).SetTab(1).SetCategoryVisible("Picture Tools", false)
	if k.r.Tab() != 1 {
		t.Fatalf("hiding a category not shown moved the tab to %d", k.r.Tab())
	}
}

// Round 2: SetTab to a contextual tab whose category is hidden is ignored:
// the tab shown stays, and none of the hidden tab's groups show; once the
// category shows, SetTab shows it.
func TestRibbonContextualSetTabHidden(t *testing.T) {
	k := contextKit(t)
	tt := k.tester(800)
	k.r.SetTab(1)
	tt.Frame()
	k.r.SetTab(2)
	tt.Frame()
	if k.r.Tab() != 1 || tt.HasText("Arrange") || tt.HasText("Rotate") || !tt.HasText("Tables") {
		t.Fatalf("SetTab(2), Format hidden: Tab %d, texts %q", k.r.Tab(), tt.Texts())
	}
	k.r.SetCategoryVisible("Picture Tools", true).SetTab(2)
	tt.Frame()
	if k.r.Tab() != 2 || !tt.HasText("Rotate") {
		t.Fatalf("SetTab(2), Format shown: Tab %d, texts %q", k.r.Tab(), tt.Texts())
	}
}

// Own case: two tabs of one category sit under one band spanning both; a
// contextual tab of no category shows always, with no band.
func TestRibbonContextualOneBandForACategorysTabs(t *testing.T) {
	k := contextKit(t)
	k.r.AddContextualTab("Picture Tools", "Crop").AddGroup("Size").AddSmall(ToolButton("Trim"))
	k.r.AddContextualTab("", "Plain").AddGroup("P").AddSmall(ToolButton("Pp"))
	tt := k.tester(800)
	if countNodes(tt, ui.RoleTab, "Plain") != 1 {
		t.Fatalf("the tab of no category does not show: %q", tt.Texts())
	}
	k.r.SetCategoryVisible("Picture Tools", true)
	tt.Frame()
	bands := allBounds(tt, ui.RoleText, "Picture Tools")
	format, crop := nodeOf(t, tt, ui.RoleTab, "Format").Bounds, nodeOf(t, tt, ui.RoleTab, "Crop").Bounds
	if len(bands) != 1 {
		t.Fatalf("%d bands Picture Tools, want 1", len(bands))
	}
	if b := bands[0]; b.X > format.X+0.5 || b.X+b.W < crop.X+crop.W-0.5 {
		t.Fatalf("the band %v does not span Format %v and Crop %v", b, format, crop)
	}
}

// Own case: under each built-in skin, light and dark, the band's name reads
// 4.5:1 on each series color.
func TestRibbonContextualBandReads(t *testing.T) {
	check := func(t *testing.T, colors []ui.Color, th ui.Theme) {
		t.Helper()
		for i, fill := range colors {
			ink := th.Text
			if contrastOf(th.Background, fill) > contrastOf(ink, fill) {
				ink = th.Background
			}
			if r := ratio(ink, fill); r < 4.5 {
				t.Errorf("Chart[%d] %v: the name reads %.2f:1", i, fill, r)
			}
		}
	}
	t.Run("none", func(t *testing.T) { check(t, plainChartColors, *ui.LightTheme()) })
	for _, name := range []string{"Calm", "Classic", "HighContrast"} {
		for _, dark := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "-light", true: "-dark"}[dark], func(t *testing.T) {
				useSkinNamed(t, name, false)
				k := contextKit(t)
				k.r.SetCategoryVisible("Picture Tools", true)
				tt := ui.NewTester(k.form.view, 800, 400)
				tt.SetDark(dark)
				tt.Frame()
				tk := skinhook.Get(dark)
				band := nodeOf(t, tt, ui.RoleText, "Picture Tools").Bounds
				if c := tt.Image().RGBAAt(int(band.X+1), int(band.Y+1)); !near(c, tk.Chart[0]) {
					t.Fatalf("the band is %v, want Chart[0] %v", c, tk.Chart[0])
				}
				check(t, tk.Chart[:], ui.Theme{Text: tk.Text, Background: tk.Background}) // the skin's theme's
				// The name is drawn in the ink that reads best.
				ink := tk.Text
				if contrastOf(tk.Background, tk.Chart[0]) > contrastOf(ink, tk.Chart[0]) {
					ink = tk.Background
				}
				found := false
				for y := int(band.Y) + 2; y < int(band.Y+band.H)-2 && !found; y++ { // inside the fill
					for x := int(band.X) + 2; x < int(band.X+band.W)-2; x++ {
						if nearTol(tt.Image().RGBAAt(x, y), ink, 24) {
							found = true
							break
						}
					}
				}
				if !found {
					t.Fatalf("no pixel of the ink %v in the band %v", ink, band)
				}
			})
		}
	}
}

// Own case: the arrows go onto a visible contextual tab, in its band, and
// back.
func TestRibbonContextualTabTakesTheArrows(t *testing.T) {
	k := contextKit(t)
	k.r.SetCategoryVisible("Picture Tools", true).SetTab(1).Focus()
	tt := k.tester(800)
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if k.r.Tab() != 2 || !tt.Focused("Format") {
		t.Fatalf("Right from Insert: Tab %d, Format focused %v", k.r.Tab(), tt.Focused("Format"))
	}
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if k.r.Tab() != 1 {
		t.Fatalf("Left from Format: Tab %d, want 1", k.r.Tab())
	}
}

// Own case: under a skin, the bands take its Chart colors by category order.
func TestRibbonContextualBandsTakeTheSkinsSeries(t *testing.T) {
	useSkinNamed(t, "Calm", false)
	tk := skinhook.Get(false)
	k := contextKit(t)
	k.r.AddContextualTab("Table Tools", "Layout").AddGroup("Rows").AddSmall(ToolButton("Insert row"))
	k.r.SetCategoryVisible("Picture Tools", true).SetCategoryVisible("Table Tools", true)
	tt := k.tester(800)
	for i, cat := range []string{"Picture Tools", "Table Tools"} {
		band := nodeOf(t, tt, ui.RoleText, cat).Bounds
		if c := tt.Image().RGBAAt(int(band.X+1), int(band.Y+1)); !near(c, tk.Chart[i]) {
			t.Fatalf("the band of %s is %v, want Chart[%d] %v", cat, c, i, tk.Chart[i])
		}
	}
}

// Own case: the arrows skip a hidden contextual tab.
func TestRibbonContextualHiddenTabTakesNoArrow(t *testing.T) {
	k := contextKit(t)
	k.r.AddTab("View")
	tt := k.tester(800)
	k.r.SetTab(1)
	k.r.Focus()
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if k.r.Tab() != 3 {
		t.Fatalf("Right from Insert went to tab %d, want 3 (View), past the hidden Format", k.r.Tab())
	}
}

// R6. A visible contextual tab has a key tip; a hidden one has none.
func TestRibbonContextualKeyTips(t *testing.T) {
	withMac(t, false)
	k := contextKit(t)
	tt := k.tester(800)
	typeTip(tt, ui.KeyF10)
	wantTips(t, tt, tabTips)
	typeTip(tt, ui.KeyEscape)
	k.r.SetCategoryVisible("Picture Tools", true)
	tt.Frame()
	typeTip(tt, ui.KeyF10)
	want := maps.Clone(tabTips)
	want["Format"] = "F"
	wantTips(t, tt, want)
	typeTip(tt, ui.KeyF)
	if k.r.Tab() != 2 {
		t.Fatalf("F shows tab %d, want 2", k.r.Tab())
	}
}

// headerKit is the kit in a form with a header bar.
func headerKit(t *testing.T) *qatKit {
	t.Helper()
	k := newQATKit(t)
	k.form.UseHeaderBar("")
	k.r.AddQuickAccess(k.undo, k.redo)
	return k
}

// R7. With a HeaderBar, the QAT draws in the title bar's start area, after
// one frame, and nowhere else; dragging the empty bar still drags the window,
// and a press on a QAT button does not.
func TestRibbonQATInHeaderBar(t *testing.T) {
	withMac(t, false)
	k := headerKit(t)
	tt := ui.NewTester(k.form.view, 800, 400)
	tt.SetTitleBar(windowsBar)
	tt.Frame()
	undos := allBounds(tt, ui.RoleButton, "Undo")
	title := nodeOf(t, tt, ui.RoleText, "Ribbon").Bounds
	if len(undos) != 1 {
		t.Fatalf("%d buttons Undo, want 1, in the title bar", len(undos))
	}
	if tt.FramePending() {
		t.Fatal("the form still asks for frames at rest")
	}
	undo, redo := undos[0], rnode(t, tt, "Redo")
	if cy := undo.Y + undo.H/2; cy < title.Y || cy > title.Y+title.H || undo.X+undo.W > title.X || redo.X < undo.X {
		t.Fatalf("Undo %v, Redo %v: not in the title bar's start area, before the title %v", undo, redo, title)
	}
	// The key tips reach it there.
	typeTip(tt, ui.KeyF10)
	want := maps.Clone(tabTips)
	want["Undo"], want["Redo"] = "1", "2"
	wantTips(t, tt, want)
	typeTip(tt, ui.KeyEscape)

	// A shown form: a press on the empty bar drags the window.
	f := NewForm("QAT drag", 800, 400)
	f.UseHeaderBar("")
	r := Ribbon().AddTab("Home").AddGroup("G").AddLarge(ToolButton("Big").SetIcon(chevronDown)).
		AddQuickAccess(ToolButton("Save").SetIcon(chevronDown))
	f.SetContent(VBox(0).Add(r, Label("body")))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "QAT drag")
	settleFake(fw)
	var save, bar ui.Rect
	for _, n := range accessNodes(fw) {
		b := ui.Rect{X: float32(n.Bounds.X), Y: float32(n.Bounds.Y), W: float32(n.Bounds.W), H: float32(n.Bounds.H)}
		switch n.Label {
		case "Save":
			save = b
		case "QAT drag":
			bar = b
		}
	}
	if save.W == 0 || bar.W == 0 {
		t.Fatalf("Save %v, the title %v: not both in the window", save, bar)
	}
	pressFake(fw, save.X+save.W/2, save.Y+save.H/2)
	if n := fw.Drags(); n != 0 {
		t.Fatalf("a press on Save dragged the window %d times", n)
	}
	pressFake(fw, (save.X+save.W+bar.X)/2, save.Y+save.H/2)
	if n := fw.Drags(); n != 1 {
		t.Fatalf("a press on the empty bar between Save and the title dragged the window %d times, want 1", n)
	}
}

// settleFake draws the frames fw's view asks for.
func settleFake(fw *fake.Window) {
	mygo.RunOnMain(func() {
		for range 10 {
			if !fw.FakeSurface().Frame() {
				return
			}
		}
	})
}

// pressFake presses and releases the first button at x, y in fw's view.
func pressFake(fw *fake.Window, x, y float32) {
	mygo.RunOnMain(func() {
		s := fw.FakeSurface()
		s.Send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y)})
		s.Send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y)})
		s.Frame()
	})
}

// Own case: of two ribbons with a QAT in a form with a header bar, the first
// takes the title bar and the second draws its QAT above its own tabs; the
// form rests.
func TestRibbonQATTwoRibbonsShareNoHeader(t *testing.T) {
	r1 := Ribbon().AddTab("One").AddGroup("G").AddSmall(ToolButton("A")).AddQuickAccess(ToolButton("Save").SetIcon(chevronDown))
	r2 := Ribbon().AddTab("Two").AddGroup("G").AddSmall(ToolButton("B")).AddQuickAccess(ToolButton("Print").SetIcon(chevronDown))
	f := NewForm("Two", 800, 500).SetContent(VBox(0).Add(r1, r2))
	f.UseHeaderBar("")
	tt := ui.NewTester(f.view, 800, 500)
	tt.SetTitleBar(windowsBar)
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("the form still asks for frames at rest")
	}
	title, save, prt := nodeOf(t, tt, ui.RoleText, "Two").Bounds, rnode(t, tt, "Save"), rnode(t, tt, "Print")
	if cy := save.Y + save.H/2; cy < title.Y || cy > title.Y+title.H {
		t.Fatalf("Save %v is not in the title bar (the title %v)", save, title)
	}
	two := nodeOf(t, tt, ui.RoleTab, "Two").Bounds
	if one := nodeOf(t, tt, ui.RoleTab, "One").Bounds; prt.Y < one.Y+one.H || prt.Y+prt.H > two.Y+0.5 {
		t.Fatalf("Print %v is not above its ribbon's tab %v, under the first ribbon's %v", prt, two, one)
	}
}

// Own case: the QAT leaves the title bar with the ribbon, in the frames that
// follow the change, when the ribbon's page is no longer shown.
func TestRibbonQATLeavesTheHeaderWithTheRibbon(t *testing.T) {
	r := Ribbon().AddTab("Home").AddGroup("G").AddSmall(ToolButton("A")).
		AddQuickAccess(ToolButton("Save").SetIcon(chevronDown))
	pages := TabControl().AddPage("Write", r).AddPage("Other", Label("other"))
	f := NewForm("Pages", 800, 400).SetContent(pages)
	f.UseHeaderBar("")
	tt := ui.NewTester(f.view, 800, 400)
	tt.Frame()
	if n := countNodes(tt, ui.RoleButton, "Save"); n != 1 {
		t.Fatalf("%d buttons Save with the ribbon shown, want 1", n)
	}
	pages.SetPage(1)
	tt.Frame()
	if n := countNodes(tt, ui.RoleButton, "Save"); n != 0 || !tt.HasText("other") {
		t.Fatalf("%d buttons Save with the other page shown, want 0 (texts %q)", n, tt.Texts())
	}
	pages.SetPage(0)
	tt.Frame()
	if n := countNodes(tt, ui.RoleButton, "Save"); n != 1 {
		t.Fatalf("%d buttons Save with the ribbon shown again, want 1", n)
	}
}

// R8. Right to left, the QAT starts at the right, Undo right of Redo, and the
// band stays above its tab.
func TestRibbonQATRTL(t *testing.T) {
	k := contextKit(t)
	k.r.AddQuickAccess(k.undo, k.redo).SetCategoryVisible("Picture Tools", true)
	tt := ui.NewTester(k.form.view, 800, 400)
	tt.SetDirection(ui.RightToLeft)
	tt.Frame()
	undo, redo := qatButton(t, tt, "Undo"), qatButton(t, tt, "Redo")
	home := nodeOf(t, tt, ui.RoleTab, "Home").Bounds
	if undo.X < redo.X+redo.W-0.5 || undo.X+undo.W < home.X {
		t.Fatalf("right to left: Undo %v, Redo %v, Home %v: Undo is not first from the right", undo, redo, home)
	}
	tab, band := nodeOf(t, tt, ui.RoleTab, "Format").Bounds, nodeOf(t, tt, ui.RoleText, "Picture Tools").Bounds
	if band.Y+band.H > tab.Y+0.5 || tab.X > home.X {
		t.Fatalf("right to left: band %v, Format %v, Home %v", band, tab, home)
	}
}

// R9. Off is unchanged: a ribbon that calls no QAT method offers no menu on
// its commands, and a hidden category and an empty QAT move nothing.
func TestRibbonQATOffIsUnchanged(t *testing.T) {
	plain := newRibbonKit(t)
	if got := plain.r.QuickAccess(); got != nil {
		t.Fatalf("QuickAccess() = %q on a ribbon with no QAT, want nil", got)
	}
	tt := plain.tester(800)
	if err := tt.RightClick("Paste"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m := tt.Menu(); m != nil {
		t.Fatalf("a ribbon with no QAT gives Paste the menu %q", m)
	}
	tt.CloseMenu()
	tt.Frame()
	want := layoutOf(tt)
	for name, k := range map[string]*qatKit{"hidden category": contextKit(t), "empty QAT": newQATKit(t)} {
		if name == "empty QAT" {
			k.r.AddQuickAccess()
		}
		got := k.tester(800)
		if l := layoutOf(got); !slices.Equal(l, want) {
			t.Fatalf("%s: the layout\n%q\nwant\n%q", name, l, want)
		}
	}
}

// R10. Every method from another goroutine while frames run shows on a
// following frame (run it under -race).
func TestRibbonQATFromGoroutine(t *testing.T) {
	k := newQATKit(t)
	tt := k.tester(800)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		k.r.AddQuickAccess(k.undo)
		k.r.SetQuickAccess("Undo", "Paste")
		k.r.SetQuickAccessBelow(true)
		k.r.SetQuickAccessBelow(false)
		_ = k.r.QuickAccess()
		k.r.AddContextualTab("Picture Tools", "Format").AddGroup("Arrange").AddSmall(ToolButton("Rotate"))
		k.r.SetCategoryVisible("Picture Tools", true)
		k.r.OnQuickAccessChange(func() {})
	}()
	for range 20 {
		tt.Frame()
	}
	wg.Wait()
	tt.Frame()
	qatButton(t, tt, "Undo")
	qatButton(t, tt, "Paste")
	if countNodes(tt, ui.RoleTab, "Format") != 1 || !tt.HasText("Picture Tools") {
		t.Fatalf("the contextual tab did not show: %q", tt.Texts())
	}
}

// R11. A frame at rest allocates no more with the QAT and a contextual tab
// than the same controls without them: the buttons in a row above a plain
// ribbon with one more tab, and a label above it.
func TestRibbonQATAllocNothingAtRest(t *testing.T) {
	k := contextKit(t)
	k.r.AddQuickAccess(k.undo, k.redo).SetCategoryVisible("Picture Tools", true)
	got := ui.NewTester(content(VBox(0).Add(k.r, k.body)), 800, 400)
	q := newQATKit(t)
	q.r.AddTab("Format").AddGroup("Arrange").AddSmall(ToolButton("Rotate"))
	ref := ui.NewTester(content(VBox(0).Add(HBox(0).Add(q.undo, q.redo), Label("Picture Tools"), q.r, q.body)), 800, 400)
	empty := newQATKit(t)
	empty.r.AddQuickAccess()
	gotEmpty := ui.NewTester(content(VBox(0).Add(empty.r, empty.body)), 800, 400)
	plain := newRibbonKit(t)
	refPlain := ui.NewTester(content(VBox(0).Add(plain.r, plain.body)), 800, 400)
	for range 10 {
		got.Frame()
		ref.Frame()
		gotEmpty.Frame()
		refPlain.Frame()
	}
	a, b := allocrun.Least(30, got.Frame), allocrun.Least(30, ref.Frame)
	t.Logf("allocations in a frame: QAT and a contextual tab %v, the same in boxes %v", a, b)
	if a > b {
		t.Fatalf("a frame with the QAT and a contextual tab allocates %v, without them %v", a, b)
	}
	a, b = allocrun.Least(30, gotEmpty.Frame), allocrun.Least(30, refPlain.Frame)
	t.Logf("allocations in a frame: an empty QAT %v, a plain ribbon %v", a, b)
	if a > b {
		t.Fatalf("a frame with an empty QAT allocates %v, a plain ribbon %v", a, b)
	}
}
