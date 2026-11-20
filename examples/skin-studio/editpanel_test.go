package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
	"github.com/egoist/mygo/ui/widget/skin/skinfile"
)

// openEdit builds the studio and opens its Edit tab.
func openEdit(t *testing.T) (*widget.Tester, *editPanel) {
	t.Helper()
	reset(t)
	f, p := newStudioPanel()
	tt := widget.TestForm(t, f)
	tt.Find(widget.ByRole(ui.RoleTab).Label("Edit")).Click()
	return tt, p
}

// control returns the control of a row, a header field or a look: the node
// labeled name that is not the text of a label.
func control(t *testing.T, tt *widget.Tester, name string) *widget.Node {
	t.Helper()
	for _, n := range tt.FindAll(widget.ByLabel(name)) {
		if r := n.Role(); r != ui.RoleText && r != ui.RoleHeading && r != ui.RoleGroup { // a Fields row names its box too
			return n
		}
	}
	t.Fatalf("no control %q on the Edit tab", name)
	return nil
}

// enter replaces the text of a text field.
func enter(tt *widget.Tester, n *widget.Node, s string) {
	n.Focus()
	n.Key(0, ui.KeyEnd)
	for range []rune(n.Value()) {
		n.Key(0, ui.KeyBackspace)
	}
	n.Type(s)
}

// choose opens the combo box named combo and picks item.
func choose(tt *widget.Tester, combo, item string) { pick(tt, combo, item) }

// A calm light skin's Accent.Base and its dark one, as constants.
const (
	calmLightAccent = "#1c71d8"
	calmLightText   = "#1a1d21"
)

// The ninth tab header, Edit, fits the studio's own window at text scale 1,
// so it can be clicked on every OS, in every skin: its right edge is inside
// formWidth. The test measures the font of the OS it runs on; Linux's is
// wider than macOS's, so the Linux CI run is what guards Linux.
func TestStudioTabsFit(t *testing.T) {
	reset(t)
	// Text scale 1 only: each OS measures its own font, which is wider on
	// Linux than on macOS, so the Linux CI run is the guard for Linux. A
	// scale above 1 would count the font's width twice.
	for _, scale := range []float32{1} {
		for _, s := range []*skin.Skin{nil, skin.Calm, skin.Classic, skin.HighContrast} {
			skin.Use(s)
			tt := widget.TestForm(t, newStudio())
			tt.SetPreferences(ui.Preferences{TextScale: scale})
			tt.Frame()
			tabs := tt.FindAll(widget.ByRole(ui.RoleTab))
			if len(tabs) != 9 {
				t.Fatalf("%d tabs, want 9", len(tabs))
			}
			b := tabs[8].Bounds()
			if right := b.X + b.W; right > formWidth-8 {
				t.Errorf("%s at text scale %v: the Edit header's right edge is at %v, past the window's %d", describe(s), scale, right, formWidth)
			}
		}
	}
}

// The light Accent.Base row exists with the base's value, and Chart has
// eight rows.
func TestEditTabRows(t *testing.T) {
	tt, _ := openEdit(t)
	if got := control(t, tt, "Accent.Base").Value(); got != calmLightAccent {
		t.Errorf("Accent.Base = %q, want %q", got, calmLightAccent)
	}
	charts := 0
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleTextField)) {
		if strings.HasPrefix(n.Label(), "Chart.") {
			charts++
		}
	}
	if charts != 8 {
		t.Errorf("%d Chart rows, want 8", charts)
	}
	rows := 0
	for _, n := range tt.Nodes() {
		if n.Label == "RowHeight" && n.Role != ui.RoleText && n.Role != ui.RoleHeading {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("%d RowHeight controls, want 1", rows)
	}
	for _, kind := range []string{"Button", "TabHeader", "GridHeader", "HeaderBar"} {
		control(t, tt, "Look "+kind)
	}
}

