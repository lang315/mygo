package widget

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// CalcEditWidget is a number field with a calculator in its drop-down
// (DevExpress's CalcEdit). The field takes typed numbers as NumericUpDown
// does: what parses is the value at once, and leaving the field shows it
// with the decimals of SetDecimals. The button at its end, Alt+Down or
// Ctrl+Enter opens the calculator, which starts from the field's value and
// takes the keyboard while it shows: digits, the point, + - * /, %, =,
// Enter, Escape, Backspace and Delete (CE).
//
// The calculator chains left to right, as a desk calculator does: 12+3*2=
// is 30. Enter or = puts its result in the field and closes it; Escape, a
// click outside it or on the button closes it and keeps the value. A
// division by zero shows E, and changes nothing until C.
type CalcEditWidget struct {
	common[CalcEditWidget]
	value     float64
	decimals  int
	onChange  func()
	text      mirror[string] // the field's
	typed     bool           // the user typed since the text was made from value
	open      bool           // ui's Popover writes it while the view builds, under mu
	wasOpen   bool           // open in the last build
	refocus   bool           // the calculator closed by its keys: the field takes the keyboard back
	seenHides int            // the hides the last build saw: a hide closes the calculator
	calc      calcState
	// Made once, so a frame at rest allocates nothing.
	edit, children, toggle, face, keypad func()
	panel                                func(ui.Element)
	keys                                 func(ui.InputEvent) bool
	openKey                              func(ui.TextKey) bool
	rows                                 [calcRows]func()
	clicks, faces                        [len(calcPad)]func()
	in                                   ui.Element  // the build's text field
	c                                    *ui.Context // the build's, under mu while building
}

// CalcEdit makes a field of 0, with 2 decimals and the calculator closed.
func CalcEdit() *CalcEditWidget {
	e := &CalcEditWidget{decimals: 2}
	e.text.set(e.format(0))
	e.edit = e.typedIn
	e.children = e.buildChildren
	e.toggle = func() { mu.Lock(); e.open = !e.open; mu.Unlock() }
	e.face = e.buildFace
	e.keypad = e.buildKeypad
	e.panel = e.buildPanel
	e.keys = e.typedKeys
	e.openKey = e.opensBy
	for r := range e.rows {
		e.rows[r] = func() { e.buildRow(r) }
	}
	for i := range calcPad {
		e.clicks[i] = func() { mu.Lock(); e.press(calcPad[i].key); mu.Unlock() }
		e.faces[i] = func() { e.buildKeyFace(i) }
	}
	e.init(e)
	e.lookAs(skinhook.KindField) // typed into, as a TextBox
	return e
}

// Value returns the number.
func (e *CalcEditWidget) Value() float64 { mu.Lock(); defer mu.Unlock(); return e.value }

// SetValue sets the number, without running OnChange.
func (e *CalcEditWidget) SetValue(v float64) *CalcEditWidget {
	e.change(func() { e.value = v; e.setText() })
	return e
}

// SetDecimals sets the decimals the field and the calculator show the
// number with, 2 by default; below 0 is 0. The value keeps all of its own.
func (e *CalcEditWidget) SetDecimals(n int) *CalcEditWidget {
	e.change(func() {
		e.decimals = max(n, 0)
		e.setText()
		e.calc.show(e.decimals)
	})
	return e
}

// OnChange runs fn, on the event goroutine, after each change of the user:
// a typed number, or the calculator's result. A setter does not run it.
func (e *CalcEditWidget) OnChange(fn func()) *CalcEditWidget {
	e.change(func() { e.onChange = fn })
	return e
}

// ShowPopup opens the calculator, ClosePopup closes it, keeping the value.
// Neither runs OnChange.
func (e *CalcEditWidget) ShowPopup() *CalcEditWidget {
	e.change(func() { e.open = true })
	return e
}

func (e *CalcEditWidget) ClosePopup() *CalcEditWidget {
	e.change(func() { e.open = false })
	return e
}

