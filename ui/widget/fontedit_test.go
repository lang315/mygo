package widget

import (
	"fmt"
	"image/color"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
	"golang.org/x/image/font/gofont/gomono"
)

// Go Mono, registered under two names: a row drawn in its own family draws
// "iiiiiiii" as wide as "mmmmmmmm", which the default face does not.
const (
	monoI = "iiiiiiii"
	monoM = "mmmmmmmm"
)

var registerMono = sync.OnceValue(func() error {
	for _, name := range []string{monoI, monoM} {
		if err := ui.RegisterFont(gomono.TTF, name); err != nil {
			return err
		}
	}
	return nil
})

// fontFamilies are families of the cases, out of order, as a stub of
// ui.FontFamilies.
func fontFamilies() []ui.FontFamily {
	return []ui.FontFamily{
		{Name: "Gamma"}, {Name: "Candara"}, {Name: "Arial Black"}, {Name: "Barlow"},
		{Name: "Courier", Monospaced: true}, {Name: "Arial"}, {Name: "Charter"}, {Name: "Arcade Mono", Monospaced: true},
	}
}

// newFontEdit is a FontEdit listing fams.
func newFontEdit(fams func() []ui.FontFamily) *FontEditWidget {
	f := FontEdit()
	f.families = fams
	return f
}

// fontTester shows f, 300 wide, above a button "Other", in a window of 400
// by 500.
func fontTester(t *testing.T, f *FontEditWidget) *Tester {
	t.Helper()
	calmOff()
	tt := NewTester(t, VBox(0).Add(f.SetSize(300, 0), Button("Other")), 400, 500)
	tt.Settle()
	return tt
}

// fontField is the FontEdit's field.
func fontField(tt *Tester) *Node { return tt.Find(ByRole(ui.RoleComboBox)) }

// Case N1: the drop-down lists the families sorted, case ignored, the
// app's registered ones first as ui.FontFamilies does, each row in its own
// family.
func TestFontEditListsFamilies(t *testing.T) {
	if err := registerMono(); err != nil {
		t.Fatal(err)
	}
	f := newFontEdit(func() []ui.FontFamily {
		return []ui.FontFamily{{Name: monoM, Monospaced: true}, {Name: "beta"}, {Name: "Zed", Registered: true}, {Name: "Alpha"}, {Name: monoI, Monospaced: true}}
	})
	tt := fontTester(t, f)
	fontField(tt).Click()
	if !lookOpen(tt) {
		t.Fatal("a click on the field did not open the drop-down")
	}
	wantRows(t, tt, "", "Zed", "Alpha", "beta", monoI, monoM)
	i, m := textAt(t, tt, monoI), textAt(t, tt, monoM)
	if !within(i.W, m.W, 0.5) {
		t.Fatalf("the rows are not drawn in their own family: %s is %v wide, %s %v (equal in Go Mono)", monoI, i.W, monoM, m.W)
	}
}

// Case N2: constructing, setting and drawing the closed field reads no
// family list; the first open reads it once, and later opens and typing
// read it no more.
func TestFontEditLazy(t *testing.T) {
	var reads atomic.Int32
	f := newFontEdit(func() []ui.FontFamily { reads.Add(1); return fontFamilies() })
	f.SetFamily("Arial").SetFixedPitchOnly(true).SetFixedPitchOnly(false)
	_ = f.Family()
	tt := fontTester(t, f)
	for range 5 {
		tt.Frame()
	}
	if n := reads.Load(); n != 0 {
		t.Fatalf("the closed field read the family list %d times", n)
	}
	fontField(tt).Click()
	if n := reads.Load(); n != 1 {
		t.Fatalf("the first open read the family list %d times, want once", n)
	}
	lookType(tt, "ar")
	lookKeys(tt, ui.KeyEscape)
	fontField(tt).Click()
	f.SetFixedPitchOnly(true)
	tt.Settle()
	if n := reads.Load(); n != 1 {
		t.Fatalf("after typing, a second open and SetFixedPitchOnly the list was read %d times, want once", n)
	}
}

