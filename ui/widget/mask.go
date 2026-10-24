package widget

import "unicode"

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
func (m *mask) value(text string) string {
	var out []rune
	for _, c := range m.cells(text) {
		if c != 0 {
			out = append(out, c)
		}
	}
	return string(out)
}

// completed reports whether every required slot holds a character, full
// whether every editable slot does.
func (m *mask) completed(text string) bool {
	cells := m.cells(text)
	for i, s := range m.slots {
		if s.editable() && s.required && cells[i] == 0 {
			return false
		}
	}
	return true
}

func (m *mask) full(text string) bool {
	cells := m.cells(text)
	for i, s := range m.slots {
		if s.editable() && cells[i] == 0 {
			return false
		}
	}
	return true
}

// fit offers r to slot i: the character it would hold, or 0 if it takes none.
func (m *mask) fit(i int, r rune) rune {
	s := m.slots[i]
	if s.conv != nil {
		r = s.conv(r)
	}
	if r == m.prompt || !s.accept(r) {
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
// the ones the slots do not take dropped, and clears the slots after.
func (m *mask) flow(cells []rune, from int, chars []rune) {
	j := m.next(from)
	for _, r := range chars {
		for j >= 0 && m.fit(j, r) == 0 {
			j = m.next(j + 1)
		}
		if j < 0 {
			return
		}
		cells[j] = m.fit(j, r)
		j = m.next(j + 1)
	}
}

// edit applies an edit of the user: the runes from start to end of text
// replaced by insert, with the caret at caret before it. It returns the text
// to show and the caret after the edit. Characters after the edit move to
// make room or to close the gap, as WinForms' mask in insert mode does.
func (m *mask) edit(text string, start, end int, insert string, caret ...int) (string, int, bool) {
	m.rejected = m.rejected[:0]
	n := len(m.slots)
	start, end = max(0, min(start, n)), max(0, min(end, n))
	cells := m.cells(text)
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
			return m.render(cells), start, true
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
		j := m.next(start)
		for _, r := range insert {
			if j < 0 {
				m.rejected = append(m.rejected, r)
				continue
			}
			c := m.fit(j, r)
			if c == 0 {
				m.rejected = append(m.rejected, r)
				continue
			}
			cells[j] = c
			pos = j + 1
			j = m.next(j + 1)
		}
	}
	m.flow(cells, pos, rest)
	return m.render(cells), pos, true
}

func (m *mask) anyEditable(start, end int) bool {
	for i := start; i < end && i < len(m.slots); i++ {
		if m.slots[i].editable() {
			return true
		}
	}
	return false
}

// fill returns the text of the mask holding chars, one in each editable slot
// from the first, and whether each one fitted its slot (nothing is filled
// otherwise).
func (m *mask) fill(chars string) (string, bool) {
	cells := make([]rune, len(m.slots))
	j := m.next(0)
	for _, r := range chars {
		if j < 0 {
			return m.display(), false
		}
		c := m.fit(j, r)
		if c == 0 {
			return m.display(), false
		}
		cells[j] = c
		j = m.next(j + 1)
	}
	return m.render(cells), true
}

// withPrompt returns text, shown with the prompt old, shown with the prompt r.
func (m *mask) withPrompt(text string, r rune) string {
	cells := m.cells(text)
	m.prompt = r
	return m.render(cells)
}

// literals returns text with its prompts as spaces.
func (m *mask) literals(text string) string {
	out := []rune(text)
	for i, s := range m.slots {
		if s.editable() && i < len(out) && out[i] == m.prompt {
			out[i] = ' '
		}
	}
	return string(out)
}
