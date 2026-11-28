package widget

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// TimeSpanEditWidget edits a time.Duration, shown as [-][d.]hh:mm[:ss]
// (DevExpress's TimeSpanEdit). Up and Down, and the − and + buttons at
// the field's end, step the part under the caret (days, hours, minutes or
// seconds), carrying into the next part. A value typed in whole is read
// when the focus leaves the field, or on Enter; text that does not read
// as one goes back to the last value.
type TimeSpanEditWidget struct {
	common[TimeSpanEditWidget]
	value     time.Duration // the value, in the range
	min, max  time.Duration
	days      bool // SetShowDays
	noSeconds bool // SetShowSeconds(false)
	onChange  func()
	text      mirror[string] // the field's: formatted, or what the user is typing
	formatted string         // value as shown
	caret     int            // the caret in the last build, in runes
	putCaret  int            // a part to put the caret at the end of in the next build, or -1
	// Made once, so a frame at rest allocates nothing: the build's children,
	// ui's key callback, the buttons' clicks and faces.
	children func()
	key      func(k ui.TextKey) bool
	clicks   [2]func() // Decrease, Increase
	faces    [2]func()
	c        *ui.Context // the build's, under mu while building
}

// The parts of a duration, as the caret finds them.
const (
	spanDays = iota
	spanHours
	spanMinutes
	spanSeconds
)

var spanUnits = [...]time.Duration{spanDays: 24 * time.Hour, spanHours: time.Hour, spanMinutes: time.Minute, spanSeconds: time.Second}

// TimeSpanEdit makes a duration field at 0, from 0 to 9999:59:59, showing
// seconds and no days: the hours run past 23.
func TimeSpanEdit() *TimeSpanEditWidget {
	s := &TimeSpanEditWidget{max: 9999*time.Hour + 59*time.Minute + 59*time.Second, putCaret: -1}
	s.children = s.buildChildren
	s.key = s.onKey
	for i := range s.clicks {
		s.clicks[i] = func() { mu.Lock(); s.step(s.text.shown, i == 1); mu.Unlock() }
		s.faces[i] = func() { s.face(i) }
	}
	s.reformat()
	s.init(s)
	s.lookAs(skinhook.KindField) // typed into, as a TextBox
	return s
}

// Value returns the duration.
func (s *TimeSpanEditWidget) Value() time.Duration { mu.Lock(); defer mu.Unlock(); return s.value }

// SetValue sets the duration, clamped to the range, without running
// OnChange.
func (s *TimeSpanEditWidget) SetValue(d time.Duration) *TimeSpanEditWidget {
	s.change(func() { s.value = s.clamp(d); s.reformat() })
	return s
}

// SetRange sets the least and the greatest duration, and clamps the value
// to them without running OnChange. min > max swaps them.
func (s *TimeSpanEditWidget) SetRange(min, max time.Duration) *TimeSpanEditWidget {
	if min > max {
		min, max = max, min
	}
	s.change(func() {
		s.min, s.max = min, max
		s.value = s.clamp(s.value)
		s.reformat()
	})
	return s
}

// SetShowSeconds shows the seconds, as at first, or hides them; hidden,
// they stay in the value.
func (s *TimeSpanEditWidget) SetShowSeconds(on bool) *TimeSpanEditWidget {
	s.change(func() { s.noSeconds = !on; s.reformat() })
	return s
}

// SetShowDays shows the days before the hours, as 1.02:00:00, or not, as
// at first: the hours then run past 23, as 26:00:00.
func (s *TimeSpanEditWidget) SetShowDays(on bool) *TimeSpanEditWidget {
	s.change(func() { s.days = on; s.reformat() })
	return s
}

// OnChange runs fn, on the event goroutine, after each change of the
// user: a step, or a typed value read. A setter does not run it.
func (s *TimeSpanEditWidget) OnChange(fn func()) *TimeSpanEditWidget {
	s.change(func() { s.onChange = fn })
	return s
}