// Case N2b: with no family at all (a browser that registered none), the
// drop-down says so, and the list is read once all the same.
func TestFontEditNoFamilies(t *testing.T) {
	var reads atomic.Int32
	f := newFontEdit(func() []ui.FontFamily { reads.Add(1); return nil })
	tt := fontTester(t, f)
	fontField(tt).Click()
	if !tt.HasText(l10n.NoResults()) {
		t.Fatalf("no %q with no family: %q", l10n.NoResults(), tt.Texts())
	}
	lookKeys(tt, ui.KeyEnter, ui.KeyDown, ui.KeyEscape)
	fontField(tt).Click()
	if n := reads.Load(); n != 1 {
		t.Fatalf("an empty list was read %d times, want once", n)
	}
	if got := f.Family(); got != "" {
		t.Fatalf("Family is %q", got)
	}
}

// Case N3: typing ar lists the families that start with it first, then
// those that hold it, case ignored; Enter picks the first.
func TestFontEditTypeFilter(t *testing.T) {
	var log editLog
	f := newFontEdit(fontFamilies).OnChange(log.add("change"))
	tt := fontTester(t, f)
	fontField(tt).Click()
	searchBox(tt).Type("ar")
	wantRows(t, tt, "ar", "Arcade Mono", "Arial", "Arial Black", "Barlow", "Candara", "Charter")
	lookKeys(tt, ui.KeyBackspace, ui.KeyBackspace)
	lookType(tt, "aR") // case ignored
	wantRows(t, tt, "aR", "Arcade Mono", "Arial", "Arial Black", "Barlow", "Candara", "Charter")
	lookType(tt, "ia")
	wantRows(t, tt, "aRia", "Arial", "Arial Black")
	lookKeys(tt, ui.KeyBackspace, ui.KeyBackspace, ui.KeyBackspace, ui.KeyBackspace)
	lookType(tt, "zz")
	wantRows(t, tt, "zz")
	lookKeys(tt, ui.KeyBackspace, ui.KeyBackspace)
	lookType(tt, "an")
	wantRows(t, tt, "an", "Candara")
	lookKeys(tt, ui.KeyEnter)
	if lookOpen(tt) {
		t.Fatal("Enter did not close the drop-down")
	}
	if got := f.Family(); got != "Candara" {
		t.Fatalf("Enter picked %q, want the first match Candara", got)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("picking ran %q, want one change", got)
	}
}

// Case N4: with fixed pitch only, proportional families are left out.
func TestFontEditFixedPitch(t *testing.T) {
	f := newFontEdit(fontFamilies).SetFixedPitchOnly(true)
	tt := fontTester(t, f)
	fontField(tt).Click()
	wantRows(t, tt, "", "Arcade Mono", "Courier")
	searchBox(tt).Type("c")
	wantRows(t, tt, "c", "Courier", "Arcade Mono")
	lookKeys(tt, ui.KeyEscape)
	f.SetFixedPitchOnly(false)
	tt.Settle()
	fontField(tt).Click()
	wantRows(t, tt, "", "Arcade Mono", "Arial", "Arial Black", "Barlow", "Candara", "Charter", "Courier", "Gamma")
}

// Case N5: SetFixedPitchOnly while the drop-down shows, with a row
// highlighted past the new end, neither panics nor picks a family gone.
func TestFontEditFixedPitchWhileOpen(t *testing.T) {
	var log editLog
	f := newFontEdit(fontFamilies).OnChange(log.add("change"))
	tt := fontTester(t, f)
	fontField(tt).Click()
	for range 8 {
		lookKeys(tt, ui.KeyDown) // Gamma, the last
	}
	f.SetFixedPitchOnly(true)
	tt.Settle()
	wantRows(t, tt, "", "Arcade Mono", "Courier")
	lookKeys(tt, ui.KeyEnter)
	if got := f.Family(); got != "" {
		t.Fatalf("Enter after the list shrank picked %q", got)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("Enter after the list shrank ran %q", got)
	}
}

