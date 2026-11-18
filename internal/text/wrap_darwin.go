package text

import "unicode"

// Core Text's wrapping from one line (wrap.go). Core Text finds line breaks
// a line at a time, opening a line for each and shaping it into glyphs
// through a call for each of its properties. But a line of Latin text is a run
// of the glyphs of the whole text, moved back to the start of the line. So a
// text kept across widths (keptTypesetter) is shaped once into one line, whose
// glyphs its lines take, as GPUI does.
//
// Lines break where Core Text surely would: at the spaces before plain words.
// What could tell them apart goes to Core Text, line by line, or whole: all but
// plain Latin text, and a line ending where the next word does not fit and may
// break inside itself, or is too long for a line and holds more than letters
// and digits. And the first texts of a font, and one in every few hundred
// after them, are also wrapped by Core Text: a font where the two differ is
// left to it from then on.
//
// Other text, as Arabic or Hebrew, and text in a right-to-left paragraph, is
// laid out by Core Text a line at a time, in its order and with its runs. But
// its lines break where they would by the advances of its one line, summed by
// rune: Core Text finding a break costs more the further it is in the text.

// shapeWhole shapes a kept text into one line, or returns nil if it does not
// come out as one run with a glyph for each rune.
func (e *coreText) shapeWhole(typesetter uintptr, text []rune, italic bool) *shapedWhole {
	n := len(text)
	line := ct.typesetterCreateLine(typesetter, cfRange{0, n})
	if line == 0 {
		return nil
	}
	defer ct.release(line)
	runs := ct.lineGetGlyphRuns(line)
	if ct.arrayGetCount(runs) != 1 {
		return nil
	}
	run := ct.arrayGetValueAtIndex(runs, 0)
	if ct.runGetGlyphCount(run) != n || ct.runGetStatus(run)&ctRunStatusRightToLeft != 0 {
		return nil
	}
	attrs := ct.runGetAttributes(run)
	w := &shapedWhole{
		font:     e.runFont(ct.dictionaryGetValue(attrs, ct.fontAttributeName), italic || ct.dictionaryGetValue(attrs, e.italic) != 0),
		ids:      make([]uint32, n),
		advances: make([]float32, n),
		pens:     make([]float64, n+1),
	}
	e.runIDs = growTo(e.runIDs, n)
	e.runPoints = growTo(e.runPoints, n)
	e.runAdvances = growTo(e.runAdvances, n)
	e.runIndices = growTo(e.runIndices, n)
	all := cfRange{}
	ct.runGetGlyphs(run, all, &e.runIDs[0])
	ct.runGetPositions(run, all, &e.runPoints[0])
	ct.runGetAdvances(run, all, &e.runAdvances[0])
	ct.runGetStringIndices(run, all, &e.runIndices[0])
	for i := range n {
		if e.runIndices[i] != i || e.runPoints[i].y != 0 {
			return nil
		}
		w.ids[i] = uint32(e.runIDs[i])
		w.advances[i] = float32(e.runAdvances[i].w)
		w.pens[i] = e.runPoints[i].x
	}
	w.pens[n] = w.pens[n-1] + e.runAdvances[n-1].w
	return w
}

// laidText reports whether text is of runes whose lines wrapKept may break
// in Go when Core Text lays them out: plain Latin ones, and the letters,
// marks and digits of Arabic, Persian and Hebrew, Arabic punctuation and the
// zero-width non-joiner. A line breaks before a word of them only after
// spaces.
func laidText(text []rune) bool {
	for _, r := range text {
		switch {
		case r >= 0x20 && r <= 0x7E, r >= 0xC0 && r <= 0xFF:
		case r == 0x060C, r == 0x061B, r == 0x061F, r >= 0x0621 && r <= 0x0669, r >= 0x0670 && r <= 0x06D3:
		case r >= 0x05D0 && r <= 0x05EA, r == 0x200C:
		default:
			return false
		}
	}
	return true
}

// measureWhole measures a kept text whose lines Core Text lays out from its
// one line: shaped with only the pens before each rune and at its end, the
// advances of the glyphs summed by rune in the text's order, whatever the
// runs and their directions. It returns nil for a text not laidText.
func (e *coreText) measureWhole(typesetter uintptr, text []rune) *shapedWhole {
	if !laidText(text) {
		return nil
	}
	n := len(text)
	line := ct.typesetterCreateLine(typesetter, cfRange{0, n})
	if line == 0 {
		return nil
	}
	defer ct.release(line)
	w := &shapedWhole{pens: make([]float64, n+1)}
	runs := ct.lineGetGlyphRuns(line)
	for i := range ct.arrayGetCount(runs) {
		run := ct.arrayGetValueAtIndex(runs, i)
		count := ct.runGetGlyphCount(run)
		if count == 0 {
			continue
		}
		e.runAdvances = growTo(e.runAdvances, count)
		e.runIndices = growTo(e.runIndices, count)
		ct.runGetAdvances(run, cfRange{}, &e.runAdvances[0])
		ct.runGetStringIndices(run, cfRange{}, &e.runIndices[0])
		// A text of BMP runes: its UTF-16 units are its runes.
		for j := range count {
			w.pens[e.runIndices[j]+1] += e.runAdvances[j].w
		}
	}
	for i := range n {
		w.pens[i+1] += w.pens[i]
	}
	return w
}

