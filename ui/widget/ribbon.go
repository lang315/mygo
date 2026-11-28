package widget

import (
	"slices"

	"github.com/egoist/mygo/internal/keyhold"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// RibbonWidget is a command bar of tabs, each holding groups of large and
// small commands, as Office's and DevExpress's ribbon; put it first in a
// form's content.
type RibbonWidget struct {
	common[RibbonWidget]
	tabs []*ribbonTab
	// keys are tabs as ui takes them, a new slice when the tabs change: ui
	// keeps the last frame's.
	keys      []any
	tab       mirror[int]
	collapsed bool
	// expanded is set by a click on a tab that expanded the ribbon, so that
	// the double click it starts does not collapse it again; swallow is
	// set by a double click, whose release clicks the tab once more and
	// must not expand what it collapsed. Under mu, while building.
	expanded, swallow bool
	// changed and toggle are ui's callbacks, made once: a closure made in
	// each build is an allocation in each frame.
	changed, toggle func()
	// tips is the level of the key tips: tipsOff, tipsTabs or tipsCommands.
	// tipBack is the control that had the focus before they showed, to give
	// it back; giveBack is it once they ended, until held, the letter that
	// ended them (0 for another key), is let go: the text the letter types
	// comes before, and must not reach a text box getting the focus back.
	// tipKeys answers F10 and Alt alone at the end of the form's build
	// (keyTipsHook), and tipPaint paints the badges over the ribbon: both
	// made once.
	tips     int8
	tipBack  Control
	giveBack Control
	held     ui.Key
	tipKeys  func(c *ui.Context)
	tipPaint func(p *ui.Painter, r ui.Rect)
	// shown are the badges of the last build, a slice reused from build to
	// build, and col the ribbon's element, which tipPaint reads, on the main
	// thread, as the frame paints.
	shown []ribbonTip
	col   ui.Element
	// tipW is the width of each letter at tipSize, tipH their height,
	// measured once; tipFill and tipInk the badge's colors.
	tipW            [len(tipLetters)]float32
	tipH, tipSize   float32
	tipFill, tipInk ui.Color
	// sb is the strip being built, which stripTab reads: here, not in a
	// closure, so that the tab loop allocates nothing.
	sb ribbonStrip

	// The Quick Access Toolbar and the contextual tabs (ribbon_qat.go), nil
	// until a method of theirs installs them, so that an app that calls none
	// does not link them. qatHook builds the toolbar where it goes, before
	// the tabs (below false) and after the groups (below true); cmdMenu
	// gives a command that showed the toolbar's context menu; bands builds
	// the tabs with their categories' bands, in place of the plain loop.
	// qatTaken are the tip slots of the toolbar's digits, which the tabs do
	// not take.
	qat      *ribbonQAT
	cats     *ribbonCats
	qatHook  func(c *ui.Context, below bool)
	cmdMenu  func(w Control)
	bands    func(c *ui.Context)
	qatTaken uint64

	// galleryTip opens the drop-down of a command's tip when the command is
	// an enabled gallery (ribbon_gallery.go), and reports whether it did:
	// nil until AddGallery installs it.
	galleryTip func(tp ribbonTip) bool
}

// ribbonStrip is the strip being built: ui's tabs, the tab shown, the
// skin's TabHeader look, the tips taken, and what the pointer did to a tab.
type ribbonStrip struct {
	tabs                     ui.TabsParts
	p                        *int
	look                     *Style
	taken                    uint64
	clicked, double, pressed bool
}

// The levels of the key tips: none, a letter on each tab, a letter on each
// command of the tab shown.
const (
	tipsOff = iota
	tipsTabs
	tipsCommands
)

// tipLetters are the key tips, by slot: A to Z, then 1 to 9.
const tipLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"

// ribbonTip is a badge: the element it is over, its slot in tipLetters, and
// the command (nil on a tab) or the tab (-1 on a command) its letter acts on.
type ribbonTip struct {
	el   ui.Element
	slot int8
	w    Control
	tab  int
}

// keyTipsHook is, while a form builds, the F10 and Alt alone of its first
// ribbon, which the form calls at the end of its build when it has no menu
// bar to give those keys to (menuBarMode). Under mu.
var keyTipsHook func(c *ui.Context)

// ribbonTab is a tab of a ribbon, and ribbonGroup a group of a tab. A tab
// is its own key in ui's tabs: a pointer to a value that is not empty. act
// shows its commands' tips, for its access key, made once. category is a
// contextual tab's (AddContextualTab), "" for a tab always shown.
type ribbonTab struct {
	title    string
	groups   []*ribbonGroup
	act      func()
	category string
}

type ribbonGroup struct {
	title string
	items []ribbonItem
}

type ribbonItem struct {
	w     Control
	large bool
}

// Ribbon creates a ribbon with no tab.
func Ribbon() *RibbonWidget {
	r := &RibbonWidget{}
	r.init(r)
	r.changed = onChange(&r.tab, nil)
	r.toggle = func() { r.change(func() { r.collapsed = !r.collapsed }) }
	r.tipKeys = func(c *ui.Context) {
		pressed := c.Shortcut(0, ui.KeyF10)
		if c.Shortcut(0, ui.KeyMenuBar) {
			pressed = true
		}
		switch {
		case !pressed:
		case r.tips != tipsOff:
			r.leaveTips(true)
		default:
			r.enterTips()
		}
	}
	r.tipPaint = r.paintTips
	keyhold.Enable() // the keys typed after Alt alone or F10 reach the tips
	return r
}

// AddTab adds a tab titled title after the others.
func (r *RibbonWidget) AddTab(title string) *RibbonWidget {
	r.change(func() { r.addTab(title) })
	return r
}

// addTab adds a tab. Under mu.
func (r *RibbonWidget) addTab(title string) {
	t := &ribbonTab{title: title}
	t.act = func() {
		if i := slices.Index(r.tabs, t); i >= 0 {
			r.enterTips()
			r.showTab(i)
		}
	}
	r.tabs = append(r.tabs, t)
	r.keys = append(slices.Clip(r.keys), t)
}

// AddGroup adds a group titled title to the last tab (to a new tab with
// no title before any).
func (r *RibbonWidget) AddGroup(title string) *RibbonWidget {
	r.change(func() { r.addGroup(title) })
	return r
}

// addGroup adds a group to the last tab. Under mu.
func (r *RibbonWidget) addGroup(title string) {
	if len(r.tabs) == 0 {
		r.addTab("")
	}
	t := r.tabs[len(r.tabs)-1]
	t.groups = append(t.groups, &ribbonGroup{title: title})
}

// AddLarge adds buttons to the last group (a new one with no title before
// any), each drawn large: its icon above its text.
func (r *RibbonWidget) AddLarge(buttons ...*ToolButtonWidget) *RibbonWidget {
	r.change(func() {
		for _, b := range buttons {
			if b != nil {
				r.add(b, true)
			}
		}
	})
	return r
}

// AddSmall adds controls to the last group, stacked three to a column: tool
// buttons, drop-down buttons, combo boxes, check boxes.
func (r *RibbonWidget) AddSmall(controls ...Control) *RibbonWidget {
	r.change(func() {
		for _, c := range controls {
			if c != nil {
				r.add(c, false)
			}
		}
	})
	return r
}

// add adds w to the last group, unless the ribbon holds it already, as
// Fields.Row refuses a control the form holds. Under mu.
func (r *RibbonWidget) add(w Control, large bool) {
	for _, t := range r.tabs {
		for _, g := range t.groups {
			for _, it := range g.items {
				if it.w == w {
					return
				}
			}
		}
	}
	t := len(r.tabs) - 1
	if t < 0 || len(r.tabs[t].groups) == 0 {
		r.addGroup("")
		t = len(r.tabs) - 1
	}
	g := r.tabs[t].groups[len(r.tabs[t].groups)-1]
	g.items = append(g.items, ribbonItem{w, large})
	adopt(&r.core, w)
}

// Tab returns the index of the tab shown; SetTab shows tab i.
func (r *RibbonWidget) Tab() int { mu.Lock(); defer mu.Unlock(); return r.tab.value }

func (r *RibbonWidget) SetTab(i int) *RibbonWidget {
	r.change(func() {
		if r.cats == nil || !r.cats.hides(r.tabs, i) { // a hidden contextual tab is not shown
			r.tab.set(i)
		}
	})
	return r
}

// SetCollapsed shows the tabs alone, or the tabs and their groups; a double
// click on a tab and Ctrl+F1 toggle it. Collapsed reports it.
func (r *RibbonWidget) SetCollapsed(on bool) *RibbonWidget {
	r.change(func() { r.collapsed = on })
	return r
}

func (r *RibbonWidget) Collapsed() bool { mu.Lock(); defer mu.Unlock(); return r.collapsed }

func (r *RibbonWidget) build(c *ui.Context) ui.Element {
	col := ui.Column(c).AlignSelf(ui.Stretch)
	r.shown = r.shown[:0]
	if r.giveBack != nil && (r.held == 0 || !c.KeyDown(r.held)) {
		focusLater(r.giveBack)
		r.giveBack, r.held = nil, 0
	}
	if len(r.tabs) == 0 {
		r.tips = tipsOff
		return col
	}
	if buildingForm != nil && !macMenus && keyTipsHook == nil {
		keyTipsHook = r.tipKeys
	}
	col.Children(func() {
		if r.qatHook != nil {
			r.qatHook(c, false)
		}
		p, _ := r.tab.bind()
		strip := r.strip(c, p)
		r.inner = strip // Focus (an access key) goes to the tabs, as a TabControl's
		c.OnWindowShortcut(ui.Ctrl, ui.KeyF1, r.toggle)
		if i := *p; !r.collapsed && i >= 0 && i < len(r.tabs) {
			r.groups(c, r.tabs[i])
		}
		if r.qatHook != nil {
			r.qatHook(c, true)
		}
	})
	if r.tips != tipsOff {
		r.tipMode(c, col)
	}
	return col
}

// enterTips shows the tabs' tips, the shown tab taking the focus from the
// control that has it, which gets it back as they end. Under mu, at the
// end of a form's build.
func (r *RibbonWidget) enterTips() {
	if r.tips == tipsOff {
		r.tipBack = buildingForm.focusedControl()
	}
	r.tips, r.wantFocus = tipsTabs, true
}

// leaveTips ends the tips, giving the focus back to the control that had it
// when restore is set, in a following build (giveBack). Under mu.
func (r *RibbonWidget) leaveTips(restore bool) {
	if restore && r.tipBack != nil {
		r.giveBack = r.tipBack
	}
	r.tips, r.tipBack = tipsOff, nil
}

// showTab shows tab i, expanded, with its commands' tips. Under mu.
func (r *RibbonWidget) showTab(i int) {
	r.tab.set(i)
	r.collapsed, r.tips = false, tipsCommands
}

// tipMode answers the keys of the tips, which end with a press in the window
// or the window going inactive: Escape goes back a level, a tip's letter
// acts, and any other letter or digit ends them. Every key is asked for in
// every build, so that the next build has those typed meanwhile. Under mu,
// after the ribbon's children built.
func (r *RibbonWidget) tipMode(c *ui.Context, col ui.Element) {
	t := c.Theme()
	r.col, r.tipFill, r.tipInk = col, t.Text, t.Background
	col.ZIndex(1).DrawOver(r.tipPaint) // over what follows, as a tab's badge reaches below the ribbon
	if size := t.Rem(0.75); size != r.tipSize {
		r.tipSize, r.tipW = size, [len(tipLetters)]float32{}
	}
	for _, tp := range r.shown {
		if r.tipW[tp.slot] == 0 {
			r.tipW[tp.slot], r.tipH = c.MeasureText(0, ui.Span{Text: tipLetters[tp.slot : tp.slot+1], Size: r.tipSize})
		}
	}
	pressed := -1 // the slot typed, len(tipLetters) for 0
	for s := 0; s <= len(tipLetters); s++ {
		if c.Shortcut(0, tipKey(s)) && pressed < 0 {
			pressed = s
		}
	}
	escape := c.Shortcut(0, ui.KeyEscape)
	switch {
	case !c.WindowActive(), col.PressedInside(), col.PressedOutside():
		r.leaveTips(false) // the press puts the focus where it goes
	case escape && r.tips == tipsCommands:
		r.tips, r.wantFocus = tipsTabs, true
	case escape:
		r.leaveTips(true)
	case pressed >= 0:
		r.held = tipKey(pressed)
		i := slices.IndexFunc(r.shown, func(tp ribbonTip) bool { return int(tp.slot) == pressed })
		switch {
		case i < 0:
			r.leaveTips(true)
		case r.shown[i].w == nil: // a tab's; a command's, or the Quick Access Toolbar's beside the tabs', acts
			r.showTab(r.shown[i].tab)
		default:
			r.runTip(r.shown[i])
		}
	}
}

// runTip acts on tp's command as its access key would, and ends the tips:
// a tool button clicks, or toggles and runs OnChange; any other control,
// a drop-down button too, takes the focus; a gallery opens its drop-down
// (galleryTip). A disabled one does nothing. Under mu.
func (r *RibbonWidget) runTip(tp ribbonTip) {
	if r.galleryTip != nil && r.galleryTip(tp) {
		r.leaveTips(false) // the drop-down takes the focus
		return
	}
	b, tool := tp.w.(*ToolButtonWidget)
	if tp.el.IsDisabled() || tool {
		r.leaveTips(true)
	} else {
		r.leaveTips(false)
		focusLater(tp.w)
	}
	switch {
	case tp.el.IsDisabled() || !tool:
	case b.toggle:
		b.on.set(!b.on.value)
		if b.onChange != nil {
			Post(b.onChange)
		}
	case b.onClick != nil:
		Post(b.onClick)
	}
}

// tipKey is the key of slot s of tipLetters; s == len(tipLetters) is 0.
func tipKey(s int) ui.Key {
	switch {
	case s < 26:
		return ui.KeyA + ui.Key(s)
	case s < len(tipLetters):
		return ui.Key1 + ui.Key(s-26)
	}
	return ui.Key0
}

// keyTip gives name a tip among those taken, a set of slots of tipLetters:
// the first of its characters whose letter (foldKey) is free, else the first
// slot free. It reports -1 when all are taken.
func keyTip(name string, taken *uint64) int {
	for _, ch := range name {
		k := foldKey(ch)
		s := -1
		switch {
		case k >= 'a' && k <= 'z':
			s = int(k - 'a')
		case k >= '1' && k <= '9':
			s = int(k-'1') + 26
		}
		if s >= 0 && *taken&(1<<s) == 0 {
			*taken |= 1 << s
			return s
		}
	}
	for s := range len(tipLetters) {
		if *taken&(1<<s) == 0 {
			*taken |= 1 << s
			return s
		}
	}
	return -1
}

// tipName is what a command's tip is taken from: a tool button's text, or
// its tooltip with none; another control's accessible name, else its text.
// Under mu.
func tipName(w Control) string {
	switch b := w.(type) {
	case *ToolButtonWidget:
		if b.text != "" {
			return b.text
		}
		return b.tooltip
	case *MenuButtonWidget:
		if b.accName == "" {
			return b.text
		}
	case *CheckBoxWidget:
		if b.accName == "" {
			s, _, _ := parseMnemonic(b.text)
			return s
		}
	}
	return w.base().accName
}

// badge puts tip slot on element el, for tab or command w, and tells
// assistive technology its key. Under mu, while building.
func (r *RibbonWidget) badge(el ui.Element, slot int, w Control, tab int) {
	el.KeyShortcuts(0, tipKey(slot))
	r.shown = append(r.shown, ribbonTip{el, int8(slot), w, tab})
}

// paintTips paints the badges of the last build over the ribbon, whose box
// is box: each a box filled with the text's color, holding its letter in the
// background's, lined with the background's, centred on its element's
// bottom edge. It reads the elements'
// bounds as the frame paints them, on the main thread.
func (r *RibbonWidget) paintTips(p *ui.Painter, box ui.Rect) {
	o := r.col.Bounds()
	size := r.tipSize
	h := r.tipH + size*0.4
	for _, tp := range r.shown {
		b := tp.el.Bounds()
		tw := r.tipW[tp.slot]
		w := max(h, tw+size*0.8)
		cx, cy := box.X+b.X-o.X+b.W/2, box.Y+b.Y-o.Y+b.H
		// A line of the background around it keeps it apart from the text it
		// covers, a tab's title.
		p.Fill(ui.Rect{X: cx - w/2 - 1, Y: cy - h/2 - 1, W: w + 2, H: h + 2}, r.tipInk, size*0.3+1)
		p.Fill(ui.Rect{X: cx - w/2, Y: cy - h/2, W: w, H: h}, r.tipFill, size*0.3)
		p.Text(cx-tw/2, cy-r.tipH/2, tipLetters[tp.slot:tp.slot+1], size, r.tipInk)
	}
}

// strip builds the tabs, as a TabControl's: ui's look, or the skin's
// TabHeader look over each tab. A click on a tab expands a collapsed
// ribbon, and a double click collapses it. Under mu.
func (r *RibbonWidget) strip(c *ui.Context, p *int) ui.Element {
	t := c.Theme()
	s := &r.sb
	*s = ribbonStrip{tabs: ui.TabsKeyedBase(c, p, r.keys), p: p, look: lookOf(skinhook.KindTabHeader), taken: r.qatTaken}
	s.tabs.List.Gap(t.Space(1)).OnChange(r.changed).Children(func() {
		if r.bands != nil {
			r.bands(c)
			return
		}
		for i := range r.tabs {
			r.stripTab(c, i)
		}
	})
	switch {
	case s.double:
		// The first click of it expanded a collapsed ribbon: it stays so.
		r.collapsed = r.collapsed || !r.expanded
		r.swallow = true
	case s.clicked && r.swallow:
		r.swallow = false // the release of the double click
	case s.clicked:
		r.expanded = r.collapsed
		r.collapsed = false
	case !s.pressed:
		r.swallow = false // the double click's press ended with no click
	}
	return s.tabs.List
}

// stripTab builds tab i in the strip being built (sb), with its access key
// and its tip. Under mu.
func (r *RibbonWidget) stripTab(c *ui.Context, i int) {
	s, tb := &r.sb, r.tabs[i]
	var tab ui.Element
	if s.look != nil {
		tab = headerTab(c, s.tabs, i, i == *s.p, r.keys, s.look, r.disabled)
	} else {
		tab = documentTab(c, s.tabs, i, i == *s.p) // the shown one joins the groups row, Background as it
	}
	tab.Children(func() { ui.Text(c, tb.title).SingleLine() })
	// Alt and a tab's tip show its commands' tips.
	if sl := keyTip(tb.title, &s.taken); sl >= 0 {
		registerMnemonic(rune(tipLetters[sl])|0x20, tab, r, tb.act) // lower case, as foldKey gives
		if r.tips == tipsTabs {
			r.badge(tab, sl, nil, i)
		}
	}
	if i == *s.p && r.wantFocus && !c.Measuring() {
		// Focus goes to the tab shown, not to the strip around it, which
		// render would focus.
		tab.Focus()
		r.wantFocus = false
	}
	s.double = tab.DoubleClicked() || s.double
	s.clicked = tab.Clicked() || s.clicked
	s.pressed = tab.Pressed() || s.pressed
}

// groups builds the groups of tab tb in a row filled with the skin's Chrome
// (Background without a skin, as the shown tab above it), a line under it,
// which clips what the window has no room for. Under mu.
func (r *RibbonWidget) groups(c *ui.Context, tb *ribbonTab) {
	t := c.Theme()
	fill := t.Background
	if tk, _ := skinOf(c); tk != nil {
		fill = tk.Chrome
	}
	ui.Row(c).Background(fill).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Clip().Children(func() {
		outer := axis
		axis = vertical
		defer func() { axis = outer }()
		var taken uint64 // the commands' tips
		for i, g := range tb.groups {
			r.group(c, g, i == len(tb.groups)-1, &taken)
		}
	})
}

// group builds g: its items in a row three small rows high, its title under
// them, and a line after it but the last; with the commands' tips, each
// item's among those taken. Under mu.
func (r *RibbonWidget) group(c *ui.Context, g *ribbonGroup, last bool, taken *uint64) {
	t := c.Theme()
	row := t.Metrics.ControlHeight // ui scales Metrics with the text
	if row <= 0 {
		row = t.Rem(2.5)
	}
	gap := t.Space(0.5)
	high := 3*row + 2*gap
	col := ui.Column(c).Padding(t.Space(1), t.Space(1.5)).Gap(gap).AlignItems(ui.Center).Shrink(0)
	if g.title != "" {
		col.Label(g.title)
	}
	if !last {
		col.BorderWidth(0, 1, 0, 0).BorderColor(t.Border)
	}
	col.Children(func() {
		ui.Row(c).Height(high).Gap(t.Space(1)).AlignItems(ui.Start).Children(func() {
			for i := 0; i < len(g.items); {
				if g.items[i].large {
					if r.large(c, g.items[i].w, high) {
						r.commandTip(g.items[i].w, taken)
					}
					i++
					continue
				}
				j := i + 1
				for j < len(g.items) && j-i < 3 && !g.items[j].large {
					j++
				}
				smalls := g.items[i:j]
				ui.Column(c).Gap(gap).Children(func() {
					for _, it := range smalls {
						if e := render(c, it.w); e != (ui.Element{}) {
							e.MinHeight(row)
							flatTool(it.w, e)
							r.commandTip(it.w, taken)
						}
					}
				})
				i = j
			}
		})
		if g.title != "" {
			ui.Text(c, g.title).TextColor(t.TextMuted).FontSize(t.Rem(0.85)).SingleLine()
		}
	})
}

// large builds w, a tool button, drawn large and high, and reports whether
// it showed. Under mu.
func (r *RibbonWidget) large(c *ui.Context, w Control, high float32) bool {
	outer := toolLarge
	toolLarge = largeFace
	e := render(c, w)
	toolLarge = outer
	if e != (ui.Element{}) {
		e.Height(high)
		flatTool(w, e)
	}
	return e != (ui.Element{})
}

// commandTip gives w, a command that showed, its tip among those taken
// while the commands' tips show, and the Quick Access Toolbar's context
// menu once a method of the toolbar installed it. Under mu, after w builds.
func (r *RibbonWidget) commandTip(w Control, taken *uint64) {
	if r.cmdMenu != nil {
		r.cmdMenu(w)
	}
	if r.tips != tipsCommands {
		return
	}
	if s := keyTip(tipName(w), taken); s >= 0 {
		r.badge(w.base().el, s, w, -1)
	}
}

// largeFace shows b's icon, 32 DIPs at the text scale, above its text,
// centred, the text in two lines at most no wider than twice the icon; it
// names e as iconFace does. Under mu, while b builds.
func largeFace(c *ui.Context, e ui.Element, b *ToolButtonWidget) {
	scale := c.Preferences().TextScale
	if scale <= 0 {
		scale = 1
	}
	side := 32 * scale
	name := b.text
	if name == "" {
		name = b.tooltip
	}
	e.Label(name).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(c.Theme().Space(1)).Children(func() {
			var icon ui.Element
			if b.icon != nil {
				icon = ui.Icon(c, b.icon)
			} else {
				icon = bitmapIcon(c, b.png)
			}
			icon.FontSize(side) // an icon is as high as its font size
			if b.text != "" {
				ui.Text(c, b.text).TextAlign(ui.Center).MaxLines(2).MaxWidth(2 * side)
			}
		})
	})
}

// flatTool gives a plain tool button of the ribbon no face until hovered,
// pressed or checked, as a tool strip's: outside ui.Toolbar a button has
// one. Under mu, after w builds.
func flatTool(w Control, e ui.Element) {
	if b, ok := w.(*ToolButtonWidget); ok && b.look == plainLook && !b.selected && !e.Hovered() && !e.Pressed() {
		e.Background(ui.Color{}).BorderColor(ui.Color{})
	}
}

func (r *RibbonWidget) children() []Control {
	var cs []Control
	for _, t := range r.tabs {
		for _, g := range t.groups {
			for _, it := range g.items {
				cs = append(cs, it.w)
			}
		}
	}
	return cs
}