// format is v as the field shows it. Under mu.
func (e *CalcEditWidget) format(v float64) string {
	s := strconv.FormatFloat(v, 'f', e.decimals, 64)
	if s[0] == '-' && strings.Trim(s, "-0.") == "" {
		s = s[1:] // -0.001 rounds to 0.00, not -0.00
	}
	return s
}

// setText makes the field's text from the value. Under mu.
func (e *CalcEditWidget) setText() {
	e.text.set(e.format(e.value))
	e.typed = false
}

// commit takes v, the user's, as the value: OnChange runs, then the
// binding's write, on the event goroutine. Under mu.
func (e *CalcEditWidget) commit(v float64) {
	if v == e.value {
		return
	}
	e.value = v
	h, write := e.onChange, e.bound.write
	Post(func() {
		if h != nil {
			h()
		}
		if write != nil {
			write(v)
		}
	})
}

// typedIn is ui's OnChange callback of the field: a text that parses as a
// number is the value at once, as in a NumericUpDown.
func (e *CalcEditWidget) typedIn() {
	mu.Lock()
	defer mu.Unlock()
	e.text.value, e.text.dirty = e.text.shown, false
	e.typed = true
	if v, err := strconv.ParseFloat(strings.TrimSpace(e.text.value), 64); err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) {
		e.commit(v)
	}
}

func (e *CalcEditWidget) build(c *ui.Context) ui.Element {
	if e.seenHides != e.hides { // out of sight since the last build: it closes
		e.open, e.seenHides = false, e.hides
	}
	if e.disabled { // the drop-down is an overlay, out of the disabled frame
		e.open = false
	}
	e.c = c
	frame := editFrame(c, e.children, &e.in)
	if e.errMsg != "" {
		e.post = padForErrorMark
	}
	if e.typed && !e.in.Focused() && !c.Measuring() {
		// Left: the value, with its decimals, or the last one where the
		// text did not parse. (A measuring pass has no focus.)
		e.setText()
		c.After(0)
	}
	if e.open && !e.wasOpen {
		e.calc.start(e.value, e.decimals)
	}
	if e.open {
		ui.PopoverBase(c, frame, &e.open, e.panel)
	}
	if e.refocus && !c.Measuring() { // for the real window, as render's wantFocus
		e.in.Focus()
		e.refocus = false
	}
	e.wasOpen = e.open
	e.c = nil
	return frame
}

// buildChildren builds the text and the button. Under mu, building.
func (e *CalcEditWidget) buildChildren() {
	c, t := e.c, e.c.Theme()
	p, _ := e.text.bind()
	in := ui.TextInputBase(c, p).Grow(1).MinWidth(0).Padding(t.Space(1.5), t.Space(2.5))
	e.in, e.inner = in, in
	in.OnChange(e.edit)
	if h := formAccept; h != nil {
		in.OnSubmit(h) // Enter in the line clicks the form's accept button
	}
	in.OnTextKey(e.openKey)
	b := editButtonBase(c, editKey(0)).Label(l10n.Open())
	b.Expanded(e.open).OnClick(e.toggle).Children(e.face)
}

// opensBy is the field's OnTextKey: Alt+Down and Ctrl+Enter open the
// calculator.
func (e *CalcEditWidget) opensBy(k ui.TextKey) bool {
	if k.Mods == ui.Alt && k.Key == ui.KeyDown || k.Mods == ui.Ctrl && k.Key == ui.KeyEnter {
		e.change(func() { e.open = true }) // redraws: OnTextKey promises no frame
		return true
	}
	return false
}

// buildFace draws the button's chevron. Under mu, building.
func (e *CalcEditWidget) buildFace() {
	t := e.c.Theme()
	ui.Icon(e.c, editIcon(EditDropDown)).FontSize(t.Space(3.5)).TextColor(t.TextMuted)
}

