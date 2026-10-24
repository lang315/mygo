package widget

import (
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// ColorDialogWidget asks the user for a color, as WinForms' ColorDialog:
// a color picker (a square of saturation and brightness, sliders of hue and
// opacity, the color in hex, and swatches of common colors) with OK and
// Cancel. It is drawn by ui, not the system's. Like WinForms' colors, the
// result has no transparency: the opacity slider and an alpha in the hex are
// ignored, and Color is opaque.
type ColorDialogWidget struct {
	color ui.Color // under mu
	title string
}

// ColorDialog makes a color dialog holding black.
func ColorDialog() *ColorDialogWidget {
	return &ColorDialogWidget{color: ui.RGB(0, 0, 0), title: "Color"}
}

// SetColor sets the color the next ShowDialog starts from. It does not
// change a dialog that is open.
func (d *ColorDialogWidget) SetColor(c ui.Color) *ColorDialogWidget {
	mu.Lock()
	d.color = c
	mu.Unlock()
	return d
}

// Color returns the color chosen by the last ShowDialog that answered OK,
// or the one SetColor gave.
func (d *ColorDialogWidget) Color() ui.Color {
	mu.Lock()
	defer mu.Unlock()
	return d.color
}

// SetTitle sets the title of the next ShowDialog.
func (d *ColorDialogWidget) SetTitle(s string) *ColorDialogWidget {
	mu.Lock()
	d.title = s
	mu.Unlock()
	return d
}

// ShowDialog shows the dialog over parent (nil for none) and returns true if
// the user chose OK, which sets Color; Cancel, Escape and closing the window
// return false and leave Color alone. It blocks as Dialog.Run does, and
// panics on the main thread.
func (d *ColorDialogWidget) ShowDialog(parent *Form) bool {
	notMain("ColorDialog.ShowDialog")
	_, run := d.build(parent)
	return run()
}

// colorPick is the picker inside the dialog; it holds the color being
// edited until OK.
type colorPick struct {
	common[colorPick]
	value mirror[ui.Color]
}

func (p *colorPick) build(c *ui.Context) ui.Element {
	v, _ := p.value.bind()
	return ui.ColorPicker(c, v).OnChange(onChange(&p.value, nil))
}

func (p *colorPick) current() ui.Color {
	mu.Lock()
	defer mu.Unlock()
	return p.value.value
}

// build makes the dialog, and the function that runs it and applies the
// answer.
func (d *ColorDialogWidget) build(parent *Form) (*Dialog, func() bool) {
	mu.Lock()
	start, title := d.color, d.title
	mu.Unlock()
	pick := &colorPick{}
	pick.value.set(start)
	pick.init(pick)
	var dlg *Dialog
	var picked *ui.Color // OK's answer
	bar := ButtonBox().Add(
		Button(l10n.Cancel()).OnClick(func() { dlg.Close() }),
		Button(l10n.Ok()).Suggested().OnClick(func() {
			c := pick.current()
			c.A = 255
			mu.Lock()
			picked = &c
			mu.Unlock()
			dlg.Close()
		}),
	)
	dlg = NewDialog(parent, title, "", 320, 520, VBox(12).Pack(pick, true, true).Add(bar))
	return dlg, func() bool {
		dlg.Run()
		mu.Lock()
		defer mu.Unlock()
		if picked == nil {
			return false
		}
		d.color = *picked
		return true
	}
}
