package widget

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
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
	searchBox(tt).Type("aR")
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