// buildPanel builds the calculator in the drop-down: it takes the keyboard
// while it shows. Under mu, building.
func (e *CalcEditWidget) buildPanel(panel ui.Element) {
	c, t := e.c, e.c.Theme()
	comboPanel(c, panel)
	panel.Padding(t.Space(2)).Gap(t.Space(1)).Width(t.Space(56)).
		Focusable().FocusRing(false).AutoFocus().HandleInput(e.keys)
	switch {
	case panel.Shortcut(0, ui.KeyEnter):
		e.press('=')
	case panel.Shortcut(0, ui.KeyBackspace):
		e.press('b')
	case panel.Shortcut(0, ui.KeyDelete):
		e.press('E')
	}
	panel.Children(e.keypad)
}

// buildKeypad builds the display and the keys' rows. Under mu, building.
func (e *CalcEditWidget) buildKeypad() {
	c, t := e.c, e.c.Theme()
	d := ui.Text(c, e.calc.display).FontSize(t.Space(5)).TextAlign(ui.End).AlignSelf(ui.Stretch).
		Padding(t.Space(1), t.Space(1.5))
	if e.calc.err {
		d.TextColor(t.Danger)
	}
	for r := range e.rows {
		ui.Row(c).Gap(t.Space(1)).Children(e.rows[r])
	}
}

// buildRow builds row r of the keys. Under mu, building.
func (e *CalcEditWidget) buildRow(r int) {
	c, t := e.c, e.c.Theme()
	for i := r * calcCols; i < min(r*calcCols+calcCols, len(calcPad)); i++ {
		k := ui.ButtonBase(c.Key(calcPadKey(i))).Unfocusable().KeepFocus().Role(ui.RoleButton).
			Grow(1).Basis(0).Height(t.Space(8)).Radius(t.Radius).Border(1, t.Outline()).
			AlignItems(ui.Center).Justify(ui.Center)
		switch {
		case k.Pressed():
			k.Background(t.SurfacePressed)
		case k.Hovered():
			k.Background(t.SurfaceHover)
		}
		k.Label(calcPad[i].label).OnClick(e.clicks[i]).Children(e.faces[i])
	}
}

// buildKeyFace draws key i's label. Under mu, building.
func (e *CalcEditWidget) buildKeyFace(i int) { ui.Text(e.c, calcPad[i].label) }

// calcPadKey keys a key of the calculator by its index.
type calcPadKey int