// Case N6: a family the system lacks keeps its name in the field, drawn in
// the default face, and the drop-down highlights no row for it.
func TestFontEditUnknownFamily(t *testing.T) {
	f := newFontEdit(fontFamilies).SetFamily("NoSuchFont")
	tt := fontTester(t, f)
	if got := f.Family(); got != "NoSuchFont" {
		t.Fatalf("Family is %q", got)
	}
	if v := fontField(tt).Value(); v != "NoSuchFont" {
		t.Fatalf("the field shows %q", v)
	}
	plain := NewTester(t, HBox(0).Add(Label("NoSuchFont")), 400, 500)
	plain.Settle()
	if got, want := textAt(t, tt, "NoSuchFont"), textAt(t, plain, "NoSuchFont"); !within(got.W, want.W, 0.5) || !within(got.H, want.H, 0.5) {
		t.Fatalf("the field draws NoSuchFont %vx%v, the default face %vx%v", got.W, got.H, want.W, want.H)
	}
	fontField(tt).Click()
	wantRows(t, tt, "", "Arcade Mono", "Arial", "Arial Black", "Barlow", "Candara", "Charter", "Courier", "Gamma")
	if got := focusedRows(tt); len(got) != 0 {
		t.Fatalf("the rows %q are highlighted for a family the system lacks", got)
	}
	lookKeys(tt, ui.KeyEscape)
	if got := f.Family(); got != "NoSuchFont" {
		t.Fatalf("after Escape Family is %q", got)
	}
}

// Case N7: Down opens, Down and Up move the highlight and stop at the ends,
// Enter picks and runs OnChange once, Escape closes and keeps the family;
// picking the family it has runs nothing.
func TestFontEditKeys(t *testing.T) {
	var log editLog
	f := newFontEdit(fontFamilies).OnChange(log.add("change"))
	tt := fontTester(t, f)
	fontField(tt).Focus()
	lookKeys(tt, ui.KeyDown)
	if !lookOpen(tt) || !searchBox(tt).Focused() {
		t.Fatalf("Down: open %v, the search box focused %v", lookOpen(tt), searchBox(tt).Focused())
	}
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyEnter) // Arial
	if lookOpen(tt) {
		t.Fatal("Enter did not close the drop-down")
	}
	if got := f.Family(); got != "Arial" {
		t.Fatalf("Down, Down, Enter picked %q, want Arial", got)
	}
	if v := fontField(tt).Value(); v != "Arial" {
		t.Fatalf("the field shows %q", v)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("a pick ran %q, want one change", got)
	}

	// The drop-down opens on the family: Up from Arial is Arcade Mono, and
	// Up stops there.
	log.reset()
	fontField(tt).Click()
	if got := focusedRows(tt); !slices.Equal(got, []string{"Arial"}) {
		t.Fatalf("on opening the highlight is on %q, want Arial", got)
	}
	lookKeys(tt, ui.KeyUp, ui.KeyUp, ui.KeyUp, ui.KeyEnter)
	if got := f.Family(); got != "Arcade Mono" {
		t.Fatalf("Up, Up, Up, Enter picked %q, want Arcade Mono", got)
	}
	// Down stops at the last.
	fontField(tt).Click()
	for range 12 {
		lookKeys(tt, ui.KeyDown)
	}
	lookKeys(tt, ui.KeyEnter)
	if got := f.Family(); got != "Gamma" {
		t.Fatalf("Down past the end, Enter picked %q, want Gamma", got)
	}
	// Escape closes and keeps the family.
	fontField(tt).Click()
	lookKeys(tt, ui.KeyUp, ui.KeyEscape)
	if lookOpen(tt) {
		t.Fatal("Escape did not close the drop-down")
	}
	if got := f.Family(); got != "Gamma" {
		t.Fatalf("after Escape Family is %q", got)
	}
	// Picking the family it has runs nothing.
	fontField(tt).Click()
	lookKeys(tt, ui.KeyEnter)
	if lookOpen(tt) {
		t.Fatal("Enter on the highlighted family did not close the drop-down")
	}
	if got := log.all(t); !slices.Equal(got, []string{"change", "change"}) {
		t.Fatalf("ran %q, want a change for Arcade Mono and one for Gamma", got)
	}

	// Enter with no row highlighted picks nothing and keeps it open.
	f2 := newFontEdit(fontFamilies)
	tt2 := fontTester(t, f2)
	fontField(tt2).Click()
	lookKeys(tt2, ui.KeyEnter)
	if !lookOpen(tt2) || f2.Family() != "" {
		t.Fatalf("Enter on no row: open %v, Family %q", lookOpen(tt2), f2.Family())
	}
}