// clamp returns d in the range. Under mu.
func (s *TimeSpanEditWidget) clamp(d time.Duration) time.Duration { return min(max(d, s.min), s.max) }

// reformat shows the value, in place of any typed text. Under mu.
func (s *TimeSpanEditWidget) reformat() {
	s.formatted = formatSpan(s.value, s.days, !s.noSeconds)
	s.text.set(s.formatted)
}

// put makes d, clamped, the value, as the user's change: OnChange and the
// binding's write run if it changed. The field shows it, with the caret
// at the end of part, unless part is -1. Under mu.
func (s *TimeSpanEditWidget) put(d time.Duration, part int) {
	d = s.clamp(d)
	s.putCaret = part
	changed := d != s.value
	s.value = d
	s.reformat()
	h, write := s.onChange, s.bound.write
	if !changed || h == nil && write == nil {
		return
	}
	Post(func() {
		if h != nil {
			h()
		}
		if write != nil {
			write(d)
		}
	})
}

// step steps the part under the caret of text, the field's, up or down:
// from text's value if it reads as one, else from the last value. Under
// mu.
func (s *TimeSpanEditWidget) step(text string, up bool) {
	part := spanPart(text, s.caret)
	d, ok := parseSpan(text)
	if !ok || text == s.formatted {
		d = s.value // nothing typed: keep what the field hides, seconds and fractions
	}
	if up {
		d += spanUnits[part]
	} else {
		d -= spanUnits[part]
	}
	s.put(d, part)
}

// commit reads text, the field's, as the user's value, or shows the last
// value again if it does not read as one. Under mu.
func (s *TimeSpanEditWidget) commit(text string) {
	if text == s.formatted {
		return // nothing typed: a value with hidden seconds keeps them
	}
	if d, ok := parseSpan(text); ok {
		s.put(d, -1)
	} else {
		s.reformat()
	}
}

