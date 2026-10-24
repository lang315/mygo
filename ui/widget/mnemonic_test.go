package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestParseMnemonic(t *testing.T) {
	for _, c := range []struct {
		in, display string
		key         rune
		at          int
	}{
		{"&Save", "Save", 's', 0},
		{"Sa&ve", "Save", 'v', 2},
		{"Fish && &Chips", "Fish & Chips", 'c', 7},
		{"Plain", "Plain", 0, -1},
		{"A &&B", "A &B", 0, -1},
		{"Trailing &", "Trailing &", 0, -1},
		{"&Đức", "Đức", 'd', 0},
		{"&Ẩn", "Ẩn", 'a', 0},
		{"&Ước", "Ước", 'u', 0},
		{"&@x", "@x", 0, -1},
		{"&1st", "1st", '1', 0},
	} {
		display, key, at := parseMnemonic(c.in)
		if display != c.display || key != c.key || at != c.at {
			t.Errorf("%q: %q %q %d, want %q %q %d", c.in, display, string(key), at, c.display, string(c.key), c.at)
		}
	}
}

func withMac(t *testing.T, on bool) {
	old := macMenus
	macMenus = on
	t.Cleanup(func() { macMenus = old })
}

func TestButtonMnemonicShowsTheLabelWithoutTheAmpersandAndAltKeyClicks(t *testing.T) {
	withMac(t, false)
	clicks := make(chan struct{}, 4)
	b := Button("&Save").OnClick(func() { clicks <- struct{}{} })
	f := NewForm("m", 400, 200)
	f.SetContent(VBox(4).Add(b, Button("Other")))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("Save")) || tt.Has(ByLabel("&Save")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	received(t, clicks)
	b.SetDisabled(true)
	tt.Settle()
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	select {
	case <-clicks:
		t.Error("a disabled button was clicked by its mnemonic")
	default:
	}
}

func TestFieldsLabelMnemonicFocusesItsControl(t *testing.T) {
	withMac(t, false)
	// A fresh form for each key: a focused text box takes Alt+letter itself on
	// the macOS this may run on, where mnemonics are off anyway.
	for _, c := range []struct {
		key  ui.Key
		want int
	}{{ui.KeyN, 0}, {ui.KeyM, 1}} {
		f := NewForm("m", 400, 200)
		f.SetContent(Fields().Row("&Name", TextBox()).Row("E&mail", TextBox()))
		tt := TestForm(t, f)
		tt.Settle()
		tt.Key(ui.Alt, c.key)
		tt.Settle()
		if !tt.Find(ByRole(ui.RoleTextField).Nth(c.want)).Focused() {
			t.Errorf("Alt+key %v did not focus text box %d", c.key, c.want)
		}
	}
}

// On macOS the native menu bar owns Alt+key: the label loses the ampersand and
// nothing else happens.
func TestMnemonicsAreOffOnMac(t *testing.T) {
	withMac(t, true)
	clicks := make(chan struct{}, 4)
	f := NewForm("m", 400, 200)
	f.SetContent(VBox(4).Add(Button("&Save").OnClick(func() { clicks <- struct{}{} })))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("Save")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	select {
	case <-clicks:
		t.Error("Alt+S clicked on macOS")
	default:
	}
}

func TestDoubleAmpersandIsALiteralOne(t *testing.T) {
	withMac(t, false)
	tt := NewTester(t, Button("Fish && Chips"), 300, 100)
	if !tt.Has(ByLabel("Fish & Chips")) {
		t.Errorf("labels %q", tt.Texts())
	}
}

func TestStackedFieldsLabelMnemonicFocusesItsControl(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 200)
	f.SetContent(Fields().Stacked().Row("&Name", TextBox()).Row("E&mail", TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	tt.Key(ui.Alt, ui.KeyM)
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
		t.Error("Alt+M did not focus the Email box")
	}
}