// Review focus 1: setting light Accent.Base to #7c3aed makes a Suggested
// button on the Buttons page paint #7c3aed.
func TestEditReachesControls(t *testing.T) {
	tt, _ := openEdit(t)
	tt.SetDark(false)
	suggested := func() ui.Color {
		tt.Find(widget.ByRole(ui.RoleTab).Label("Buttons and toggles")).Click()
		b := tt.Find(widget.ByLabel("Suggested")).Bounds()
		c := tt.Image().RGBAAt(int(b.X)+6, int(b.Y+b.H/2)) // left of the text, inside the corner
		return ui.RGB(c.R, c.G, c.B)
	}
	if got := suggested(); got == ui.Hex("#7c3aed") {
		t.Fatalf("the Suggested button is %v before the edit", got)
	}
	tt.Find(widget.ByRole(ui.RoleTab).Label("Edit")).Click()
	enter(tt, control(t, tt, "Accent.Base"), "#7c3aed")
	if got, want := suggested(), ui.Hex("#7c3aed"); got != want {
		t.Errorf("the Suggested button paints %v, want %v", got, want)
	}
}

// Review focus 3: light Text #cccccc puts "Text on Background" in the
// status strip's report; setting it back clears it.
func TestEditStatusReport(t *testing.T) {
	tt, p := openEdit(t)
	tt.SetDark(false)
	tt.Find(widget.ByLabel("Ready"))
	text := control(t, tt, "Text")
	enter(tt, text, "#cccccc")
	if !strings.Contains(p.report, "Text on Background") {
		t.Errorf("report = %q, want Text on Background in it", p.report)
	}
	pairs := len(strings.Split(strings.TrimSpace(p.report), "\n"))
	if pairs != 4 { // Text on Background, on Chrome, on RowSelected, and TextMuted's pair
		t.Fatalf("%d pairs in %q, want 4", pairs, p.report)
	}
	tt.Find(widget.ByLabel("Contrast: 4 pairs under their floor"))
	enter(tt, text, calmLightText)
	tt.Find(widget.ByLabel("Contrast: all pairs pass"))
	if p.report != "" {
		t.Errorf("report = %q after the text is back, want none", p.report)
	}
}

// Review focus 4: a bad value is refused at the field and not applied.
func TestEditRefusesBadValues(t *testing.T) {
	tt, p := openEdit(t)
	tt.SetDark(false)
	enter(tt, control(t, tt, "Accent.Base"), "#7c3aed") // a good edit first: the skin is on
	text := control(t, tt, "Text")
	enter(tt, text, "#zzz")
	if msg := p.errs.Error(p.rows["Text"]); !strings.Contains(msg, "Text") || !strings.Contains(msg, "#zzz") {
		t.Errorf("error of the Text field = %q, want one naming Text and #zzz", msg)
	}
	if got := skin.Active().Tokens(false).Text; got != ui.Hex(calmLightText) {
		t.Errorf("Text = %v, want the last good %v", got, ui.Hex(calmLightText))
	}
	enter(tt, text, "#000000")
	if msg := p.errs.Error(p.rows["Text"]); msg != "" {
		t.Errorf("error after a good value = %q, want none", msg)
	}

	look := control(t, tt, "Look Button")
	enter(tt, look, "bg-nonsense")
	if msg := p.errs.Error(p.looks["Button"]); !strings.Contains(msg, "bg-nonsense") {
		t.Errorf("error of the Button look = %q, want one naming bg-nonsense", msg)
	}
	if got := skin.Active().Looks().Button; got != "" {
		t.Errorf("Looks.Button = %q, want none", got)
	}
	enter(tt, look, "border")
	if msg := p.errs.Error(p.looks["Button"]); msg != "" {
		t.Errorf("error after a good look = %q", msg)
	}
	if got := skin.Active().Looks().Button; got != "border" {
		t.Errorf("Looks.Button = %q, want border", got)
	}
}

