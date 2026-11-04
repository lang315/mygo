package widget

import (
	"runtime"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// barApp is a form with a text box and a menu bar under it, and the items
// the bar ran.
type barApp struct {
	mu     sync.Mutex
	ran    []string
	bar    *MenuBarWidget
	box    *TextBoxWidget
	tt     *Tester
	form   *Form
	button int // clicks of the button &File
}

func (a *barApp) run(s string) func() {
	return func() { a.mu.Lock(); a.ran = append(a.ran, s); a.mu.Unlock() }
}

func (a *barApp) got() []string {
	<-posted()
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.ran)
}

func newBarApp(t *testing.T) *barApp {
	a := &barApp{}
	a.bar = MenuBar(
		MenuItem{Text: "&File", Children: []MenuItem{
			{Text: "&Open", OnClick: a.run("Open"), Shortcut: Shortcut{ui.Ctrl, ui.KeyO}},
			{Text: "Recent", Children: []MenuItem{{Text: "notes.txt", OnClick: a.run("notes.txt")}}},
			{},
			{Text: "Locked", Disabled: true, OnClick: a.run("Locked")},
			{Text: "E&xit", OnClick: a.run("Exit")},
		}},
		MenuItem{Text: "&Edit", Children: []MenuItem{
			{Text: "Undo", OnClick: a.run("Undo")},
			{Text: "Wrap", Checked: true, OnClick: a.run("Wrap")},
			{Text: "Delete all", Destructive: true, OnClick: a.run("Delete all")},
		}},
		MenuItem{Text: "&Help", Disabled: true, Children: []MenuItem{{Text: "About"}}},
	)
	a.box = TextBox()
	a.form = NewForm("bar", 600, 400)
	a.form.SetContent(VBox(4).Add(a.bar, a.box, Button("&File").OnClick(func() { a.button++ })))
	a.tt = TestForm(t, a.form)
	a.tt.Settle()
	return a
}

// open reports the items of the menus that show.
func (a *barApp) shows(label string) bool { return a.tt.Has(ByRole(ui.RoleMenuItem).Label(label)) }

func (a *barApp) title(label string) *Node {
	return a.tt.Find(ByRole(ui.RoleMenuItem).Label(label))
}

func TestMenuBarIsAMenuBarOfMenuItemsWithoutTheAmpersands(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	if !a.tt.Has(ByRole(ui.RoleMenuBar)) {
		t.Fatal("no menu bar for assistive technology")
	}
	for _, s := range []string{"File", "Edit", "Help"} {
		if !a.shows(s) {
			t.Errorf("no menu item %q in the bar: %q", s, a.tt.Texts())
		}
	}
	if a.tt.HasText("&File") || a.shows("Open") {
		t.Errorf("texts %q", a.tt.Texts())
	}
	if !a.title("Help").Disabled() {
		t.Error("Help is not disabled")
	}
}

func TestMenuBarClickOpensAMenuAndChoosesAnItem(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	a.title("File").Click()
	if !a.tt.Has(ByRole(ui.RoleMenu)) || !a.shows("Open") || !a.shows("Exit") {
		t.Fatalf("File did not open its menu: %q", a.tt.Texts())
	}
	want := "Ctrl+O"
	if runtime.GOOS == "darwin" {
		want = "⌃O"
	}
	if !a.tt.HasText(want) {
		t.Errorf("no shortcut %q: %q", want, a.tt.Texts())
	}
	if !a.title("Locked").Disabled() {
		t.Error("Locked is not disabled")
	}
	a.title("Exit").Click()
	if got := a.got(); !slices.Equal(got, []string{"Exit"}) || a.shows("Open") {
		t.Errorf("ran %q, menu still shows %v", got, a.shows("Open"))
	}
	// The title clicked again closes its menu.
	a.title("File").Click()
	a.title("File").Click()
	if a.shows("Open") {
		t.Error("a second click on File left its menu open")
	}
	// A disabled title opens nothing.
	a.tt.Click("Help")
	a.tt.Settle()
	if a.shows("About") {
		t.Error("the disabled Help opened")
	}
}

