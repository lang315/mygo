package main

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// visit opens the tab, and the item of a Sidebar in it, that the path names.
func visit(tt *widget.Tester, path ...string) {
	for _, p := range path {
		tt.FindAll(widget.ByLabel(p))[0].Click()
		tt.Settle()
	}
}

// Each row of docs/ui/winforms.md without a tour test of its own has its
// entry on a tab of the tour, and what it shows is there.
func TestTourHasAnEntryForEachControl(t *testing.T) {
	for _, e := range []struct {
		row   string
		path  []string
		shows []string
	}{
		{"Button, MessageBox, Open/SaveFileDialog, FolderBrowserDialog", []string{"Dialogs"}, []string{"Alert", "Confirm", "Input text", "Open file", "Save file", "Select folder", "Custom dialog"}},
		{"MenuButton, ContextMenuStrip", []string{"Dialogs"}, []string{"Sort by", "Right-click me"}},
		{"CheckBox, Toggle, Switch, TextBox, MultilineTextBox", []string{"Inputs"}, []string{"Remember me", "Bold", "Notifications", "Count in the background"}},
		{"NumericUpDown, ProgressBar, RadioButton, RadioGroup", []string{"Choices"}, []string{"Red", "Green", "Small", "Large", "30%"}},
		{"Fields, VBox, HBox", []string{"Layout"}, []string{"Boxes", "Name", "Email address", "Country", "Notes"}},
		{"GroupBox, Grid", []string{"Containers", "GroupBox and grid"}, []string{"Frame", "C spans two"}},
		{"Card", []string{"Containers", "Card"}, []string{"Card title", "Card body"}},
		{"FlowPanel", []string{"Containers", "FlowPanel"}, []string{"chip 0", "chip 11"}},
		{"ScrollPanel", []string{"Containers", "ScrollPanel"}, []string{"Row 0"}},
		{"Sidebar, TabControl", []string{"Containers"}, []string{"Card", "FlowPanel", "ScrollPanel", "SplitContainer"}},
		{"PictureBox, Label", []string{"Extras"}, []string{"Screen size", "With an icon"}},
		{"ImageList, UserControl, LinkLabel", []string{"Components"}, []string{"ImageList", "Tomato", "Sky", "No icon", "Gauge 0%", "The docs"}},
		{"NotifyIcon", []string{"App shell"}, []string{"Tray icon"}},
	} {
		tt := widget.NewTester(t, tour(nil), 1100, 1800)
		visit(tt, e.path...)
		for _, s := range e.shows {
			if !tt.Has(widget.ByLabel(s)) {
				t.Errorf("%s: %q is not on %v: %q", e.row, s, e.path, tt.Texts())
			}
		}
	}
}

func TestTourShellHasItsMenuToolAndStatusStrips(t *testing.T) {
	f := widget.NewForm("tour", 1100, 800)
	appShell(f)
	f.SetContent(tour(f))
	tt := widget.TestForm(t, f)
	for _, s := range []string{"Plus", "Bold", "Ready"} {
		if !tt.Has(widget.ByLabel(s)) {
			t.Errorf("no %q among %q", s, tt.Texts())
		}
	}
	tt.Find(widget.ByLabel("Plus")).Click()
	tt.Settle()
	tt.Find(widget.ByLabel("Bold")).Click()
	if !tt.Has(widget.ByLabel("Bold: true")) {
		t.Errorf("the Bold tool did not tell the status strip: %q", tt.Texts())
	}
}

func TestTourComponentsGaugeMoves(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Components")
	tt.Find(widget.ByLabel("More")).Click()
	tt.Settle()
	if !tt.Has(widget.ByLabel("Gauge 10%")) {
		t.Errorf("More did not move the gauge: %q", tt.Texts())
	}
}

// The mnemonics of the Components tab show their text without the "&" (the
// keys themselves are tested in ui/widget).
func TestTourMnemonics(t *testing.T) {
	f := widget.NewForm("tour", 1100, 1800)
	f.SetContent(tour(f))
	tt := widget.TestForm(t, f)
	tt.Find(widget.ByLabel("Components")).Click()
	tt.Settle()
	for _, s := range []string{"Save", "Close", "Name", "Email", "City"} {
		if !tt.Has(widget.ByLabel(s)) || tt.Has(widget.ByLabel("&"+s)) {
			t.Errorf("%q is missing or shows its ampersand: %q", s, tt.Texts())
		}
	}
}

// The rows of the ImageList demo show their image, 24 DIPs square, and the
// row with no key none.
func TestTourImageListRowsShowTheirImages(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	visit(tt, "Components")
	red := func(label string) int {
		r := tt.Find(widget.ByRole(ui.RoleListItem).Label(label)).Bounds()
		img := tt.Image()
		n := 0
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			for x := int(r.X) - 30; x < int(r.X+r.W); x++ {
				if x < 0 {
					continue
				}
				if c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA); c.R > 200 && c.G < 120 && c.B < 100 {
					n++
				}
			}
		}
		return n
	}
	if n := red("Tomato"); n < 24*24 {
		t.Errorf("the Tomato row has %d red pixels, want at least %d", n, 24*24)
	}
	if n := red("No icon"); n != 0 {
		t.Errorf("the row with no key has %d red pixels", n)
	}
}