// Case N7b: after Enter and after Escape the field has the keyboard, and
// Down opens the drop-down again.
func TestFontEditKeysKeepTheFocus(t *testing.T) {
	f := newFontEdit(fontFamilies)
	tt := fontTester(t, f)
	fontField(tt).Focus()
	lookKeys(tt, ui.KeyDown, ui.KeyDown, ui.KeyEnter) // Arcade Mono
	if !fontField(tt).Focused() {
		t.Fatal("after Enter the field has not the focus")
	}
	lookKeys(tt, ui.KeyDown)
	if !lookOpen(tt) {
		t.Fatal("Down after Enter did not open the drop-down")
	}
	lookKeys(tt, ui.KeyEscape)
	if lookOpen(tt) || !fontField(tt).Focused() {
		t.Fatalf("after Escape: open %v, the field focused %v", lookOpen(tt), fontField(tt).Focused())
	}
	lookKeys(tt, ui.KeyDown)
	if !lookOpen(tt) {
		t.Fatal("Down after Escape did not open the drop-down")
	}
	if got := f.Family(); got != "Arcade Mono" {
		t.Fatalf("Family is %q, want Arcade Mono", got)
	}
}

// Case N7c: the drop-down opens on the family whatever the case SetFamily
// gave it in, as the text system finds it; SetFamily's spelling stays.
func TestFontEditOpensOnTheFamilyWhateverTheCase(t *testing.T) {
	f := newFontEdit(fontFamilies).SetFamily("arial")
	tt := fontTester(t, f)
	fontField(tt).Click()
	if got := focusedRows(tt); !slices.Equal(got, []string{"Arial"}) {
		t.Fatalf("on opening the highlight is on %q, want Arial", got)
	}
	if got := f.Family(); got != "arial" {
		t.Fatalf("Family is %q, want arial as set", got)
	}
}

// Case N7d: a disabled FontEdit opens on neither a click nor Down.
func TestFontEditDisabled(t *testing.T) {
	var reads atomic.Int32
	f := newFontEdit(func() []ui.FontFamily { reads.Add(1); return fontFamilies() }).SetDisabled(true)
	tt := fontTester(t, f)
	r := fontField(tt).Bounds()
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Settle()
	tt.Key(0, ui.KeyDown)
	tt.Key(ui.Alt, ui.KeyDown)
	tt.Settle()
	if lookOpen(tt) || reads.Load() != 0 {
		t.Fatalf("a disabled FontEdit opened (%v) or read the families (%d)", lookOpen(tt), reads.Load())
	}
}

// Without a skin the frame looks as a TextBox: as high, edged with the
// outline, and with the accent while the field has the focus.
func TestFontEditLooksAsATextBox(t *testing.T) {
	calmOff()
	th := new(ui.Theme)
	keep := Custom(func(c *ui.Context) ui.Element { *th = *c.Theme(); return ui.Box(c) })
	box := TextBox().SetText("Arial").SetSize(300, 0)
	f := newFontEdit(fontFamilies).SetFamily("Arial").SetSize(300, 0)
	tt := NewTester(t, VBox(0).Add(keep, box, f), 400, 200)
	tt.Settle()
	boxH := tt.Find(ByRole(ui.RoleTextField)).Bounds().H
	in := fontField(tt)
	r := in.Bounds()
	if r.H+2 < boxH-0.5 || r.H+2 > boxH+0.5 {
		t.Fatalf("the FontEdit's frame is %v high (field %v and its edge), a TextBox %v", r.H+2, r.H, boxH)
	}
	edge := func() color.RGBA { return tt.Image().RGBAAt(0, int(r.Y+r.H/2)) }
	if c := edge(); !near(c, th.Outline().Over(th.FieldFill())) && !near(c, th.Outline().Over(th.Background)) {
		t.Fatalf("the frame's edge is %v, want the outline %v", c, th.Outline())
	}
	in.Focus()
	if c := edge(); !near(c, th.Accent) {
		t.Fatalf("focused, the frame's edge is %v, want the accent %v", c, th.Accent)
	}
}

// The frame takes the skin's Field look: a look of bg-chrome fills it with
// Chrome.
func TestFontEditFieldLook(t *testing.T) {
	useLooks(t, map[string]string{"Field": "bg-chrome"})
	tk := skinhook.Get(false)
	tt := NewTester(t, VBox(0).Add(newFontEdit(fontFamilies)), 400, 120)
	tt.Settle()
	r := fontField(tt).Bounds()
	if c := tt.Image().RGBAAt(int(r.X+4), int(r.Y+r.H/2)); !near(c, tk.Chrome) || tk.Chrome == tk.Background {
		t.Fatalf("the field fills with %v, want Chrome %v", c, tk.Chrome)
	}
}