func TestMenuBarHoverGoesFromOneOpenMenuToAnother(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	edit := a.title("Edit").Bounds()
	// No menu open: hovering opens nothing.
	a.tt.Move(edit.X+edit.W/2, edit.Y+edit.H/2)
	a.tt.Settle()
	if a.shows("Undo") {
		t.Fatal("hovering Edit opened it with no menu open")
	}
	a.title("File").Click()
	a.tt.Move(edit.X+edit.W/2, edit.Y+edit.H/2)
	a.tt.Settle()
	if !a.shows("Undo") || a.shows("Open") {
		t.Fatalf("hovering Edit with File open: %q", a.tt.Texts())
	}
	// The pointer resting there does not take a menu the keys went to back.
	a.tt.Key(0, ui.KeyLeft)
	a.tt.Settle()
	if !a.shows("Open") || a.shows("Undo") {
		t.Fatalf("Left from Edit with the pointer on it: %q", a.tt.Texts())
	}
}

func TestMenuBarKeysMoveAmongTheMenus(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	a.title("File").Click()
	steps := []struct {
		key  ui.Key
		want string // an item of the menu that shows after
	}{
		{ui.KeyRight, "Undo"}, // Edit, Help disabled skipped next
		{ui.KeyRight, "Open"}, // round, past Help
		{ui.KeyLeft, "Undo"},  // round the other way, past Help
		{ui.KeyLeft, "Open"},
	}
	for i, s := range steps {
		a.tt.Key(0, s.key)
		a.tt.Settle()
		if !a.shows(s.want) {
			t.Fatalf("step %d: %q does not show: %q", i, s.want, a.tt.Texts())
		}
	}
	// Opened by the keys, a menu starts on its first item.
	a.tt.Key(0, ui.KeyEnter)
	if got := a.got(); !slices.Equal(got, []string{"Open"}) {
		t.Fatalf("Enter ran %q, want Open", got)
	}
	// Into a submenu and out, then Right on a plain item goes to Edit.
	a.title("File").Click()
	a.tt.Key(0, ui.KeyDown)
	a.tt.Key(0, ui.KeyDown)
	a.tt.Key(0, ui.KeyRight)
	a.tt.Settle()
	if !a.shows("notes.txt") {
		t.Fatalf("Right did not open Recent: %q", a.tt.Texts())
	}
	a.tt.Key(0, ui.KeyRight)
	a.tt.Settle()
	if !a.shows("Undo") || a.shows("notes.txt") {
		t.Fatalf("Right on notes.txt: %q", a.tt.Texts())
	}
	a.tt.Key(0, ui.KeyEscape)
	a.tt.Settle()
	if a.shows("Undo") {
		t.Error("Escape left Edit open")
	}
}

func TestMenuBarAltLetterOpensAMenuAndBeatsAControl(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	a.tt.Key(ui.Alt, ui.KeyE)
	a.tt.Settle()
	if !a.shows("Undo") {
		t.Fatalf("Alt+E did not open Edit: %q", a.tt.Texts())
	}
	a.tt.Key(0, ui.KeyEscape)
	a.tt.Key(ui.Alt, ui.KeyF)
	a.tt.Settle()
	if !a.shows("Open") || a.button != 0 {
		t.Fatalf("Alt+F: File open %v, button clicked %d times", a.shows("Open"), a.button)
	}
	a.tt.Key(0, ui.KeyEnter) // on the first item
	if got := a.got(); !slices.Equal(got, []string{"Open"}) {
		t.Errorf("ran %q", got)
	}
	mu.Lock()
	keys := a.form.menuBarKeys()
	mu.Unlock()
	if !keys['f'] || !keys['e'] || keys['h'] {
		t.Errorf("menu bar keys %v; the disabled Help's is not one", keys)
	}
	// With a menu open, another title's key opens that one instead.
	a.title("File").Click()
	a.tt.Key(ui.Alt, ui.KeyE)
	a.tt.Settle()
	if !a.shows("Undo") || a.shows("Open") {
		t.Fatalf("Alt+E with File open: %q", a.tt.Texts())
	}
	a.tt.Key(0, ui.KeyEscape)
	a.tt.Settle()
	// A disabled title's key opens nothing.
	a.tt.Key(ui.Alt, ui.KeyH)
	a.tt.Settle()
	if a.shows("About") {
		t.Error("Alt+H opened the disabled Help")
	}
}

