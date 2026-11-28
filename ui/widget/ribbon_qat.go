package widget

import (
	"slices"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// ribbonQAT is a ribbon's Quick Access Toolbar.
type ribbonQAT struct {
	buttons []*ToolButtonWidget // shown, in order
	// known are the buttons AddQuickAccess gave, which SetQuickAccess finds
	// by name besides the ribbon's commands.
	known    []*ToolButtonWidget
	below    bool
	onChange func()
	// addMenus and menus are the context menus of the ribbon's commands and
	// of the toolbar's buttons, made once for each button.
	addMenus, menus map[*ToolButtonWidget]func(m *ui.Menu)
	// drawn are the tips of the buttons the toolbar showed in this frame,
	// the first nine, given their badges while the tabs' tips show. fill
	// builds the buttons, made once, in ctx.
	drawn []ribbonTip
	fill  func()
	ctx   *ui.Context
	// form is the form that built the ribbon last. seen is set as the ribbon
	// builds with the toolbar in the form's header bar, which draws it in
	// the next frame from it, as it builds before the ribbon; inHeader is
	// set by the header bar that drew it in this frame.
	form           *Form
	seen, inHeader bool
}

// qatRibbons are the ribbons with a toolbar, among which a header bar finds
// its form's. Under mu.
//
// shortcut: it never shrinks, as a ribbon lives as long as its form; drop
// the closed forms' ribbons if an app makes many.
var qatRibbons []*RibbonWidget

// headerQAT draws a ribbon's toolbar in a header bar's start area, and
// formQATEnd ends a form's build: both installed by the first ribbon with a
// toolbar, so that an app with none does not link them. Under mu.
var (
	headerQAT  func(c *ui.Context, h *HeaderBarWidget)
	formQATEnd func(c *ui.Context, w *Form)
)

// AddQuickAccess adds buttons to the Quick Access Toolbar, after those it
// holds: small icons in a row above the tabs, in the title bar's start area
// when the form has a HeaderBar, or below the ribbon (SetQuickAccessBelow).
// A button may be one of the ribbon's commands or not. Calling it, with no
// buttons too, lets the user change the toolbar from the commands' context
// menus. nil, and a button the toolbar holds, are skipped.
func (r *RibbonWidget) AddQuickAccess(buttons ...*ToolButtonWidget) *RibbonWidget {
	r.change(func() {
		q := r.qatOn()
		for _, b := range buttons {
			if b == nil || slices.Contains(q.buttons, b) {
				continue
			}
			q.buttons = append(slices.Clip(q.buttons), b) // a new array: a build may be ranging over the old
			if !slices.Contains(q.known, b) {
				q.known = append(q.known, b)
				if !slices.Contains(r.children(), Control(b)) {
					r.qatOnly = append(r.qatOnly, b)
				}
			}
		}
		r.retake()
	})
	return r
}

// SetQuickAccessBelow shows the Quick Access Toolbar below the ribbon, or
// above its tabs again; the user does it from a toolbar button's context
// menu.
func (r *RibbonWidget) SetQuickAccessBelow(on bool) *RibbonWidget {
	r.change(func() { r.qatOn().below = on })
	return r
}

// QuickAccess returns the names of the Quick Access Toolbar's buttons, in
// order: each one's text, or its tooltip with none. SetQuickAccess sets the
// toolbar from such names, so an app saves and restores what the user
// chose. A name the ribbon has no button for is skipped. Neither runs
// OnQuickAccessChange.
func (r *RibbonWidget) QuickAccess() []string {
	mu.Lock()
	defer mu.Unlock()
	if r.qat == nil {
		return nil
	}
	names := make([]string, len(r.qat.buttons))
	for i, b := range r.qat.buttons {
		names[i] = tipName(b)
	}
	return names
}

func (r *RibbonWidget) SetQuickAccess(names ...string) *RibbonWidget {
	r.change(func() {
		q := r.qatOn()
		var bs []*ToolButtonWidget
		for _, name := range names {
			if b := r.command(name); b != nil && !slices.Contains(bs, b) {
				bs = append(bs, b)
			}
		}
		q.buttons = bs
		r.retake()
	})
	return r
}

// OnQuickAccessChange runs fn on the event goroutine after the user adds a
// button to the Quick Access Toolbar or removes one, from their context
// menus; not after the setters, nor when the toolbar moves below the ribbon.
func (r *RibbonWidget) OnQuickAccessChange(fn func()) *RibbonWidget {
	r.change(func() { r.qatOn().onChange = fn })
	return r
}

// qatOn installs the toolbar. Under mu.
func (r *RibbonWidget) qatOn() *ribbonQAT {
	if r.qat == nil {
		q := &ribbonQAT{}
		q.fill = func() { r.fillQAT(q.ctx) }
		r.qat, r.qatHook, r.cmdMenu = q, r.buildQAT, r.commandMenu
		headerQAT, formQATEnd = headerBarQAT, endQAT
		qatRibbons = append(qatRibbons, r)
	}
	return r.qat
}

// retake keeps the tip slots of the toolbar's digits, 1 to 9, from the
// tabs. Under mu.
func (r *RibbonWidget) retake() {
	r.qatTaken = 0
	for i := range min(len(r.qat.buttons), 9) {
		r.qatTaken |= 1 << (26 + i)
	}
}

// command returns the tool button named name: a command of the ribbon, else
// one AddQuickAccess gave; nil for none. Under mu.
func (r *RibbonWidget) command(name string) *ToolButtonWidget {
	for _, t := range r.tabs {
		for _, g := range t.groups {
			for _, it := range g.items {
				if b, ok := it.w.(*ToolButtonWidget); ok && tipName(b) == name {
					return b
				}
			}
		}
	}
	for _, b := range r.qat.known {
		if tipName(b) == name {
			return b
		}
	}
	return nil
}

// qatChanged posts OnQuickAccessChange, for a change of the user's. Under mu.
func (r *RibbonWidget) qatChanged() {
	if fn := r.qat.onChange; fn != nil {
		Post(fn)
	}
}

// buildQAT builds the toolbar where it goes: before the tabs (below false),
// above them, or nothing where the form's header bar draws it; after the
// groups (below true), below them, with the badges of its digits while the
// tabs' tips show. Under mu, as the ribbon builds (qatHook).
func (r *RibbonWidget) buildQAT(c *ui.Context, below bool) {
	q := r.qat
	if below {
		if q.below {
			r.qatBar(c)
		}
		if r.tips == tipsTabs {
			for _, tp := range q.drawn {
				r.badge(tp.el, int(tp.slot), tp.w, -1)
			}
		}
		return
	}
	f := buildingForm
	header := !q.below && f != nil && f.header != nil && !f.header.hidden
	q.form = f
	switch {
	case header && q.inHeader: // the header bar drew it in this frame
	case header:
		// The header bar built before the ribbon: it draws the toolbar in
		// the next frame.
		q.drawn = q.drawn[:0]
		if !c.Measuring() {
			c.AnimationFrame()
		}
	case q.below:
		q.drawn = q.drawn[:0]
	default:
		r.qatBar(c)
	}
	q.seen, q.inHeader = header, false
}

// headerBarQAT draws, in h's start area, the toolbar of the ribbon that its
// form built in the last frame with the toolbar there. Under mu, as h
// builds (headerQAT).
func headerBarQAT(c *ui.Context, h *HeaderBarWidget) {
	for _, r := range qatRibbons {
		if q := r.qat; q.form == h.win && q.seen {
			r.qatBar(c)
			q.inHeader = true
			return
		}
	}
}

// endQAT ends w's build: the toolbar its header bar drew is of a ribbon that
// did not build in this frame, as its page is no longer shown, and goes in
// the next. Under mu (formQATEnd).
func endQAT(c *ui.Context, w *Form) {
	for _, r := range qatRibbons {
		if q := r.qat; q.form == w && q.inHeader {
			q.seen, q.inHeader = false, false
			q.drawn = q.drawn[:0]
			if !c.Measuring() {
				c.AnimationFrame()
			}
		}
	}
}

// qatBar builds the toolbar's buttons in a row, small icons, each named by
// its text. Under mu.
func (r *RibbonWidget) qatBar(c *ui.Context) {
	q := r.qat
	q.drawn = q.drawn[:0]
	if len(q.buttons) == 0 {
		return
	}
	q.ctx = c
	outer, outerAxis := toolLarge, axis
	toolLarge, axis = qatFace, horizontal
	ui.Row(c).Gap(c.Theme().Space(0.5)).AlignItems(ui.Center).Children(q.fill)
	toolLarge, axis = outer, outerAxis
}

// fillQAT builds the toolbar's buttons, each with its context menu. Under
// mu, in qatBar's row.
func (r *RibbonWidget) fillQAT(c *ui.Context) {
	q := r.qat
	for i, b := range q.buttons {
		e := render(c, b)
		if e == (ui.Element{}) {
			continue
		}
		flatTool(b, e)
		b.el.ContextMenu(r.menuOf(&q.menus, b, false))
		if i < 9 {
			q.drawn = append(q.drawn, ribbonTip{b.el, int8(26 + i), b, -1})
		}
	}
}

// qatFace draws b's icon alone, at the size of a tool strip's, and names
// the button by its text, or its tooltip with none. Under mu, while b builds
// in the toolbar (toolLarge).
func qatFace(c *ui.Context, e ui.Element, b *ToolButtonWidget) {
	side := float32(0)
	if tk, scale := skinOf(c); tk != nil {
		side = tk.ToolbarIcon * scale
	}
	iconFace(c, e, b.icon, b.png, "", tipName(b), true, side)
}

// commandMenu gives w, a command of the ribbon that showed, the menu that
// adds it to the toolbar: a tool button's, not another control's. A
// command with a context menu of its own keeps that one. Under mu (cmdMenu).
func (r *RibbonWidget) commandMenu(w Control) {
	if b, ok := w.(*ToolButtonWidget); ok {
		b.el.ContextMenu(r.menuOf(&r.qat.addMenus, b, true))
	}
}

// menuOf returns b's menu in *menus, made once: the command's (addMenu)
// when add is set, else the toolbar button's (qatMenu). Under mu.
func (r *RibbonWidget) menuOf(menus *map[*ToolButtonWidget]func(m *ui.Menu), b *ToolButtonWidget, add bool) func(m *ui.Menu) {
	fn := (*menus)[b]
	if fn == nil {
		if *menus == nil {
			*menus = map[*ToolButtonWidget]func(m *ui.Menu){}
		}
		if add {
			fn = func(m *ui.Menu) { r.addMenu(m, b) }
		} else {
			fn = func(m *ui.Menu) { r.qatMenu(m, b) }
		}
		(*menus)[b] = fn
	}
	return fn
}

// addMenu is a command's menu: "Add to Quick Access Toolbar", disabled
// while the toolbar holds it. Under mu, as the command builds.
func (r *RibbonWidget) addMenu(m *ui.Menu, b *ToolButtonWidget) {
	q := r.qat
	in := slices.Contains(q.buttons, b)
	if m.Item(l10n.AddToQuickAccess()).Disabled(in).Chosen() && !in {
		q.buttons = append(slices.Clip(q.buttons), b)
		r.retake()
		r.qatChanged()
	}
}

// qatMenu is a toolbar button's menu: "Remove from Quick Access Toolbar",
// and "Show below the ribbon", checked while it is. Under mu, as the button
// builds in the toolbar.
func (r *RibbonWidget) qatMenu(m *ui.Menu, b *ToolButtonWidget) {
	q := r.qat
	if m.Item(l10n.RemoveFromQuickAccess()).Chosen() {
		if i := slices.Index(q.buttons, b); i >= 0 {
			q.buttons = slices.Delete(slices.Clone(q.buttons), i, i+1) // a new array: fillQAT ranges over the old
			r.retake()
			r.qatChanged()
		}
	}
	if m.Item(l10n.ShowBelowRibbon()).Checked(q.below).Chosen() {
		q.below = !q.below
	}
}

// ribbonCats are a ribbon's categories of contextual tabs, in the order
// of their first tab, and whether each shows. cat, fill, ink and run are
// the band being built, which band and tabs build, made once, in ctx.
type ribbonCats struct {
	names      []string
	shown      map[string]bool
	cat        string
	fill, ink  ui.Color
	run        [2]int
	band, tabs func()
	ctx        *ui.Context
}

// AddContextualTab adds a tab titled title to category, after the other
// tabs; the AddGroup calls that follow fill it, as they fill AddTab's. A
// category's tabs show only while SetCategoryVisible(category, true), under
// a band of the category's name, in the skin's series of colors (its Chart,
// as charts draw), by the order in which the categories came.
func (r *RibbonWidget) AddContextualTab(category, title string) *RibbonWidget {
	r.change(func() {
		k := r.catsOn()
		r.addTab(title)
		r.tabs[len(r.tabs)-1].category = category
		if category != "" && !slices.Contains(k.names, category) {
			k.names = append(k.names, category)
		}
	})
	return r
}

// SetCategoryVisible shows or hides the tabs of category, as the app's
// selection calls for them (a picture's Format tab). Hiding the category of
// the tab shown shows the first tab.
func (r *RibbonWidget) SetCategoryVisible(category string, visible bool) *RibbonWidget {
	r.change(func() {
		k := r.catsOn()
		k.shown[category] = visible
		if i := r.tab.value; !visible && category != "" && i >= 0 && i < len(r.tabs) && r.tabs[i].category == category {
			r.tab.set(0)
		}
	})
	return r
}

// catsOn installs the categories. Under mu.
func (r *RibbonWidget) catsOn() *ribbonCats {
	if r.cats == nil {
		k := &ribbonCats{shown: map[string]bool{}}
		k.band = func() { r.band(k.ctx) }
		k.tabs = func() {
			for i := k.run[0]; i < k.run[1]; i++ {
				r.stripTab(k.ctx, i)
			}
		}
		r.cats, r.bands = k, r.bandTabs
	}
	return r.cats
}

// bandTabs builds the strip's tabs: a category's in a column under its
// band, or hidden, out of the layout and the arrows' way, while the
// category is; the others as the strip does. Under mu, in the strip (bands).
func (r *RibbonWidget) bandTabs(c *ui.Context) {
	k, s, t := r.cats, &r.sb, c.Theme()
	k.ctx = c
	colors := plainChartColors
	if tk := skinhook.Get(t.Dark); tk != nil {
		colors = tk.Chart[:]
	}
	for i := 0; i < len(r.tabs); {
		cat := r.tabs[i].category
		if cat == "" {
			r.stripTab(c, i)
			i++
			continue
		}
		j := i + 1
		for j < len(r.tabs) && r.tabs[j].category == cat {
			j++
		}
		if !k.shown[cat] {
			for ; i < j; i++ {
				s.tabs.Tab(i).Hidden(true).Disabled(true)
			}
			continue
		}
		s.tabs.List.AlignItems(ui.End) // the other tabs on the line of the band's
		k.cat, k.run = cat, [2]int{i, j}
		k.fill = chartColor(colors, slices.Index(k.names, cat))
		k.ink = t.Text
		if contrastOf(t.Background, k.fill) > contrastOf(k.ink, k.fill) {
			k.ink = t.Background
		}
		ui.Column(c).Shrink(0).Children(k.band)
		i = j
	}
}

// band builds the band being built over its tabs. Under mu (ribbonCats.band).
func (r *RibbonWidget) band(c *ui.Context) {
	k, t := r.cats, c.Theme()
	ui.Text(c, k.cat).SingleLine().FontSize(t.Rem(0.8)).TextColor(k.ink).Background(k.fill).
		Padding(t.Space(0.5), t.Space(2))
	ui.Row(c).Gap(t.Space(1)).Children(k.tabs)
}