// typedKeys takes the text typed while the calculator has the keyboard:
// its digits, point and operators.
func (e *CalcEditWidget) typedKeys(ev ui.InputEvent) bool {
	if ev.Kind != ui.InputText {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	took := false
	for _, r := range ev.Text {
		if k := calcRune(r); k != 0 && e.open {
			e.press(k)
			took = true
		}
	}
	return took
}

// press presses key k of the calculator: = puts the result in the field,
// as the user's change, and closes it. Under mu.
func (e *CalcEditWidget) press(k byte) {
	e.calc.key(k)
	if k == '=' && !e.calc.err {
		e.commit(e.calc.acc)
		e.setText()
		e.open, e.refocus = false, true
	}
	e.calc.show(e.decimals)
}

// The calculator's keys, in rows of calcCols, = alone in the last.
const (
	calcCols = 4
	calcRows = (len(calcPad) + calcCols - 1) / calcCols
)

var calcPad = [...]struct {
	key   byte
	label string
}{
	{'%', "%"}, {'E', "CE"}, {'C', "C"}, {'b', "⌫"},
	{'7', "7"}, {'8', "8"}, {'9', "9"}, {'/', "÷"},
	{'4', "4"}, {'5', "5"}, {'6', "6"}, {'*', "×"},
	{'1', "1"}, {'2', "2"}, {'3', "3"}, {'-', "−"},
	{'n', "±"}, {'0', "0"}, {'.', "."}, {'+', "+"},
	{'=', "="},
}

// calcRune is the key a typed rune presses, 0 for none.
func calcRune(r rune) byte {
	switch r {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '%', '=', '.':
		return byte(r)
	case ',':
		return '.'
	case '-', '−':
		return '-'
	case '*', 'x', 'X', '×':
		return '*'
	case '/', '÷':
		return '/'
	}
	return 0
}

// calcState is a desk calculator: acc and op, the pending operation, and
// entry, the number being typed ("" for none, when acc shows).
type calcState struct {
	acc     float64
	op      byte
	entry   string
	err     bool
	display string
}

// start clears the calculator to v. dec is the decimals shown.
func (s *calcState) start(v float64, dec int) {
	*s = calcState{acc: v}
	s.show(dec)
}

// operand is the number typed, or acc where none is.
func (s *calcState) operand() float64 {
	if s.entry == "" {
		return s.acc
	}
	v, _ := strconv.ParseFloat(s.entry, 64)
	return v
}

// result sets acc to r, or the error where r is no number: a division by
// zero (±Inf, or NaN for 0/0), or an overflow.
func (s *calcState) result(r float64) {
	if math.IsInf(r, 0) || math.IsNaN(r) {
		s.err = true
		return
	}
	s.acc = r
}

// apply is acc op b.
func (s *calcState) apply(b float64) {
	switch s.op {
	case '+':
		s.result(s.acc + b)
	case '-':
		s.result(s.acc - b)
	case '*':
		s.result(s.acc * b)
	case '/':
		s.result(s.acc / b)
	}
}

// key presses key k: a digit, '.', an operator, '%', '=', 'n' (±), 'b'
// (Backspace), 'C' or 'E' (CE). Only C leaves the error: until then the
// display shows E and = puts nothing in the field.
func (s *calcState) key(k byte) {
	switch {
	case k >= '0' && k <= '9':
		switch {
		case len(s.entry) >= 16:
		case s.entry == "0":
			s.entry = string(k)
		case s.entry == "-0":
			s.entry = "-" + string(k)
		default:
			s.entry += string(k)
		}
	case k == '.':
		if s.entry == "" {
			s.entry = "0."
		} else if !strings.Contains(s.entry, ".") {
			s.entry += "."
		}
	case k == '+' || k == '-' || k == '*' || k == '/':
		if s.entry != "" {
			if s.op != 0 {
				s.apply(s.operand())
			} else {
				s.acc = s.operand()
			}
			s.entry = ""
		}
		s.op = k
	case k == '%':
		b := s.operand()
		switch s.op {
		case 0:
			s.result(b / 100)
		case '+':
			s.result(s.acc + s.acc*b/100)
		case '-':
			s.result(s.acc - s.acc*b/100)
		case '*':
			s.result(s.acc * b / 100)
		case '/':
			s.result(s.acc / b * 100)
		}
		s.op, s.entry = 0, ""
	case k == '=':
		if s.op != 0 {
			s.apply(s.operand())
		} else if s.entry != "" {
			s.acc = s.operand()
		}
		s.op, s.entry = 0, ""
	case k == 'n':
		switch {
		case s.entry != "" && s.entry[0] == '-':
			s.entry = s.entry[1:]
		case s.entry != "":
			s.entry = "-" + s.entry
		case s.op == 0:
			s.acc = -s.acc
		default:
			s.entry = strconv.FormatFloat(-s.acc, 'f', -1, 64)
		}
	case k == 'b':
		if s.entry != "" {
			s.entry = s.entry[:len(s.entry)-1]
			if s.entry == "" || s.entry == "-" {
				s.entry = "0"
			}
		}
	case k == 'C':
		*s = calcState{}
	case k == 'E':
		s.entry = "0"
	}
}

// show makes the display: E for the error, the number typed, or acc with
// dec decimals at most.
func (s *calcState) show(dec int) {
	switch {
	case s.err:
		s.display = "E"
	case s.entry != "":
		s.display = s.entry
	default:
		d := strconv.FormatFloat(s.acc, 'f', dec, 64)
		if strings.IndexByte(d, '.') >= 0 {
			d = strings.TrimSuffix(strings.TrimRight(d, "0"), ".")
		}
		if d == "-0" {
			d = "0"
		}
		s.display = d
	}
}

func (e *CalcEditWidget) shows() string { return e.calc.display }

func (e *CalcEditWidget) bindValue() float64  { return e.Value() }
func (e *CalcEditWidget) bindApply(v float64) { e.SetValue(v) }
