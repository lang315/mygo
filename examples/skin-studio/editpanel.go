package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

// The file dialogs and the color dialog sit behind variables, which the
// tests replace: the real ones wait for a person.
var (
	openFile, saveFile = widget.OpenFile, widget.SaveFile
	pickColor          = showColorDialog
)

// showColorDialog asks for a color starting from c; false for a cancel.
func showColorDialog(f *widget.Form, c ui.Color) (ui.Color, bool) {
	d := widget.ColorDialog().SetColor(c)
	if !d.ShowDialog(f) {
		return c, false
	}
	return d.Color(), true
}

var skinFiles = widget.Filter{Name: "Skin file", Extensions: []string{"json"}}

// editPanel is the Edit tab: the editor's controls, which change the skin in
// use at each edit.
type editPanel struct {
	form *widget.Form
	ed   *editor
	dark bool // the appearance whose tokens the rows show; not the app's

	errs    *widget.ErrorProviderWidget
	rows    map[string]widget.Control // token controls, by path
	looks   map[string]widget.Control // look boxes, by kind
	show    map[string]func(skin.Tokens)
	accent  *widget.TextBoxWidget
	openBtn *widget.ButtonWidget
	saveBtn *widget.ButtonWidget

	base, palette, density, appearance *widget.ComboBoxWidget
	name                               *widget.TextBoxWidget

	status     *widget.StatusLabelWidget
	report     string // Check's error, "" when every pair passes
	edited     bool
	markEdited func(bool) // the tool strip's skin box
}

// newEditPanel makes the panel of the form f; the studio sets status and
// markEdited.
func newEditPanel(f *widget.Form) *editPanel {
	return &editPanel{form: f, ed: newEditor("calm"), errs: widget.ErrorProvider(),
		rows: map[string]widget.Control{}, looks: map[string]widget.Control{}, show: map[string]func(skin.Tokens){}}
}

// groups are the sections of the token list, by the type of a Tokens field,
// in the order they are shown.
var groups = []string{"Colors", "Sizes", "Type roles", "Families", "Chart", "Shadows", "Other"}

// groupOf returns the section of the token at path.
func groupOf(path string) string {
	top, _, _ := strings.Cut(path, ".")
	f, _ := reflect.TypeFor[skin.Tokens]().FieldByName(top)
	switch {
	case f.Type == colorType:
		return "Colors"
	case f.Type.Kind() == reflect.Float32:
		return "Sizes"
	case f.Type == reflect.TypeFor[skin.Type]():
		return "Type roles"
	case f.Type == reflect.TypeFor[skin.Family]():
		return "Families"
	case f.Type.Kind() == reflect.Array:
		return "Chart"
	case f.Type == reflect.TypeFor[skin.Shadow]():
		return "Shadows"
	}
	return "Other"
}