// Review focus 5: the appearance being edited shows the dark tokens, and
// does not change the app's appearance.
func TestEditAppearanceSwitch(t *testing.T) {
	tt, p := openEdit(t)
	before := mygo.Theme.Source()
	if before != widget.ThemeSystem {
		t.Fatalf("the theme source is %q before, want system", before)
	}
	choose(tt, "Appearance", "Dark")
	if got := control(t, tt, "Accent.Base").Value(); got != "#78aeed" { // Calm Blue's dark accent
		t.Errorf("dark Accent.Base = %q, want #78aeed", got)
	}
	if got := mygo.Theme.Source(); got != widget.ThemeSystem {
		t.Errorf("the app's appearance is %q after the switch, want system", got)
	}
	enter(tt, control(t, tt, "Accent.Base"), "#123456")
	if got := p.ed.changed[1]["Accent.Base"]; got != "#123456" || len(p.ed.changed[0]) != 0 {
		t.Errorf("changed = %v, want only the dark Accent.Base", p.ed.changed)
	}
	choose(tt, "Appearance", "Light")
	if got := control(t, tt, "Accent.Base").Value(); got != calmLightAccent {
		t.Errorf("light Accent.Base = %q after the switch back, want %q", got, calmLightAccent)
	}
}

// A color's "…" opens the color dialog, whose answer is the token with its
// own opacity kept; a cancel changes nothing.
func TestEditColorDialog(t *testing.T) {
	tt, p := openEdit(t)
	var asked ui.Color
	answer, ok := ui.Hex("#112233"), true
	pickColor = func(_ *widget.Form, c ui.Color) (ui.Color, bool) { asked = c; return answer, ok }
	t.Cleanup(func() { pickColor = showColorDialog })

	tt.SetDark(false)
	tt.Find(widget.ByLabel("Choose Text")).Click()
	if asked != ui.Hex(calmLightText) {
		t.Errorf("the dialog started from %v, want %v", asked, ui.Hex(calmLightText))
	}
	if got := control(t, tt, "Text").Value(); got != "#112233" {
		t.Errorf("Text = %q, want #112233", got)
	}
	if got := p.ed.changed[0]["Text"]; got != "#112233" {
		t.Errorf("changed Text = %q, want #112233", got)
	}

	ok = false
	answer = ui.Hex("#445566")
	tt.Find(widget.ByLabel("Choose Text")).Click()
	if got := control(t, tt, "Text").Value(); got != "#112233" {
		t.Errorf("Text = %q after a cancel, want #112233", got)
	}

	// A translucent token keeps its alpha under an opaque answer.
	ok = true
	choose(tt, "Appearance", "Dark")
	tt.Find(widget.ByLabel("Choose Selection")).Click()
	if got := control(t, tt, "Selection").Value(); !strings.HasPrefix(got, "#445566") || len(got) != 9 {
		t.Errorf("dark Selection = %q, want #445566 and the old alpha", got)
	}
}

// A number box steps the size and the weight.
func TestEditNumberRows(t *testing.T) {
	tt, p := openEdit(t)
	tt.SetDark(false)
	n := control(t, tt, "RowHeight")
	n.Key(0, ui.KeyUp)
	if got := p.ed.changed[0]["RowHeight"]; got != "31" {
		t.Errorf("RowHeight = %q after Up, want 31", got)
	}
	w := control(t, tt, "Caption.Weight")
	w.Key(0, ui.KeyUp)
	if got := p.ed.changed[0]["Caption.Weight"]; got != "500" {
		t.Errorf("Caption.Weight = %q after Up, want 500", got)
	}
	flag := control(t, tt, "ScrollbarStyle.Square")
	flag.Click()
	if got := p.ed.changed[0]["ScrollbarStyle.Square"]; got != "true" {
		t.Errorf("ScrollbarStyle.Square = %q after a click, want true", got)
	}
}

