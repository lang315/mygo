package widget

import (
	"slices"
	"unicode"
)

// A mask is a WinForms-style input mask: a template of slots the user fills,
// and literals between them. It is pure: the editing functions take the text
// shown and the edit and return the text and the caret to show, which ui's
// OnEdit applies.
//
// Characters of the specification:
//
//	0  a digit, required        9  a digit, optional
//	#  a digit, space, + or -, optional
//	L  a letter, required       ?  a letter, optional
//	A  a letter or digit, required   a  a letter or digit, optional
//	&  any character, required       C  any character, optional
//	>  letters typed after it are upper case, < lower case, | as typed
//	\  the next character is a literal
//
// Every other character is a literal.
type mask struct {
	slots    []slot
	prompt   rune
	rejected []rune // the runes the last edit refused
}

type slot struct {
	lit      rune // a literal's character, 0 for an editable slot
	required bool
	accept   func(r rune) bool
	conv     func(r rune) rune
}

func newMask(spec string, prompt rune) *mask {
	m := &mask{prompt: prompt}
	var conv func(rune) rune
	runes := []rune(spec)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		var s slot
		switch r {
		case '>':
			conv = unicode.ToUpper
			continue
		case '<':
			conv = unicode.ToLower
			continue
		case '|':
			conv = nil
			continue
		case '\\':
			if i+1 < len(runes) {
				i++
				m.slots = append(m.slots, slot{lit: runes[i]})
			}
			continue
		case '0':
			s = slot{required: true, accept: unicode.IsDigit}
		case '9':
			s = slot{accept: unicode.IsDigit}
		case '#':
			s = slot{accept: func(r rune) bool { return unicode.IsDigit(r) || r == ' ' || r == '+' || r == '-' }}
		case 'L':
			s = slot{required: true, accept: unicode.IsLetter}
		case '?':
			s = slot{accept: unicode.IsLetter}
		case 'A':
			s = slot{required: true, accept: isLetterOrDigit}
		case 'a':
			s = slot{accept: isLetterOrDigit}
		case '&':
			s = slot{required: true, accept: func(r rune) bool { return !unicode.IsControl(r) }}
		case 'C':
			s = slot{accept: func(r rune) bool { return !unicode.IsControl(r) }}
		default:
			m.slots = append(m.slots, slot{lit: r})
			continue
		}
		s.conv = conv
		m.slots = append(m.slots, s)
	}
	return m
}

