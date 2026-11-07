package ui

import (
	"math"
	"strconv"
	"strings"
	"time"
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
// reports and is named as NumberInput is. Its arrows are the buttons
// "Increase" and "Decrease" to assistive technology, but no stops of Tab:
// a press on one steps, again as it is held, faster after a while, and
// leaves the keyboard in the text, as WinForms' does. They stay at the
// end of the box, whatever its padding. A negative step steps by its
// absolute value.
//
//	ui.SpinBox(c, &app.copies, 1, 99, 1).Label("Copies")
func coreSpinBox(c *context, value *float64, lo, hi, step float64) *node {
	return numberInput(c, value, lo, hi, math.Abs(step), true)
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
		arrowW := t.Space(5)
		if joined {
			// The arrows are out of the row's layout, at its end: the text
			// leaves them their room.
			in = coreTextInputBase(c, text).Width(t.Space(20)).Grow(1).Shrink(1).MinWidth(0).MarginEnd(arrowW)
		} else {
			in = coreTextInput(c, text).Width(t.Space(20)).Shrink(1).MinWidth(0)
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
			for i, delta := range steps {
				disabled := delta < 0 && *value <= lo || delta > 0 && *value >= hi
				var btn *node
				if joined {
					b := coreButtonBase(c).Unfocusable().KeepFocus().Grow(1).Role(RoleButton)
					b.builtWith(t)
					setStyle(b, pressSurface, staticPressSurface)
					b.Draw(func(p *Painter, r Rect) {
						color := t.Text
						if disabled {
							color = t.TextMuted.Alpha(0.5)
						}
						stepArrow(p, r, delta > 0, color)
					})
					s := b.st
					if disabled {
						// Its afterInput does not run: a hold ends here.
						s.holding, s.holdSteps = false, 0
					}
					b.afterInput(func() {
						switch {
						case b.Pressed():
							// A press steps at once, and again as it is held,
							// faster after a while, as Stepper's arrows.
							if !s.holding {
								s.holdStart, s.holding, s.holdSteps = c.now, true, 0
								s.touch()
							}
							due := 1
							if held := c.now.Sub(s.holdStart); held > 400*time.Millisecond {
								due += int((held - 400*time.Millisecond) / (80 * time.Millisecond))
							}
							for s.holdSteps < due {
								s.holdSteps++
								set(*value + delta)
								*text = format(*value)
							}
							in.Focus()
							c.After(80 * time.Millisecond)
						case b.Clicked() && s.holdSteps == 0: // pressed by assistive technology
							set(*value + delta)
							*text = format(*value)
						}
						if !b.Pressed() {
							s.holding, s.holdSteps = false, 0
						}
					})
					btn = b
				} else {
					// As at first: − then +, whatever the sign of step.
					b := coreButton(c, [2]string{"−", "+"}[i]).Padding(t.Space(1), t.Space(2))
					b.TextColor(t.Text)
					b.afterInput(func() {
						if b.Clicked() {
							set(*value + delta)
							*text = format(*value)
						}
					})
					btn = b
				}
				btn.Label(map[bool]string{true: "Increase", false: "Decrease"}[delta > 0])
				btn.nameFrom, btn.nameJoin = box, true // "Copies Increase"
				btn.Disabled(disabled)
			}
		}
		if joined {
			coreColumn(c).Absolute().Top(0).Bottom(0).End(0).Width(arrowW).
				BorderStartWidth(1).BorderColor(t.Outline()).Children(arrows)
		} else {
			arrows()
		}
		return in
	}
	if joined {
		row, _ = field(c, content)
		row.widget = "SpinBox"
		row.Padding(t.Space(1.5), t.Space(1)).PaddingStart(t.Space(2.5)).Shrink(0).Clip()
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