// The header: base, palette, accent, density, name.
func TestEditHeader(t *testing.T) {
	tt, p := openEdit(t)
	choose(tt, "Edit base", "Classic")
	if p.ed.base != "classic" || p.ed.density != skin.Compact {
		t.Errorf("base = %q density = %v, want classic and Compact", p.ed.base, p.ed.density)
	}
	if got := skin.Active(); got == nil || got.Name() != "Classic" || got.Density() != skin.Compact {
		t.Errorf("the skin in use is not Classic Compact")
	}
	choose(tt, "Edit palette", "Slate")
	if p.ed.palette != "Slate" || skin.Active().Palette().Name() != "Slate" {
		t.Errorf("palette = %q, skin's %v", p.ed.palette, skin.Active().Palette())
	}
	choose(tt, "Density", "Standard")
	if skin.Active().Density() != skin.Standard {
		t.Error("the density is not Standard")
	}
	enter(tt, control(t, tt, "Skin name"), "Mine")
	if skin.Active().Name() != "Mine" {
		t.Errorf("name = %q, want Mine", skin.Active().Name())
	}

	choose(tt, "Edit palette", "Accent…")
	acc := control(t, tt, "Accent")
	enter(tt, acc, "#7c3aed")
	if got := skin.Active().Palette().Name(); got != "#7c3aed" {
		t.Errorf("palette = %q, want #7c3aed", got)
	}
	if got := control(t, tt, "Accent.Base").Value(); got == "" || got == "#44607a" {
		t.Errorf("Accent.Base = %q: the rows did not follow the accent", got)
	}
	enter(tt, acc, "#zzz")
	if msg := p.errs.Error(p.accent); !strings.Contains(msg, "#zzz") {
		t.Errorf("accent error = %q, want one naming #zzz", msg)
	}
	if got := skin.Active().Palette().Name(); got != "#7c3aed" {
		t.Errorf("palette = %q after a bad accent, want #7c3aed", got)
	}
	enter(tt, acc, "")
	if msg := p.errs.Error(p.accent); msg == "" || p.ed.accent != "#7c3aed" {
		t.Errorf("an empty accent: error %q, accent %q, want an error and the accent kept", msg, p.ed.accent)
	}
	choose(tt, "Edit palette", "Slate")
	if got := skin.Active().Palette().Name(); got != "Slate" || p.ed.accent != "" {
		t.Errorf("palette = %q, accent %q after Slate, want Slate and no accent", got, p.ed.accent)
	}
}

// Reset goes back to the base as it ships.
func TestEditReset(t *testing.T) {
	tt, p := openEdit(t)
	choose(tt, "Edit base", "Classic")
	enter(tt, control(t, tt, "Text"), "#000000")
	enter(tt, control(t, tt, "Look Button"), "border")
	tt.Find(widget.ByLabel("Reset")).Click()
	sameSkin(t, p.ed.build(), skin.Classic)
	sameSkin(t, skin.Active(), skin.Classic)
	if got := control(t, tt, "Text").Value(); got != "#1a1d21" {
		t.Errorf("Text = %q after Reset, want Classic's #1a1d21", got)
	}
	if got := control(t, tt, "Look Button").Value(); got != skin.Classic.Looks().Button {
		t.Errorf("Look Button = %q after Reset, want Classic's", got)
	}
}

