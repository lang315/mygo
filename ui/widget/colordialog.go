package widget

import (
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// ColorDialogWidget asks the user for a color, as WinForms' ColorDialog:
// a color picker (a square of saturation and brightness, sliders of hue and
// opacity, the color in hex, and swatches of common colors) with OK and
// Cancel. It is drawn by ui, not the system's.
type ColorDialogWidget struct {
	color  ui.Color // under mu
	title  string
	chosen *chosen // OK's answer, between build and run
}

// ColorDialog makes a color dialog holding black.
func ColorDialog() *ColorDialogWidget {
	return &ColorDialogWidget{color: ui.RGB(0, 0, 0), title: "Color"}
}

// SetColor sets the color the dialog starts from.
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

// SetTitle sets the dialog's title.
func (d *ColorDialogWidget) SetTitle(s string) *ColorDialogWidget {
	mu.Lock()
	d.title = s
	mu.Unlock()
	return d
}

// ShowDialog shows the dialog over parent (nil for none) and returns true if
// the user chose OK, which sets Color; Cancel, Escape and closing the window
// return false and leave Color alone. It blocks as Dialog.Run does.
func (d *ColorDialogWidget) ShowDialog(parent *Form) bool { return d.run(d.build(parent)) }

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

// build makes the dialog and the function reading its answer.
func (d *ColorDialogWidget) build(parent *Form) *Dialog {
	mu.Lock()
	start, title := d.color, d.title
	mu.Unlock()
	pick := &colorPick{}
	pick.value.set(start)
	pick.init(pick)
	var dlg *Dialog
	bar := ButtonBox().Add(
		Button(l10n.Cancel()).OnClick(func() { dlg.Close() }),
		Button(l10n.Ok()).Suggested().OnClick(func() {
			c := pick.current()
			mu.Lock()
			d.chosen = &chosen{c}
			mu.Unlock()
			dlg.Close()
		}),
	)
	dlg = NewDialog(parent, title, "", 320, 520, VBox(12).Pack(pick, true, true).Add(bar))
	mu.Lock()
	d.chosen = nil
	mu.Unlock()
	return dlg
}

type chosen struct{ c ui.Color }

// run runs the dialog and applies the answer.
func (d *ColorDialogWidget) run(dlg *Dialog) bool {
	dlg.Run()
	mu.Lock()
	defer mu.Unlock()
	if d.chosen == nil {
		return false
	}
	d.color = d.chosen.c
	d.chosen = nil
	return true
}