func TestMenuBarUnderlinesItsKeysWhileAltIsHeld(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	r := a.title("File").Bounds()
	title := func() []byte {
		img := a.tt.Image()
		var px []byte
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			px = append(px, img.Pix[img.PixOffset(int(r.X), y):img.PixOffset(int(r.X+r.W), y)]...)
		}
		return px
	}
	plain := title()
	a.tt.HoldModifiers(ui.Alt)
	a.tt.Frame()
	if slices.Equal(plain, title()) {
		t.Error("File looks the same while Alt is held: its F is not underlined")
	}
	a.tt.HoldModifiers(0)
	a.tt.Frame()
	if !slices.Equal(plain, title()) {
		t.Error("File does not look as before once Alt is let go")
	}
}

func TestMenuBarHasNoAccessKeysOnMac(t *testing.T) {
	withMac(t, true)
	a := newBarApp(t)
	a.tt.Key(ui.Alt, ui.KeyF)
	a.tt.Settle()
	if a.shows("Open") {
		t.Error("Alt+F opened File on macOS")
	}
	if !a.shows("File") || a.tt.HasText("&File") {
		t.Errorf("texts %q", a.tt.Texts())
	}
	a.title("File").Click()
	if !a.shows("Open") {
		t.Error("a click did not open File on macOS")
	}
}

func TestMenuBarGivesTheFocusBack(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	focused := func() bool { return a.tt.Find(ByRole(ui.RoleTextField)).Focused() }
	opens := map[string]func(){
		"click": func() { a.title("File").Click() },
		"hover": func() {
			a.title("File").Click()
			e := a.title("Edit").Bounds()
			a.tt.Move(e.X+e.W/2, e.Y+e.H/2)
			a.tt.Settle()
		},
		"arrows": func() {
			a.title("File").Click()
			a.tt.Key(0, ui.KeyRight)
			a.tt.Key(0, ui.KeyRight)
			a.tt.Key(0, ui.KeyLeft)
			a.tt.Settle()
		},
		"alt in a menu": func() {
			a.title("File").Click()
			a.tt.Key(ui.Alt, ui.KeyE)
			a.tt.Settle()
		},
	}
	if runtime.GOOS != "darwin" {
		// On macOS a text box with the focus types Option and a letter: the
		// form never sees the key (see mnemonics).
		opens["alt"] = func() { a.tt.Key(ui.Alt, ui.KeyF); a.tt.Settle() }
	}
	a.tt.Find(ByRole(ui.RoleTextField)).Click()
	if !focused() {
		t.Fatal("the text box did not take the focus")
	}
	for name, open := range opens {
		open()
		if focused() {
			t.Fatalf("%s: the text box kept the focus with a menu open", name)
		}
		a.tt.Key(0, ui.KeyEscape)
		a.tt.Settle()
		if !focused() {
			t.Errorf("%s: the focus did not go back to the text box", name)
		}
	}
}

func TestMenuBarCheckAndSubmenuItemsForAssistiveTechnology(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	a.title("Edit").Click()
	wrap := a.tt.Find(ByRole(ui.RoleMenuItemCheckBox).Label("Wrap"))
	if !wrap.Checked() {
		t.Error("Wrap is not a checked check box item")
	}
	wrap.Click()
	if got := a.got(); !slices.Equal(got, []string{"Wrap"}) {
		t.Errorf("ran %q", got)
	}
	a.title("File").Click()
	a.title("Recent").Click()
	a.title("notes.txt").Click()
	if got := a.got(); !slices.Equal(got, []string{"Wrap", "notes.txt"}) {
		t.Errorf("ran %q", got)
	}
}

func TestMenuBarSetMenusChangesTheTitles(t *testing.T) {
	withMac(t, false)
	a := newBarApp(t)
	a.title("File").Click()
	a.bar.SetMenus(MenuItem{Text: "&View", Children: []MenuItem{{Text: "Zoom"}}})
	a.tt.Settle()
	if a.shows("File") || !a.shows("View") || a.shows("Open") {
		t.Fatalf("texts %q", a.tt.Texts())
	}
	a.tt.Key(ui.Alt, ui.KeyV)
	a.tt.Settle()
	if !a.shows("Zoom") {
		t.Errorf("Alt+V did not open View: %q", a.tt.Texts())
	}
}