func isLetterOrDigit(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func (s slot) editable() bool { return s.lit == 0 }

// cells reads the shown text into the value of each slot: 0 for an empty
// slot (and the literals, which hold none).
func (m *mask) cells(text string) []rune {
	shown := []rune(text)
	cells := make([]rune, len(m.slots))
	for i, s := range m.slots {
		if s.editable() && i < len(shown) && shown[i] != m.prompt {
			cells[i] = shown[i]
		}
	}
	return cells
}

// render writes the cells as the text shown.
func (m *mask) render(cells []rune) string {
	out := make([]rune, len(m.slots))
	for i, s := range m.slots {
		switch {
		case !s.editable():
			out[i] = s.lit
		case cells[i] == 0:
			out[i] = m.prompt
		default:
			out[i] = cells[i]
		}
	}
	return string(out)
}

// display is the text of an empty mask.
func (m *mask) display() string { return m.render(make([]rune, len(m.slots))) }

// value returns the characters typed, without literals and prompts.
func (m *mask) value(text string) string { return m.valueOf(m.cells(text)) }

func (m *mask) valueOf(cells []rune) string {
	var out []rune
	for _, c := range cells {
		if c != 0 {
			out = append(out, c)
		}
	}
	return string(out)
}

// completed reports whether every required slot holds a character, full
// whether every editable slot does.
func (m *mask) completed(text string) bool { return m.completedOf(m.cells(text)) }

func (m *mask) completedOf(cells []rune) bool {
	for i, s := range m.slots {
		if s.editable() && s.required && cells[i] == 0 {
			return false
		}
	}
	return true
}

func (m *mask) full(text string) bool { return m.fullOf(m.cells(text)) }

func (m *mask) fullOf(cells []rune) bool {
	for i, s := range m.slots {
		if s.editable() && cells[i] == 0 {
			return false
		}
	}
	return true
}

// fit offers r to slot i: the character it would hold, or 0 if it takes none.
// The prompt is a display, not a character the slot refuses.
func (m *mask) fit(i int, r rune) rune {
	s := m.slots[i]
	if s.conv != nil {
		r = s.conv(r)
	}
	if !s.accept(r) {
		return 0
	}
	return r
}

// next returns the first editable slot at or after i, or -1.
func (m *mask) next(i int) int {
	for ; i < len(m.slots); i++ {
		if i >= 0 && m.slots[i].editable() {
			return i
		}
	}
	return -1
}

// prev returns the last editable slot before i, or -1.
func (m *mask) prev(i int) int {
	for i--; i >= 0; i-- {
		if m.slots[i].editable() {
			return i
		}
	}
	return -1
}

// flow puts the characters in order into the editable slots from slot i on,
// the ones the slots do not take dropped, and returns how many were.
func (m *mask) flow(cells []rune, from int, chars []rune) (dropped int) {
	j := m.next(from)
	for k, r := range chars {
		for j >= 0 && m.fit(j, r) == 0 {
			j = m.next(j + 1)
		}
		if j < 0 {
			return len(chars) - k
		}
		cells[j] = m.fit(j, r)
		j = m.next(j + 1)
	}
	return 0
}

// edit applies an edit of the user to the text shown: see editCells.
func (m *mask) edit(text string, start, end int, insert string, caret ...int) (string, int, bool) {
	cells, at := m.editCells(m.cells(text), start, end, insert, caret...)
	return m.render(cells), at, true
}

// editCells applies an edit of the user: the slots from start to end
// replaced by insert, with the caret at caret before it. It returns the cells
// after the edit and the caret. Characters after the edit move to make room
// or to close the gap, as WinForms' mask in insert mode does; an insert
// that would push one out of the mask is refused whole, and m.rejected holds
// its first character. A character that is the literal at the caret steps
// over it (WinForms' SkipLiterals).
func (m *mask) editCells(cells []rune, start, end int, insert string, caret ...int) ([]rune, int) {
	m.rejected = m.rejected[:0]
	n := len(m.slots)
	start, end = max(0, min(start, n)), max(0, min(end, n))
	if start == end && insert == "" {
		return cells, start // nothing goes, as when an input method starts
	}
	orig := cells
	cells = slices.Clone(cells)
	if insert == "" && !m.anyEditable(start, end) {
		// Only literals go: Backspace takes the character before them, Delete
		// the one after.
		at := -1
		if len(caret) > 0 && caret[0] == start && end > start {
			at = m.next(end) // Delete
		} else {
			at = m.prev(start)
			if at < 0 {
				at = m.next(end)
			}
		}
		if at < 0 {
			return orig, start
		}
		start, end = at, at+1
	}
	// What follows the edit, to flow again after it.
	var rest []rune
	for i := end; i < n; i++ {
		if m.slots[i].editable() && cells[i] != 0 {
			rest = append(rest, cells[i])
		}
	}
	for i := start; i < n; i++ {
		if m.slots[i].editable() {
			cells[i] = 0
		}
	}
	pos := start
	if insert != "" {
		for _, r := range insert {
			p, placed := pos, false
			for p < n {
				s := m.slots[p]
				if s.editable() {
					if c := m.fit(p, r); c != 0 {
						cells[p] = c
						p++
						placed = true
					}
					break
				}
				p++
				if s.lit == r {
					placed = true
					break
				}
			}
			if placed {
				pos = p
			}
			if !placed {
				m.rejected = append(m.rejected, r)
			}
		}
	}
	if m.flow(cells, pos, rest) > 0 && insert != "" {
		m.rejected = append(m.rejected[:0], []rune(insert)[0])
		return orig, start
	}
	return cells, pos
}

func (m *mask) anyEditable(start, end int) bool {
	for i := start; i < end && i < len(m.slots); i++ {
		if m.slots[i].editable() {
			return true
		}
	}
	return false
}

// fill returns the cells of the mask holding chars, one in each editable slot
// from the first, and whether each one fitted its slot (nothing is filled
// otherwise). chars may also be the text with its literals, as WinForms'
// Text takes it: a space or the prompt then stands for a slot not filled.
func (m *mask) fill(chars string) ([]rune, bool) {
	cells := make([]rune, len(m.slots))
	j := m.next(0)
	ok := true
	for _, r := range chars {
		if j < 0 {
			ok = false
			break
		}
		c := m.fit(j, r)
		if c == 0 {
			ok = false
			break
		}
		cells[j] = c
		j = m.next(j + 1)
	}
	if ok {
		return cells, true
	}
	shown := []rune(chars)
	if len(shown) != len(m.slots) {
		return nil, false
	}
	cells = make([]rune, len(m.slots))
	for i, s := range m.slots {
		switch {
		case !s.editable():
			if shown[i] != s.lit {
				return nil, false
			}
		case shown[i] == ' ' || shown[i] == m.prompt:
		default:
			if cells[i] = m.fit(i, shown[i]); cells[i] == 0 {
				return nil, false
			}
		}
	}
	return cells, true
}

// refill puts the characters of old, as the cells of another mask held them,
// into this one, those that do not fit dropped.
func (m *mask) refill(chars string) []rune {
	cells := make([]rune, len(m.slots))
	m.flow(cells, 0, []rune(chars))
	return cells
}

// blanks returns the cells as text with a space for each slot not filled.
func (m *mask) blanks(cells []rune) string {
	out := []rune(m.render(cells))
	for i, s := range m.slots {
		if s.editable() && cells[i] == 0 {
			out[i] = ' '
		}
	}
	return string(out)
}