// onKey steps the part under the caret on Up and Down.
func (s *TimeSpanEditWidget) onKey(k ui.TextKey) bool {
	if k.Mods != 0 || k.Key != ui.KeyUp && k.Key != ui.KeyDown {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	s.step(k.Text(), k.Key == ui.KeyUp)
	k.SetText(s.text.value)
	return true
}

func (s *TimeSpanEditWidget) build(c *ui.Context) ui.Element {
	s.c = c
	frame := editFrame(c, s.children)
	s.c = nil
	s.edgeEditFrame(c.Theme(), frame)
	return frame
}

// buildChildren builds the text and the buttons. Under mu, building.
func (s *TimeSpanEditWidget) buildChildren() {
	c, t := s.c, s.c.Theme()
	p, _ := s.text.bind()
	in := ui.TextInputBase(c, p).Grow(1).MinWidth(0).Padding(t.Space(1.5), t.Space(2.5))
	s.inner = in
	in.OnTextKey(s.key)
	if !c.Measuring() {
		// Leaving the field, or Enter, reads what was typed; Enter then
		// clicks the form's accept button, which reads the value.
		submitted := in.Submitted()
		if in.FocusLeft() || submitted {
			s.commit(*p)
			s.text.shown = s.text.value // shown in this build, as LookUpEdit's show
		}
		if submitted {
			if h := formAccept; h != nil {
				h()
			}
		}
		if part := s.putCaret; part >= 0 {
			i := spanEnd(*p, part)
			in.SetTextSelection(i, i)
			s.putCaret = -1
		}
		if in.Focused() {
			_, s.caret = in.TextAnchorCaret()
		}
	}
	for i := range s.clicks {
		e := editButtonBase(c.Key(spanKey(i)))
		e.Label([...]string{"Decrease", "Increase"}[i]) // as ui's SpinBox names its arrows
		e.OnClick(s.clicks[i]).Children(s.faces[i])
	}
}

// spanKey keys a TimeSpanEdit's button.
type spanKey int

// face draws button i's icon: − or +. Under mu, building.
func (s *TimeSpanEditWidget) face(i int) {
	t := s.c.Theme()
	ui.Icon(s.c, editIcon([...]EditButtonKind{EditMinus, EditPlus}[i])).FontSize(t.Space(3.5)).TextColor(t.TextMuted)
}

func (s *TimeSpanEditWidget) bindValue() time.Duration  { return s.Value() }
func (s *TimeSpanEditWidget) bindApply(v time.Duration) { s.SetValue(v) }

// formatSpan shows d as [-][d.]hh:mm[:ss], its whole seconds.
func formatSpan(d time.Duration, days, seconds bool) string {
	sec := int64(d / time.Second)
	var b [32]byte
	out := b[:0]
	if sec < 0 {
		out, sec = append(out, '-'), -sec
	}
	h := sec / 3600
	if days {
		out = strconv.AppendInt(out, h/24, 10)
		out, h = append(out, '.'), h%24
	}
	out = twoDigits(out, h)
	out = twoDigits(append(out, ':'), sec/60%60)
	if seconds {
		out = twoDigits(append(out, ':'), sec%60)
	}
	return string(out)
}

// twoDigits appends n with at least two digits.
func twoDigits(b []byte, n int64) []byte {
	if n < 10 {
		b = append(b, '0')
	}
	return strconv.AppendInt(b, n, 10)
}

// spanMaxHours bounds the hours parseSpan reads, days included, so a step
// of a day past them stays a Duration (2562047 hours at most).
const spanMaxHours = 2_500_000

// parseSpan reads [-][d.]h[:m[:s]], m and s below 60, around spaces.
func parseSpan(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var days int64
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		n, ok := spanNumber(s[:dot]) // digits alone: 1:00.5 is no days
		if !ok {
			return 0, false
		}
		days, s = n, s[dot+1:]
	}
	var n [3]int64 // hours, minutes, seconds
	for i := range n {
		part, rest, more := strings.Cut(s, ":")
		v, ok := spanNumber(part)
		if !ok || i > 0 && v > 59 {
			return 0, false
		}
		n[i], s = v, rest
		if !more {
			break
		}
		if i == len(n)-1 {
			return 0, false // a fourth part
		}
	}
	if days*24+n[0] > spanMaxHours {
		return 0, false
	}
	d := time.Duration(days*24+n[0])*time.Hour + time.Duration(n[1])*time.Minute + time.Duration(n[2])*time.Second
	if neg {
		d = -d
	}
	return d, true
}

// spanNumber reads s, one to nine digits, as a number: no more, so that
// it cannot wrap an int64, nor days*24 either.
func spanNumber(s string) (int64, bool) {
	if s == "" || len(s) > 9 {
		return 0, false
	}
	var n int64
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int64(s[i]-'0')
	}
	return n, true
}

// spanPart is the part of text under caret, an offset in runes: after
// two colons the seconds, after one the minutes, after the days' dot the
// hours; before it the days, and the hours where there is none.
func spanPart(text string, caret int) int {
	colons, dot, i := 0, false, 0
	for _, r := range text {
		if i == caret {
			break
		}
		switch {
		case r == ':':
			colons++
		case r == '.':
			dot = true // the days' (after a colon, a colon's case wins)
		}
		i++
	}
	switch {
	case colons >= 2:
		return spanSeconds
	case colons == 1:
		return spanMinutes
	case dot:
		return spanHours
	}
	head, _, _ := strings.Cut(text, ":")
	if strings.Contains(head, ".") {
		return spanDays
	}
	return spanHours
}

// spanEnd is where part ends in text, formatted, in runes: before the
// separator after it, or at the end.
func spanEnd(text string, part int) int {
	at := len(text)
	switch part {
	case spanDays:
		at = strings.IndexByte(text, '.')
	case spanHours:
		at = strings.IndexByte(text, ':')
	case spanMinutes:
		if first := strings.IndexByte(text, ':'); first >= 0 {
			if second := strings.IndexByte(text[first+1:], ':'); second >= 0 {
				at = first + 1 + second
			}
		}
	}
	if at < 0 {
		return len(text)
	}
	return at
}
