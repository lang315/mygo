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
		{"&Đức", "Đức", 'đ', 0},
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