// Nothing is applied, and the strip is as it was, until the first edit; an
// edit shows "Edited" in the tool strip's skin box, and choosing a skin or a
// palette there resets the editor to it.
func TestEditToolStrip(t *testing.T) {
	tt, p := openEdit(t)
	if skin.Active() != nil {
		t.Error("opening the Edit tab turned a skin on")
	}
	tt.Find(widget.ByLabel("Ready"))
	if got := tt.Find(widget.ByLabel("Skin")).Value(); got != "none" {
		t.Errorf("Skin box = %q, want none", got)
	}
	enter(tt, control(t, tt, "Text"), "#000000")
	if got := tt.Find(widget.ByLabel("Skin")).Value(); got != "edited" {
		t.Errorf("Skin box = %q after an edit, want edited", got)
	}
	choose(tt, "Skin", "Edited") // the edited skin is on already: nothing resets
	if p.ed.base != "calm" || p.ed.changed[0]["Text"] != "#000000" || skin.Active().Tokens(false).Text != ui.Hex("#000000") {
		t.Errorf("choosing Edited reset the editor: base %q, changed %v", p.ed.base, p.ed.changed)
	}
	choose(tt, "Skin", "Classic")
	tt.Find(widget.ByLabel("Ready")) // nothing is edited yet
	if p.ed.base != "classic" || len(p.ed.changed[0]) != 0 {
		t.Errorf("editor = %q with %v changed, want classic and none", p.ed.base, p.ed.changed)
	}
	if got := tt.Find(widget.ByLabel("Skin")).Value(); got != "classic" {
		t.Errorf("Skin box = %q, want classic", got)
	}
	if got := control(t, tt, "Text").Value(); got != "#1a1d21" {
		t.Errorf("Text = %q, want Classic's #1a1d21", got)
	}
	enter(tt, control(t, tt, "Text"), "#000000")
	choose(tt, "Palette", "Slate")
	if p.ed.palette != "Slate" || len(p.ed.changed[0]) != 0 {
		t.Errorf("editor palette %q with %v changed, want Slate and none", p.ed.palette, p.ed.changed)
	}
	enter(tt, control(t, tt, "Text"), "#000000")
	choose(tt, "Skin", "None")
	if skin.Active() != nil || p.ed.base != "calm" {
		t.Errorf("skin %v, editor base %q, want none and calm", skin.Active(), p.ed.base)
	}
}

