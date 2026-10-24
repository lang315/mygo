package widget

import (
	"strings"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// FontDialogWidget asks the user for a font, as WinForms' FontDialog: lists
// of families and styles, a size, and a sample, with OK and Cancel. It is
// drawn by ui, not the system's.
type FontDialogWidget struct {
	font             ui.Font // under mu
	title            string
	minSize, maxSize float32
	fixedPitch       bool
	sample           string
	families         func() []ui.FontFamily
	familyList       *ListBoxWidget[ui.FontFamily, string] // the last dialog's, for tests
}

// FontDialog makes a font dialog holding the interface font at 14 DIPs, as
// ui.Font{} is.
func FontDialog() *FontDialogWidget {
	return &FontDialogWidget{title: l10n.FontDialogTitle(), minSize: 1, maxSize: 999, sample: "AaBbYyZz", families: ui.FontFamilies}
}

// SetFont sets the font the next ShowDialog starts from. It does not change
// a dialog that is open.
func (d *FontDialogWidget) SetFont(f ui.Font) *FontDialogWidget {
	mu.Lock()
	d.font = f
	mu.Unlock()
	return d
}

// Font returns the font chosen by the last ShowDialog that answered OK, or
// the one SetFont gave. The dialog sets its Family, Size, Weight (400 or 700,
// kept as it was when the style did not change) and Italic; the rest is
// what SetFont gave.
func (d *FontDialogWidget) Font() ui.Font {
	mu.Lock()
	defer mu.Unlock()
	return d.font
}

// SetTitle sets the title of the next ShowDialog.
func (d *FontDialogWidget) SetTitle(s string) *FontDialogWidget {
	mu.Lock()
	d.title = s
	mu.Unlock()
	return d
}

// SetMinSize and SetMaxSize bound the sizes, in DIPs, that the dialog takes,
// as WinForms' MinSize and MaxSize (there in points); 1 and 999 by default.
// A size typed outside them is not taken: the size stays what it was. As in
// WinForms, setting one past the other moves the other to match.
func (d *FontDialogWidget) SetMinSize(v float32) *FontDialogWidget {
	mu.Lock()
	d.minSize, d.maxSize = v, max(d.maxSize, v)
	mu.Unlock()
	return d
}

// SetMaxSize: see SetMinSize.
func (d *FontDialogWidget) SetMaxSize(v float32) *FontDialogWidget {
	mu.Lock()
	d.maxSize, d.minSize = v, min(d.minSize, v)
	mu.Unlock()
	return d
}

// SetSampleText sets the text shown in the font chosen, "AaBbYyZz" by
// default as WinForms'; an app of another language shows its own.
func (d *FontDialogWidget) SetSampleText(s string) *FontDialogWidget {
	mu.Lock()
	d.sample = s
	mu.Unlock()
	return d
}

// SetFixedPitchOnly lists only monospaced families, as WinForms'
// FixedPitchOnly.
func (d *FontDialogWidget) SetFixedPitchOnly(on bool) *FontDialogWidget {
	mu.Lock()
	d.fixedPitch = on
	mu.Unlock()
	return d
}

// ShowDialog shows the dialog over parent (nil for none) and returns true if
// the user chose OK, which sets Font; Cancel, Escape and closing the window
// return false and leave Font alone. It blocks as Dialog.Run does, and
// panics on the main thread.
func (d *FontDialogWidget) ShowDialog(parent *Form) bool {
	notMain("FontDialog.ShowDialog")
	_, run := d.build(parent)
	return run()
}

// fontStyle is a style of the style list: bold or not, italic or not.
type fontStyle struct {
	name         string
	bold, italic bool
}

func fontStyles() []fontStyle {
	return []fontStyle{
		{l10n.FontRegular(), false, false},
		{l10n.FontBold(), true, false},
		{l10n.FontItalic(), false, true},
		{l10n.FontBoldItalic(), true, true},
	}
}

// build makes the dialog, and the function that runs it and applies the
// answer.
func (d *FontDialogWidget) build(parent *Form) (*Dialog, func() bool) {
	mu.Lock()
	start, title, lo, hi, fixed, families, sampleText := d.font, d.title, d.minSize, d.maxSize, d.fixedPitch, d.families, d.sample
	mu.Unlock()
	if start.Size == 0 {
		start.Size = 14
	}
	start.Size = min(max(start.Size, lo), hi)

	var names []ui.FontFamily
	for _, f := range families() {
		if !fixed || f.Monospaced {
			names = append(names, f)
		}
	}
	if i := indexFamily(names, start.Family); i >= 0 {
		start.Family = names[i].Name // the system's spelling
	} else if fixed {
		if len(names) > 0 {
			start.Family = names[0].Name // the font is proportional: start on a fixed-pitch one
		}
	} else if start.Family != "" {
		names = append([]ui.FontFamily{{Name: start.Family}}, names...) // the family SetFont gave shows even if the system lacks it
	}

	styles := fontStyles()
	startStyle := fontStyle{bold: start.Weight >= 600, italic: start.Italic}
	for _, s := range styles {
		if s.bold == startStyle.bold && s.italic == startStyle.italic {
			startStyle = s
		}
	}

	// cur is the font being edited; sample reads it, and OK applies it.
	cur := start
	curStyle := startStyle

	// The lists have a height of their own: a list in a box shorter than its
	// rows is not clipped to it, and the family list holds hundreds.
	family := ListBox(func(f *ui.FontFamily) string { return f.Name }, func(f *ui.FontFamily) string { return f.Name }).SetItems(names).SetSize(0, 200)
	if start.Family != "" {
		family.SetSelected(start.Family)
	}
	mu.Lock()
	d.familyList = family
	mu.Unlock()
	famBox := TextBox().SetText(start.Family)
	if start.Family != "" {
		family.EnsureVisible(start.Family)
	}
	family.OnSelect(func(ks []string) {
		if len(ks) == 1 {
			Change(func() { cur.Family = ks[0] })
			famBox.SetText(ks[0])
		}
	})
	famBox.OnChange(func() { // type-ahead: the first family that starts with what is typed
		typed := famBox.Text()
		for _, f := range names {
			if typed != "" && len(f.Name) >= len(typed) && strings.EqualFold(f.Name[:len(typed)], typed) {
				family.SetSelected(f.Name).EnsureVisible(f.Name)
				Change(func() { cur.Family = f.Name })
				return
			}
		}
	})
	style := ListBox(func(s *fontStyle) string { return s.name }, func(s *fontStyle) string { return s.name }).SetItems(styles).SetSize(0, 200)
	style.SetSelected(startStyle.name)
	style.OnSelect(func(ks []string) {
		if len(ks) == 1 {
			for _, s := range styles {
				if s.name == ks[0] {
					Change(func() { curStyle = s })
				}
			}
		}
	})
	var size *NumericUpDownWidget
	size = NumericUpDown(float64(lo), float64(hi), 1).SetValue(float64(start.Size)).OnChange(func() {
		v := float32(size.Value())
		Change(func() { cur.Size = v })
	})
	sample := Custom(func(c *ui.Context) ui.Element {
		weight := chosenWeight(start.Weight, startStyle.bold, curStyle.bold)
		if weight == 0 {
			weight = 400 // a font's 0 is the regular weight
		}
		e := ui.Text(c, sampleText).Font(cur.Family).FontSize(cur.Size).FontWeight(weight)
		if cur.Features != "" {
			e = e.FontFeatures(cur.Features)
		}
		if cur.Lang != "" {
			e = e.Lang(cur.Lang)
		}
		if curStyle.italic {
			e = e.Italic()
		}
		return e
	}).SetSize(0, 80)

	var dlg *Dialog
	var picked *ui.Font // OK's answer
	bar := ButtonBox().Add(
		Button(l10n.Cancel()).OnClick(func() { dlg.Close() }),
		Button(l10n.Ok()).Suggested().OnClick(func() {
			mu.Lock()
			f := cur
			f.Weight = chosenWeight(start.Weight, startStyle.bold, curStyle.bold)
			f.Italic = curStyle.italic
			picked = &f
			mu.Unlock()
			dlg.Close()
		}),
	)
	column := func(label string, w Control) *BoxWidget {
		return VBox(4).Add(Label(label)).Pack(w, true, true)
	}
	lists := HBox(12).Pack(VBox(4).Add(Label(l10n.FontFamily()), famBox).Pack(family, true, true), true, true).Pack(column(l10n.FontStyle(), style), true, true)
	dlg = NewDialog(parent, title, "", 460, 460, VBox(12).Pack(lists, true, true).Add(Label(l10n.FontSize()), size, sample, bar))
	return dlg, func() bool {
		dlg.Run()
		mu.Lock()
		defer mu.Unlock()
		if picked == nil {
			return false
		}
		d.font = *picked
		return true
	}
}

// indexFamily returns the place of the family called name in fs, without
// regard to case as the text system does, or -1.
func indexFamily(fs []ui.FontFamily, name string) int {
	for i, f := range fs {
		if strings.EqualFold(f.Name, name) {
			return i
		}
	}
	return -1
}

// chosenWeight is the weight of a font the user left in the style it started
// in (the weight it had, whatever it was), or took to regular or bold (400 or
// 700). The sample shows it and OK returns it.
func chosenWeight(startWeight int, startBold, nowBold bool) int {
	if nowBold == startBold {
		return startWeight
	}
	if nowBold {
		return 700
	}
	return 400
}
