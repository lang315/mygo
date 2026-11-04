package ui

import (
	"math"
	"strconv"
	"strings"
)

// NumberInput creates a text input editing *value as a number between lo
// and hi, which the arrows Up and Down, and the buttons beside it, step
// by step. What is typed applies as soon as it is a number in range, and
// shows rounded to the decimals of step once the input loses the focus.
// Changed reports a new value, Submitted an Enter in it, and Focus gives
// its text field the keyboard. Label names the field and, before
// "Increase" and "Decrease", the buttons for assistive technology.
//
//	ui.NumberInput(c, &app.copies, 1, 99, 1)
func coreNumberInput(c *context, value *float64, lo, hi, step float64) *node {
	return numberInput(c, value, lo, hi, step, false)
}

// SpinBox creates a NumberInput in the look of a classic spin box, as
// WinForms' NumericUpDown and GTK's SpinButton: one box holding the text
// and, joined at its end, an up arrow over a down arrow. It edits, steps,
// reports and is named as NumberInput is; its arrows are the buttons
// "Increase" and "Decrease".
//
//	ui.SpinBox(c, &app.copies, 1, 99, 1).Label("Copies")
func coreSpinBox(c *context, value *float64, lo, hi, step float64) *node {
	return numberInput(c, value, lo, hi, step, true)
}

// numberInput is NumberInput, or SpinBox when joined.
func numberInput(c *context, value *float64, lo, hi, step float64, joined bool) *node {
	t := c.theme
	decimals := 0
	if s := strconv.FormatFloat(step, 'f', -1, 64); strings.Contains(s, ".") {
		decimals = len(s) - strings.IndexByte(s, '.') - 1
	}
	format := func(v float64) string { return strconv.FormatFloat(v, 'f', decimals, 64) }
	var row *node
	set := func(v float64) {
		v = math.Max(lo, math.Min(hi, v))
		if decimals >= 0 {
			p := math.Pow(10, float64(decimals))
			v = math.Round(v*p) / p
		}
		if v != *value {
			*value = v
			row.st.markChanged()
			c.rt.consumed = true
		}
	}
	// content builds the text field and the buttons in row, and returns the field.
	content := func() *node {
		box := c.parent // row, which field makes before it builds content
		text := coreLocal(box, "text", func() string { return format(*value) })
		var in *node
		if joined {
			in = coreTextInputBase(c, text).Width(t.Space(20)).Padding(t.Space(1.5), t.Space(1), t.Space(1.5), 0)
		} else {
			in = coreTextInput(c, text).Width(t.Space(20))
		}
		in.widget = "NumberInput"
		in.afterInput(func() {
			if in.Changed() {
				if v, err := strconv.ParseFloat(strings.TrimSpace(*text), 64); err == nil && v >= lo && v <= hi {
					set(v)
				}
			}
			if in.Shortcut(0, KeyUp) {
				set(*value + step)
				*text = format(*value)
			}
			if in.Shortcut(0, KeyDown) {
				set(*value - step)
				*text = format(*value)
			}
			if in.Submitted() {
				row.st.markSubmitted()
			}
			if !in.Focused() {
				// What the app set, or what was typed, shown in full.
				*text = format(*value)
			}
		})
		in.hasRange, in.accRange, in.accStep = true, [3]float64{lo, hi, *value}, step
		steps := []float64{-step, step}
		if joined {
			steps = []float64{step, -step} // the up arrow over the down one
		}
		arrows := func() {
			for _, delta := range steps {
				var btn *node
				if joined {
					btn = coreButtonBase(c).Width(t.Space(5)).Grow(1)
					btn.builtWith(t)
					setStyle(btn, pressSurface, staticPressSurface)
					can := delta > 0 && *value < hi || delta < 0 && *value > lo
					btn.Draw(func(p *Painter, r Rect) {
						color := t.Text
						if !can {
							color = t.TextMuted.Alpha(0.5)
						}
						stepArrow(p, r, delta > 0, color)
					})
				} else {
					btn = coreButton(c, map[bool]string{true: "+", false: "−"}[delta > 0]).Padding(t.Space(1), t.Space(2))
					btn.TextColor(t.Text)
				}
				btn.Label(map[bool]string{true: "Increase", false: "Decrease"}[delta > 0])
				btn.nameFrom, btn.nameJoin = box, true // "Copies Increase"
				btn.Disabled(delta < 0 && *value <= lo || delta > 0 && *value >= hi)
				btn.afterInput(func() {
					if btn.Clicked() {
						set(*value + delta)
						*text = format(*value)
					}
				})
			}
		}
		if joined {
			coreColumn(c).Shrink(0).BorderWidth(0, 0, 0, 1).BorderColor(t.Border).Children(arrows)
		} else {
			arrows()
		}
		return in
	}
	if joined {
		row, _ = field(c, content)
		row.widget = "SpinBox"
		row.Padding(0, 0, 0, t.Space(2.5)).Gap(t.Space(1)).Shrink(0).AlignItems(Stretch).Clip()
		return row
	}
	row = coreRow(c).Gap(t.Space(1)).Shrink(0).AlignItems(Center)
	row.widget = "NumberInput"
	var in *node
	row.Children(func() { in = content() })
	row.focusTo = in
	in.nameFrom = row // named by the Label of the number, which is set after this build
	return row
}