// wrapKept wraps a kept text from its one line, or returns nil for Core Text to
// wrap it. The lines of a text kept.laid are laid out by Core Text, a line at a
// time, from where they break.
func (e *coreText) wrapKept(kept *keptTypesetter, text []rune, style Style, width float32) []shapedLine {
	fonts := e.wrapFonts
	if kept.laid {
		fonts = e.laidFonts
	}
	f := fonts[kept.font]
	if f == nil {
		f = &wrapFont{}
		fonts[kept.font] = f
	}
	if f.disagreed {
		return nil
	}
	if !kept.shaped {
		kept.shaped = true
		if kept.laid {
			kept.whole = e.measureWhole(kept.typesetter, text)
		} else {
			kept.whole = e.shapeWhole(kept.typesetter, text, style.Italic)
		}
	}
	w := kept.whole
	if w == nil {
		return nil
	}
	if !f.spaceKnown {
		f.spaceKnown = true
		f.space = -1
		if run := oneRun(e.shapeText([]rune{' '}, Style{Size: style.Size, Family: style.Family, Weight: style.Weight, Italic: style.Italic}, nil, 0, false, false)); run != nil && len(run.glyphs) == 1 {
			f.space = run.glyphs[0].Advance
		}
	}
	if f.space < 0 {
		return nil
	}
	n := len(text)
	limit := float64(width) + 1.0/1024
	var lines []shapedLine
	mark := len(e.shapeScratch.lines)
	// The word walked last: the lines Core Text breaks a word too long for a
	// line into start inside it, and take its end without walking it again,
	// so that a paragraph without spaces wraps in time linear in its length.
	wordFrom, wordTo := 0, 0
	// alnum tells that the word from alnumFrom holds only letters and digits.
	alnumFrom, alnum := -1, false
	indexed := false
	for start := 0; start < n; {
		end, pen := start, w.pens[start]
		for i := start; i < n; {
			// A word and the spaces after it, which take no room.
			if i >= wordFrom && i < wordTo {
				i = wordTo
			} else {
				wordFrom = i
				for i < n && text[i] != ' ' {
					i++
				}
				wordTo = i
				e.wrapSteps += uint64(wordTo - wordFrom)
			}
			if w.pens[i]-pen > limit {
				break
			}
			for i < n && text[i] == ' ' {
				i++
			}
			end = i
		}
		// A line ending before n ends where the next word, from end to
		// wordTo, does not fit.
		plain := end == n || end == start || plainWord(text[end:wordTo])
		if end > start && end < n {
			e.wrapSteps += uint64(wordTo - end)
		}
		// A word of letters and digits too long for a line breaks after
		// the last letter that fits, at least one, as Core Text breaks it.
		cut := false
		if end == start && !kept.laid {
			if alnumFrom != wordFrom {
				alnumFrom, alnum = wordFrom, true
				for _, r := range text[wordFrom:wordTo] {
					alnum = alnum && (unicode.IsLetter(r) || unicode.IsDigit(r))
				}
				e.wrapSteps += uint64(wordTo - wordFrom)
			}
			if alnum {
				c := start + 1
				for c < wordTo && w.pens[c+1]-pen <= limit {
					c++
				}
				e.wrapSteps += uint64(c - start)
				// A last letter too wide for a line before spaces goes to
				// Core Text, which may hang the spaces on its line.
				if c < wordTo || c == n {
					end, cut = c, true
				}
			}
		}
		if end == start || !plain || kept.laid {
			count := end - start
			if end == start || !plain {
				// The first word does not fit, or the next may break inside.
				count = max(ct.suggestLineBreak(kept.typesetter, start, limit), 1)
			}
			line := ct.typesetterCreateLine(kept.typesetter, cfRange{start, count})
			if line == 0 {
				return nil
			}
			if !indexed {
				e.index = runeIndex(e.index, n)
				e.wrapSteps += uint64(n)
				indexed = true
			}
			lines = e.addLine(mark, e.line(line, e.index, n, style.Italic))
			ct.release(line)
			start += count
			continue
		}
		glyphs := e.glyphRoom(end - start)
		for i := start; i < end; i++ {
			glyphs[i-start] = Glyph{Font: w.font, ID: w.ids[i], X: float32(w.pens[i] - pen), Advance: w.advances[i], Cluster: i}
		}
		if end < n && !cut {
			glyphs[end-start-1].Advance = f.space
		}
		line := shapedLine{start: start, end: end, runs: e.addRun(len(e.shapeScratch.runs), shapedRun{font: w.font, start: start, end: end, glyphs: glyphs})}
		lines = e.addLine(mark, line)
		start = end
	}
	f.lines++
	if f.checked < wrapCheckedFirst || f.lines%wrapCheckedEvery == 0 {
		e.index = runeIndex(e.index, n)
		shaped := e.typesetLines(kept.typesetter, n, width, nil, false, false, e.index, style.Italic)
		if !sameLines(lines, shaped) {
			f.disagreed = true
			return shaped
		}
		f.checked++
	}
	return lines
}