// An ErrorProvider's mark sits at the frame's end: the field, its chevron
// included, stops before it.
func TestFontEditErrorMarkClearsTheChevron(t *testing.T) {
	f := newFontEdit(fontFamilies).SetFamily("Arial")
	tt := fontTester(t, f)
	ErrorProvider().SetError(f, "Wrong")
	tt.Settle()
	r := fontField(tt).Bounds()
	if r.X+r.W > 300-errorMarkSize-errorMarkInset {
		t.Fatalf("the field ends at %v, under the error mark at the frame's end (300)", r.X+r.W)
	}
}

// Case N8: Alt+Down, Enter and Space open the drop-down too.
func TestFontEditOpenKeys(t *testing.T) {
	keys := []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{ui.Alt, ui.KeyDown}, {0, ui.KeyEnter}, {0, ui.KeySpace}}
	for _, k := range keys {
		tt := fontTester(t, newFontEdit(fontFamilies))
		fontField(tt).Focus()
		tt.Tester.Key(k.mods, k.key)
		tt.Settle()
		if !lookOpen(tt) {
			t.Fatalf("%v %v did not open the drop-down", k.mods, k.key)
		}
	}
}

// Case N9: a click on a row picks it and closes the drop-down; a second
// click on the field closes it.
func TestFontEditClickRow(t *testing.T) {
	var log editLog
	f := newFontEdit(fontFamilies).OnChange(log.add("change"))
	tt := fontTester(t, f)
	fontField(tt).Click()
	tt.Find(ByRole(ui.RoleListItem).Label("Barlow")).Click()
	tt.Settle()
	if lookOpen(tt) {
		t.Fatal("a click on a row did not close the drop-down")
	}
	if got := f.Family(); got != "Barlow" {
		t.Fatalf("the click picked %q", got)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("a click ran %q, want one change", got)
	}
	fontField(tt).Click()
	fontField(tt).Click()
	if lookOpen(tt) {
		t.Fatal("a second click on the field did not close the drop-down")
	}
}

// Case N10: of 300 families, the drop-down builds only the rows in view,
// and opens with the family's row in view and highlighted.
func TestFontEditBuildsOnlyTheRowsShown(t *testing.T) {
	f := newFontEdit(manyFamilies).SetFamily("Fam250")
	tt := fontTester(t, f)
	fontField(tt).Click()
	rows := rowNames(tt)
	t.Logf("opening built %d rows", len(rows))
	if len(rows) == 0 || len(rows) > lookBuiltMax {
		t.Fatalf("opening built %d rows, want 1 to %d", len(rows), lookBuiltMax)
	}
	if a, ok := tt.Tester.AccessByRole(ui.RoleListItem, "Fam250"); !ok || a.Offscreen {
		t.Fatalf("Fam250 is not in view (found %v): %q", ok, rows)
	}
	if got := focusedRows(tt); !slices.Equal(got, []string{"Fam250"}) {
		t.Fatalf("the highlight is on %q, want Fam250", got)
	}
}

// Case N11: the drop-down closes when the FontEdit is hidden, and stays
// closed when it shows again.
func TestFontEditHideCloses(t *testing.T) {
	f := newFontEdit(fontFamilies)
	tt := fontTester(t, f)
	fontField(tt).Click()
	f.SetVisible(false)
	tt.Settle()
	f.SetVisible(true)
	tt.Settle()
	if lookOpen(tt) {
		t.Fatal("the drop-down shows again after a hide")
	}
}

// Case N12: right to left, the field's text is at its right, and the
// drop-down starts at the field's right.
func TestFontEditRightToLeft(t *testing.T) {
	f := newFontEdit(fontFamilies).SetFamily("Gamma")
	tt := fontTester(t, f)
	tt.SetDirection(ui.RightToLeft)
	tt.Settle()
	field := fontField(tt).Bounds()
	if r := textAt(t, tt, "Gamma"); r.X < field.X+field.W/2 {
		t.Fatalf("right to left the text is at %v, in the left half of the field %v", r, field)
	}
	fontField(tt).Click()
	list := tt.Find(ByRole(ui.RoleList)).Bounds()
	if !within(list.X+list.W, field.X+field.W, 12) {
		t.Fatalf("right to left the drop-down %v does not end at the field's right %v", list, field)
	}
}