// With saveFile answering a path, Save writes skinfile.Format's bytes; with
// openFile answering it, Open loads the file and the rows show its values;
// a cancel does nothing and a bad file is an error on the button.
func TestEditOpenSaveButtons(t *testing.T) {
	tt, p := openEdit(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "brand.json")
	var asked string
	saveFile = func(_ *widget.Form, title, name string, _ ...widget.Filter) string { asked = name; return path }
	openFile = func(_ *widget.Form, title string, _ ...widget.Filter) string { return path }
	t.Cleanup(func() { saveFile, openFile = widget.SaveFile, widget.OpenFile })

	enter(tt, control(t, tt, "Skin name"), "Brand")
	enter(tt, control(t, tt, "Background"), "#fffaf0")
	tt.Find(widget.ByLabel("Save…")).Click()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := skinfile.Format(p.ed.build(), skin.Calm)
	if string(got) != string(want) || !strings.Contains(string(got), `"Background": "#fffaf0"`) {
		t.Errorf("Save wrote\n%s\nwant\n%s", got, want)
	}
	if asked != "Brand.json" {
		t.Errorf("Save suggested %q, want Brand.json", asked)
	}

	tt.Find(widget.ByLabel("Reset")).Click()
	if got := control(t, tt, "Background").Value(); got == "#fffaf0" {
		t.Fatal("Reset left the edit")
	}
	tt.Find(widget.ByLabel("Open…")).Click()
	if got := control(t, tt, "Background").Value(); got != "#fffaf0" {
		t.Errorf("Background = %q after Open, want #fffaf0", got)
	}
	if got := control(t, tt, "Skin name").Value(); got != "Brand" {
		t.Errorf("name = %q after Open, want Brand", got)
	}
	if a := skin.Active(); a == nil || a.Name() != "Brand" || a.Tokens(false).Background != ui.Hex("#fffaf0") {
		t.Errorf("the skin in use after Open is not Brand")
	}

	// A bad file: an error on the button, the editor as it was.
	if err := os.WriteFile(path, []byte(`{"version": 1, "name": "X", "base": "Calm", "light": {"Text": "red"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tt.Find(widget.ByLabel("Open…")).Click()
	if msg := p.errs.Error(p.openBtn); !strings.Contains(msg, "light.Text") {
		t.Errorf("error of Open = %q, want one naming light.Text", msg)
	}
	if got := control(t, tt, "Skin name").Value(); got != "Brand" {
		t.Errorf("name = %q after a bad file, want Brand", got)
	}
	// A cancel is not an error, and clears the last one.
	badMsg := p.errs.Error(p.openBtn)
	openFile = func(*widget.Form, string, ...widget.Filter) string { return "" }
	tt.Find(widget.ByLabel("Open…")).Click()
	if got := control(t, tt, "Skin name").Value(); got != "Brand" {
		t.Errorf("name = %q after a cancel, want Brand", got)
	}
	if msg := p.errs.Error(p.openBtn); msg != badMsg {
		t.Errorf("error of Open = %q after a cancel, want it untouched: %q", msg, badMsg)
	}
	// An unreadable path is an error too.
	openFile = func(*widget.Form, string, ...widget.Filter) string { return filepath.Join(dir, "nope.json") }
	tt.Find(widget.ByLabel("Open…")).Click()
	if msg := p.errs.Error(p.openBtn); !strings.Contains(msg, "nope.json") {
		t.Errorf("error of Open = %q, want one naming nope.json", msg)
	}
	// A cancel of Save writes nothing.
	saveFile = func(*widget.Form, string, string, ...widget.Filter) string { return "" }
	tt.Find(widget.ByLabel("Save…")).Click()
	if msg := p.errs.Error(p.saveBtn); msg != "" {
		t.Errorf("error of Save = %q after a cancel, want none", msg)
	}
	// A path that cannot be written is an error on the button.
	saveFile = func(*widget.Form, string, string, ...widget.Filter) string { return dir } // a directory
	tt.Find(widget.ByLabel("Save…")).Click()
	if msg := p.errs.Error(p.saveBtn); !strings.Contains(msg, dir) {
		t.Errorf("error of Save = %q, want one naming %s", msg, dir)
	}
}

// Review focus 2 through the panel: Save, then Open in another studio gives
// the same skin; a file saved against Classic opens with base Classic.
func TestEditSaveThenOpenInAnotherStudio(t *testing.T) {
	tt, p := openEdit(t)
	path := filepath.Join(t.TempDir(), "mine.json")
	saveFile = func(*widget.Form, string, string, ...widget.Filter) string { return path }
	openFile = func(*widget.Form, string, ...widget.Filter) string { return path }
	t.Cleanup(func() { saveFile, openFile = widget.SaveFile, widget.OpenFile })

	choose(tt, "Edit base", "Classic")
	choose(tt, "Edit palette", "Slate")
	enter(tt, control(t, tt, "Accent.Base"), "#123456")
	enter(tt, control(t, tt, "Look Card"), "border")
	choose(tt, "Appearance", "Dark")
	enter(tt, control(t, tt, "Danger.Base"), "#ff5555")
	tt.Find(widget.ByLabel("Save…")).Click()
	saved := p.ed.build()

	tt2, p2 := openEdit(t)
	tt2.Find(widget.ByLabel("Open…")).Click()
	if p2.ed.base != "classic" {
		t.Errorf("base = %q, want classic", p2.ed.base)
	}
	sameSkin(t, p2.ed.build(), saved)
	sameSkin(t, skin.Active(), saved)
	if got := tt2.Find(widget.ByLabel("Edit base")).Value(); got != "classic" {
		t.Errorf("the base box shows %q, want classic", got)
	}
	if got := tt2.Find(widget.ByLabel("Edit palette")).Value(); got != "Slate" {
		t.Errorf("the palette box shows %q, want Slate", got)
	}
}

var statusText = regexp.MustCompile(`^Contrast: (all pairs pass|\d+ pairs under their floor)$`)

// The status strip says Ready until the first edit, then the report.
func TestEditStatusStartsReady(t *testing.T) {
	tt, p := openEdit(t)
	tt.Find(widget.ByLabel("Ready"))
	enter(tt, control(t, tt, "Text"), "#000000")
	n := 0
	for _, x := range tt.Nodes() {
		if statusText.MatchString(x.Label) {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d status texts match Contrast: ..., want 1", n)
	}
	if p.report != "" {
		t.Errorf("report = %q for a skin that passes", p.report)
	}
}