// build makes the tab.
func (p *editPanel) build() widget.Control {
	p.base = widget.ComboBox(option("calm", "Calm", "classic", "Classic", "highcontrast", "HighContrast")...).SetAccessibleName("Edit base")
	p.base.OnChange(func() { p.ed.reset(p.base.Value()); p.applyAndShow() })
	p.palette = widget.ComboBox().SetAccessibleName("Edit palette")
	p.palette.OnChange(p.paletteChanged)
	p.accent = widget.TextBox().SetAccessibleName("Accent").SetPlaceholder("#rrggbb")
	p.accent.OnChange(func() {
		// The box is on only in the accent mode: an empty box is a color not
		// yet typed, not a way back to a palette (the palette box is).
		err := errors.New("Accent: want a color #rrggbb")
		if t := p.accent.Text(); t != "" {
			err = p.ed.setAccent(t)
		}
		if err != nil {
			p.errs.SetError(p.accent, err.Error())
			return
		}
		p.errs.SetError(p.accent, "")
		p.applyAndShow()
	})
	p.density = widget.ComboBox(option("standard", "Standard", "compact", "Compact")...).SetAccessibleName("Density")
	p.density.OnChange(func() {
		p.ed.density = skin.Standard
		if p.density.Value() == "compact" {
			p.ed.density = skin.Compact
		}
		p.applyAndShow()
	})
	p.appearance = widget.ComboBox(option("light", "Light", "dark", "Dark")...).SetValue("light").SetAccessibleName("Appearance")
	p.appearance.OnChange(func() { p.dark = p.appearance.Value() == "dark"; p.refresh() })
	p.name = widget.TextBox().SetAccessibleName("Skin name")
	p.name.OnChange(func() { p.ed.name = p.name.Text(); p.apply() })
	p.openBtn = widget.Button("Open…").OnClick(p.open)
	p.saveBtn = widget.Button("Save…").OnClick(p.save)

	header := widget.FlowPanel(8).Add(
		widget.Label("Base"), p.base, widget.Label("Colors from"), p.palette, p.accent,
		widget.Label("Spacing"), p.density, widget.Label("Shows"), p.appearance,
		widget.Label("Name"), p.name,
		p.openBtn, p.saveBtn, widget.Button("Reset").OnClick(func() { p.ed.reset(p.ed.base); p.applyAndShow() }),
	)

	fields := widget.Fields()
	for _, g := range groups {
		var in []string
		for _, path := range tokenPaths() {
			if groupOf(path) == g {
				in = append(in, path)
			}
		}
		fields.Section(g)
		for _, path := range in {
			fields.Row(path, p.row(path))
		}
	}
	fields.Section("Looks")
	lk := reflect.TypeFor[skin.Looks]()
	for i := range lk.NumField() {
		kind := lk.Field(i).Name
		box := widget.TextBox().SetAccessibleName("Look " + kind)
		box.OnChange(func() {
			if err := p.ed.setLook(kind, box.Text()); err != nil {
				p.errs.SetError(box, err.Error())
				return
			}
			p.errs.SetError(box, "")
			p.apply()
		})
		p.looks[kind] = box
		fields.Row(kind, box)
	}
	p.refresh()
	return widget.VBox(8).Padding(4).Add(header).Pack(widget.ScrollPanel(fields), true, true).Expand()
}

// row makes the control of the token at path: a hex box and a "…" button for
// a color, a number box for a size or a weight, a check box for a flag, a
// text box for the font.
func (p *editPanel) row(path string) widget.Control {
	var zero skin.Tokens
	v, _ := leafAt(reflect.ValueOf(&zero).Elem(), path)
	switch {
	case v.Type() == colorType:
		box := widget.TextBox().SetAccessibleName(path)
		box.OnChange(func() { p.edit(box, path, box.Text()) })
		p.rows[path] = box
		p.show[path] = func(t skin.Tokens) { box.SetText(tokenText(t, path)) }
		more := widget.Button("…").SetAccessibleName("Choose " + path).OnClick(func() { p.pickColor(path) })
		return widget.HBox(4).Pack(box, true, true).Add(more)
	case v.Kind() == reflect.Float32, v.Kind() == reflect.Int:
		n := widget.NumericUpDown(0, 400, 1)
		if v.Kind() == reflect.Int {
			n = widget.NumericUpDown(100, 900, 100)
		}
		n.SetAccessibleName(path)
		n.OnChange(func() { p.edit(n, path, strconv.FormatFloat(n.Value(), 'g', -1, 32)) })
		p.rows[path] = n
		p.show[path] = func(t skin.Tokens) {
			f, _ := strconv.ParseFloat(tokenText(t, path), 64)
			n.SetValue(f)
		}
		return n
	case v.Kind() == reflect.Bool:
		c := widget.CheckBox("on").SetAccessibleName(path)
		c.OnChange(func() { p.edit(c, path, strconv.FormatBool(c.Checked())) })
		p.rows[path] = c
		p.show[path] = func(t skin.Tokens) { c.SetChecked(tokenText(t, path) == "true") }
		return c
	}
	box := widget.TextBox().SetAccessibleName(path)
	box.OnChange(func() { p.edit(box, path, box.Text()) })
	p.rows[path] = box
	p.show[path] = func(t skin.Tokens) { box.SetText(tokenText(t, path)) }
	return box
}

// edit sets the token at path of the appearance shown to value, as the user
// typed it, and applies it; a value the editor refuses is an error on the
// control, and the skin keeps its last good one.
func (p *editPanel) edit(w widget.Control, path, value string) {
	if err := p.ed.set(p.dark, path, value); err != nil {
		p.errs.SetError(w, err.Error())
		return
	}
	p.errs.SetError(w, "")
	p.apply()
}