func TestFieldsMnemonicIsOffOnMac(t *testing.T) {
	withMac(t, true)
	f := NewForm("m", 400, 200)
	f.SetContent(Fields().Row("&Name", TextBox()).Row("E&mail", TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("Name")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyM)
	tt.Settle()
	if tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
		t.Error("Alt+M focused a control on macOS")
	}
}

func TestAccentedMnemonicsAnswerToTheirBaseLetter(t *testing.T) {
	withMac(t, false)
	var d, a, u int
	f := NewForm("m", 400, 300)
	f.SetContent(VBox(4).Add(Button("&Đóng").OnClick(func() { d++ }), Button("&Ẩn").OnClick(func() { a++ }), Button("&Ước").OnClick(func() { u++ })))
	tt := TestForm(t, f)
	tt.Settle()
	for _, k := range []ui.Key{ui.KeyD, ui.KeyA, ui.KeyU} {
		tt.Key(ui.Alt, k)
		tt.Settle()
	}
	<-posted()
	if d != 1 || a != 1 || u != 1 {
		t.Errorf("Đóng %d Ẩn %d Ước %d, want 1 each", d, a, u)
	}
}

func TestMnemonicOfAButtonInADisabledContainerDoesNothing(t *testing.T) {
	withMac(t, false)
	n := 0
	f := NewForm("m", 400, 200)
	panel := VBox(4).Add(Button("&Save").OnClick(func() { n++ }))
	f.SetContent(VBox(4).Add(panel))
	tt := TestForm(t, f)
	panel.SetDisabled(true)
	tt.Settle()
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	<-posted()
	if n != 0 {
		t.Errorf("a button in a disabled panel was clicked %d times", n)
	}
}

func TestFormShortcutBeatsAMnemonic(t *testing.T) {
	withMac(t, false)
	btn, sc := 0, 0
	f := NewForm("m", 400, 200)
	f.SetContent(Button("&Save").OnClick(func() { btn++ }))
	f.Shortcut(ui.Alt, ui.KeyS, func() { sc++ })
	tt := TestForm(t, f)
	tt.Settle()
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	<-posted()
	if sc != 1 || btn != 0 {
		t.Errorf("shortcut %d button %d, want the shortcut alone", sc, btn)
	}
}

func TestSameMnemonicTwiceMovesTheFocusAndClicksNothing(t *testing.T) {
	withMac(t, false)
	a, b := 0, 0
	f := NewForm("m", 400, 200)
	f.SetContent(VBox(4).Add(Button("&Save").OnClick(func() { a++ }), Button("&Search").OnClick(func() { b++ })))
	tt := TestForm(t, f)
	tt.Settle()
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	<-posted()
	tt.Settle()
	if !tt.Find(ByLabel("Save")).Focused() {
		t.Error("the first Alt+S did not focus the first button")
	}
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	<-posted()
	tt.Settle()
	if !tt.Find(ByLabel("Search")).Focused() || a+b != 0 {
		t.Errorf("second Alt+S: search focused %v, clicks %d %d", tt.Find(ByLabel("Search")).Focused(), a, b)
	}
}

func TestAltShiftAnswersLikeAlt(t *testing.T) {
	withMac(t, false)
	n := 0
	f := NewForm("m", 400, 200)
	f.SetContent(Button("&Save").OnClick(func() { n++ }))
	tt := TestForm(t, f)
	tt.Settle()
	tt.Key(ui.Alt|ui.Shift, ui.KeyS)
	tt.Settle()
	<-posted()
	if n != 1 {
		t.Errorf("Alt+Shift+S clicked %d times", n)
	}
}

func TestCheckBoxAndRadioMnemonics(t *testing.T) {
	withMac(t, false)
	changed := make(chan struct{}, 4)
	cb := CheckBox("&Remember").OnChange(func() { changed <- struct{}{} })
	r1, r2 := RadioButton("&One"), RadioButton("T&wo")
	f := NewForm("m", 400, 300)
	f.SetContent(VBox(4).Add(cb, r1, r2))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("Remember")) || !tt.Has(ByLabel("Two")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyR)
	tt.Settle()
	received(t, changed)
	if !cb.Checked() {
		t.Error("Alt+R did not check the box")
	}
	tt.Key(ui.Alt, ui.KeyW)
	tt.Settle()
	<-posted()
	tt.Settle()
	if !r2.Checked() || r1.Checked() {
		t.Errorf("Alt+W: radio one %v two %v", r1.Checked(), r2.Checked())
	}
}