// Case N12b: a field narrower than the drop-down: the drop-down starts at
// the field's start, its left, or its right right to left.
func TestFontEditNarrowRightToLeft(t *testing.T) {
	for _, rtl := range []bool{false, true} {
		calmOff()
		f := newFontEdit(fontFamilies)
		// 300 DIPs from the window's start: the drop-down has room on both
		// sides of the field.
		tt := NewTester(t, HBox(0).Add(Label("start").SetSize(300, 0), f.SetSize(160, 0)), 800, 500)
		tt.Settle() // in LTR first, as an app that switches its language
		if rtl {
			tt.SetDirection(ui.RightToLeft)
		}
		tt.Settle()
		field := fontField(tt).Bounds()
		fontField(tt).Click()
		panel := tt.Find(ByRole(ui.RolePopup)).Bounds()
		if panel.W <= field.W+1 {
			t.Fatalf("the drop-down %v is not wider than the field %v: the case does not hold", panel, field)
		}
		if !rtl && !within(panel.X, field.X, 3) || rtl && !within(panel.X+panel.W, field.X+field.W, 3) {
			t.Fatalf("right to left %v: the drop-down %v does not start at the field's start %v", rtl, panel, field)
		}
	}
}

func TestFontEditSetterDoesNotFire(t *testing.T) {
	var log editLog
	f := newFontEdit(fontFamilies).OnChange(log.add("change"))
	tt := fontTester(t, f)
	f.SetFamily("Arial")
	tt.Settle()
	if v := fontField(tt).Value(); v != "Arial" {
		t.Fatalf("the field shows %q, want Arial", v)
	}
	fontField(tt).Click()
	f.SetFamily("Gamma").SetFixedPitchOnly(true).SetFixedPitchOnly(false)
	tt.Settle()
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("setters ran %q", got)
	}
}

func TestFontEditFromGoroutine(t *testing.T) {
	f := newFontEdit(fontFamilies)
	tt := fontTester(t, f)
	fontField(tt).Click()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			f.SetFamily(fmt.Sprint("Font ", i)).SetFixedPitchOnly(i%2 == 0).OnChange(func() {})
			_ = f.Family()
		}
		f.SetFamily("Courier").SetFixedPitchOnly(true)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	if v := fontField(tt).Value(); v != "Courier" {
		t.Fatalf("the field shows %q, want Courier", v)
	}
	if lookOpen(tt) {
		fontField(tt).Click() // closes
	}
	fontField(tt).Click()
	wantRows(t, tt, "", "Arcade Mono", "Courier")
}

func TestFontEditBind(t *testing.T) {
	var mu sync.Mutex
	v := "Arial"
	get := func() string { mu.Lock(); defer mu.Unlock(); return v }
	f := Bind(newFontEdit(fontFamilies), get, func(x string) { mu.Lock(); v = x; mu.Unlock() })
	tt := fontTester(t, f)
	if s := fontField(tt).Value(); s != "Arial" {
		t.Fatalf("the field shows %q, not the source's Arial", s)
	}
	fontField(tt).Click()
	lookKeys(tt, ui.KeyDown, ui.KeyEnter) // Arial Black
	flushEvents(t)
	if s := get(); s != "Arial Black" {
		t.Fatalf("after a pick the source has %q", s)
	}
	mu.Lock()
	v = "Gamma"
	mu.Unlock()
	Reload(f)
	tt.Settle()
	if s := fontField(tt).Value(); s != "Gamma" {
		t.Fatalf("after Reload the field shows %q", s)
	}
}

func TestFontEditAllocNothingAtRest(t *testing.T) {
	calmOff()
	box := ui.NewTester(content(TextBox().SetText("Arial")), 400, 120)
	edit := ui.NewTester(content(newFontEdit(fontFamilies).SetFamily("Arial")), 400, 120)
	for range 10 {
		box.Frame()
		edit.Frame()
	}
	want, got := allocrun.Least(30, box.Frame), allocrun.Least(30, edit.Frame)
	if got > want {
		t.Fatalf("a frame of the FontEdit at rest allocates %v, more than a TextBox's %v", got, want)
	}
	t.Logf("a frame allocates %v, a TextBox's %v", got, want)
}