// pickColor asks for the color of the token at path and sets it, keeping the
// token's own opacity: the dialog's colors are opaque.
func (p *editPanel) pickColor(path string) {
	cur, _ := parseColor(tokenText(p.ed.build().Tokens(p.dark), path))
	c, ok := pickColor(p.form, cur)
	if !ok {
		return
	}
	c.A = cur.A
	p.edit(p.rows[path], path, hexText(c))
	p.show[path](p.ed.build().Tokens(p.dark))
}

// paletteChanged applies the palette box: a palette of the base, or the
// accent, which starts from the accent the skin has.
func (p *editPanel) paletteChanged() {
	if v := p.palette.Value(); v != "accent" {
		p.ed.accent, p.ed.palette = "", v
		p.applyAndShow()
		return
	}
	cur, _ := parseColor(tokenText(p.ed.build().Tokens(false), "Accent.Base"))
	_ = p.ed.setAccent(hexText(cur)) // opaque by construction
	p.applyAndShow()
}

// apply turns the skin being edited on and reports its contrast in the
// status strip.
func (p *editPanel) apply() {
	s := p.ed.build()
	skin.Use(s)
	p.report = ""
	text := "Contrast: all pairs pass"
	if err := s.Check(); err != nil {
		p.report = err.Error()
		text = fmt.Sprintf("Contrast: %d pairs under their floor", len(strings.Split(strings.TrimSpace(p.report), "\n")))
	}
	p.status.SetText(text).SetTooltip(p.report)
	if !p.edited {
		p.edited = true
		p.markEdited(true)
	}
}

// applyAndShow applies the skin and puts every control to what it holds.
func (p *editPanel) applyAndShow() {
	p.apply()
	p.refresh()
}

// refresh puts every control to what the editor holds: the header, the rows
// of the appearance shown, and the looks. It clears the errors.
func (p *editPanel) refresh() {
	p.errs.Clear()
	e := p.ed
	p.base.SetValue(e.base)
	s := skins[e.base]
	opts := []widget.Option{}
	for _, pal := range s.Palettes() {
		opts = append(opts, widget.Option{Value: pal.Name(), Text: pal.Name()})
	}
	opts = append(opts, widget.Option{Value: "accent", Text: "Accent…"})
	p.palette.SetOptions(opts...)
	if e.accent != "" {
		p.palette.SetValue("accent")
	} else {
		p.palette.SetValue(e.palette)
	}
	p.accent.SetText(e.accent).SetDisabled(e.accent == "")
	p.density.SetValue(map[skin.Density]string{skin.Standard: "standard", skin.Compact: "compact"}[e.density])
	p.name.SetText(e.name)
	t := e.build().Tokens(p.dark)
	for _, show := range p.show {
		show(t)
	}
	lk := reflect.ValueOf(e.looks)
	for kind, box := range p.looks {
		box.(*widget.TextBoxWidget).SetText(lk.FieldByName(kind).String())
	}
}

// pick resets the editor to the skin key and its palette, as the tool strip
// chose them: nothing is edited yet.
func (p *editPanel) pick(key, palette string) {
	p.ed.reset(key)
	for _, pal := range skins[p.ed.base].Palettes() {
		if pal.Name() == palette {
			p.ed.palette = palette
		}
	}
	p.edited, p.report = false, ""
	p.status.SetText("Ready").SetTooltip("")
	p.refresh()
}

// open reads a skin file the user picks into the editor.
func (p *editPanel) open() {
	path := openFile(p.form, "Open a skin", skinFiles)
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err == nil {
		err = p.ed.open(data)
	}
	if err != nil {
		p.errs.SetError(p.openBtn, err.Error())
		return
	}
	p.errs.SetError(p.openBtn, "")
	p.applyAndShow()
}

// save writes the skin file of the edit where the user says.
func (p *editPanel) save() {
	path := saveFile(p.form, "Save the skin", p.ed.name+".json", skinFiles)
	if path == "" {
		return
	}
	if err := os.WriteFile(path, p.ed.save(), 0o644); err != nil {
		p.errs.SetError(p.saveBtn, err.Error())
		return
	}
	p.errs.SetError(p.saveBtn, "")
}