// Without a form building it (widget.View, NewTester) nothing answers, and
// nothing is underlined.
func TestMnemonicsAreDeadAndNotUnderlinedWithoutAForm(t *testing.T) {
	withMac(t, false)
	n := 0
	tt := NewTester(t, Button("&Save").OnClick(func() { n++ }), 300, 100)
	tt.Frame()
	if !tt.Has(ByLabel("Save")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyS)
	tt.Settle()
	if n != 0 {
		t.Errorf("clicked %d times", n)
	}
}

// The menu bar's access keys come before the controls': Alt+F opens the File
// menu, the button "&File" is not clicked, and a key the bar does not use is
// still the control's.
func TestMenuBarAccessKeysBeatControlMnemonics(t *testing.T) {
	withMac(t, false)
	for name, setup := range map[string]func(*Form){
		"menu bar":   func(f *Form) { f.SetMenuBar(MenuItem{Text: "&File"}, MenuItem{Text: "&Edit"}) },
		"menu strip": func(f *Form) { f.MenuStrip(NewMenuStrip(StripItem("&File"), StripItem("&Edit"))) },
	} {
		file, save := 0, 0
		f := NewForm("m", 400, 200)
		f.SetContent(VBox(4).Add(Button("&File").OnClick(func() { file++ }), Button("&Save").OnClick(func() { save++ })))
		setup(f)
		tt := TestForm(t, f)
		tt.Settle()
		tt.Key(ui.Alt, ui.KeyF)
		tt.Settle()
		tt.Key(ui.Alt, ui.KeyS)
		tt.Settle()
		<-posted()
		if file != 0 || save != 1 {
			t.Errorf("%s: File button %d times, Save button %d times; want 0 and 1", name, file, save)
		}
		mu.Lock()
		keys := f.menuBarKeys()
		mu.Unlock()
		if !keys['f'] || !keys['e'] || keys['s'] {
			t.Errorf("%s: menu bar keys %v", name, keys)
		}
	}
}

func TestLabelMnemonicFocusesTheNextControlThatTakesTheFocus(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 200)
	f.SetContent(VBox(4).Add(Label("&Name"), Label("a plain label"), TextBox(), TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("Name")) || tt.Has(ByLabel("&Name")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyN)
	tt.Settle()
	<-posted()
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleTextField).Nth(0)).Focused() || tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
		t.Error("Alt+N did not focus the text box after the label")
	}
}

func TestLabelMnemonicSkipsADisabledControl(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 200)
	off := TextBox()
	off.SetDisabled(true)
	f.SetContent(VBox(4).Add(Label("&Name"), off, TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	tt.Key(ui.Alt, ui.KeyN)
	tt.Settle()
	<-posted()
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
		t.Error("Alt+N did not skip the disabled box")
	}
}

func TestLabelUseMnemonicFalseShowsTheTextAsItIs(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 200)
	f.SetContent(VBox(4).Add(Label("Q&A").SetUseMnemonic(false), TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("Q&A")) {
		t.Fatalf("labels %q", tt.Texts())
	}
	tt.Key(ui.Alt, ui.KeyA)
	tt.Settle()
	if tt.Find(ByRole(ui.RoleTextField)).Focused() {
		t.Error("a label that does not use mnemonics answered one")
	}
}

// A literal ampersand in a Fields label is not read as a mnemonic twice.
func TestFieldsLabelWithALiteralAmpersandShowsIt(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 200)
	f.SetContent(Fields().Row("R&&D", TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	if !tt.Has(ByLabel("R&D")) {
		t.Errorf("labels %q", tt.Texts())
	}
}
