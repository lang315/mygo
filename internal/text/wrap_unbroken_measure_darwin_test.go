//go:build darwin && !ios

package text

import (
	"os"
	"slices"
	"testing"
	"time"
)

// TestCoreTextWrapUnbrokenMeasure logs what wrapping one paragraph without a
// break opportunity costs Core Text: the calls made and the time inside each,
// as the run grows. It measures, and asserts nothing on time; it runs only
// when MYGO_CORETEXT_MEASURE is set: to 1 for every text, or to one's name.
func TestCoreTextWrapUnbrokenMeasure(t *testing.T) {
	if os.Getenv("MYGO_CORETEXT_MEASURE") == "" {
		t.Skip("set MYGO_CORETEXT_MEASURE to measure")
	}
	s := newSystem()
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.engine().(*coreText)
	style := Style{Size: 13}

	type call struct {
		start, length int
		d             time.Duration
	}
	// reads are the times from a line made to its release: e.line, which
	// reads its runs.
	var suggests, lines, reads []call
	var creates []time.Duration
	suggest, createLine, create, release := ct.suggestLineBreak, ct.typesetterCreateLine, ct.typesetterCreate, ct.release
	defer func() {
		ct.suggestLineBreak, ct.typesetterCreateLine, ct.typesetterCreate, ct.release = suggest, createLine, create, release
	}()
	ct.suggestLineBreak = func(typesetter uintptr, start int, width float64) int {
		t0 := time.Now()
		n := suggest(typesetter, start, width)
		suggests = append(suggests, call{start, n, time.Since(t0)})
		return n
	}
	var made uintptr
	var madeAt time.Time
	var madeRange cfRange
	ct.typesetterCreateLine = func(typesetter uintptr, r cfRange) uintptr {
		t0 := time.Now()
		l := createLine(typesetter, r)
		lines = append(lines, call{r.location, r.length, time.Since(t0)})
		made, madeAt, madeRange = l, time.Now(), r
		return l
	}
	ct.release = func(obj uintptr) {
		if obj != 0 && obj == made {
			reads = append(reads, call{madeRange.location, madeRange.length, time.Since(madeAt)})
			made = 0
		}
		release(obj)
	}
	ct.typesetterCreate = func(str uintptr) uintptr {
		t0 := time.Now()
		ts := create(str)
		creates = append(creates, time.Since(t0))
		return ts
	}

	texts := []struct {
		name string
		rtl  bool
		r    func(i int) rune
	}{
		{"ascii", false, func(i int) rune { return 'a' + rune(i%26) }},
		{"cjk", false, func(i int) rune { return 0x4E00 + rune(i%512) }},
		{"arabic", true, func(i int) rune {
			// The letters ب…غ and ف…ي, without the tatweel.
			letters := []rune("بتثجحخدذرزسشصضطظعغفقكلمنهوي")
			return letters[i%len(letters)]
		}},
		// Controls: the same texts with a space every 40 runes.
		{"ascii_", false, func(i int) rune {
			if i%40 == 39 {
				return ' '
			}
			return 'a' + rune(i%26)
		}},
		{"cjk_", false, func(i int) rune {
			if i%40 == 39 {
				return ' '
			}
			return 0x4E00 + rune(i%512)
		}},
		{"arab_", true, func(i int) rune {
			if i%40 == 39 {
				return ' '
			}
			letters := []rune("بتثجحخدذرزسشصضطظعغفقكلمنهوي")
			return letters[i%len(letters)]
		}},
	}
	sum := func(cs []call) (d time.Duration) {
		for _, c := range cs {
			d += c.d
		}
		return d
	}
	// mean is the mean time of the calls of a tenth of the lines, the
	// calls ordered by start; a call over the whole text is left out.
	mean := func(cs []call, n int, last bool) time.Duration {
		var per []call
		for _, c := range cs {
			if c.length < n {
				per = append(per, c)
			}
		}
		if len(per) == 0 {
			return 0
		}
		k := max(len(per)/10, 1)
		if last {
			per = per[len(per)-k:]
		} else {
			per = per[:k]
		}
		return sum(per) / time.Duration(len(per))
	}
	median := func(ds []time.Duration) time.Duration {
		ds = slices.Clone(ds)
		slices.Sort(ds)
		return ds[len(ds)/2]
	}
	const runs = 5
	only := os.Getenv("MYGO_CORETEXT_MEASURE") // a text's name, or 1 for all
	for _, tx := range texts {
		if only != "1" && tx.name != only {
			continue
		}
		// A width of about 80 runes, from the advance of 800 of them.
		probe := make([]rune, 800)
		for i := range probe {
			probe[i] = tx.r(i)
		}
		e.resetScratch()
		var w float32
		for _, l := range e.shapeText(probe, style, nil, 0, tx.rtl, false) {
			for _, r := range l.runs {
				for _, g := range r.glyphs {
					w += g.Advance
				}
			}
		}
		width := w / 10
		var whole []time.Duration
		for _, n := range []int{8192, 32768, 131072} {
			text := make([]rune, n)
			for i := range text {
				text[i] = tx.r(i)
			}
			var total, inSuggest, inLine, inRead, inCreate []time.Duration
			var made, nSuggest, nLine, nCreate int
			var sFirst, sLast, lFirst, lLast, rFirst, rLast []time.Duration
			for run := range runs {
				for k, kt := range e.typesetters {
					e.dropTypesetter(k, kt)
				}
				for _, fonts := range []map[uintptr]*wrapFont{e.wrapFonts, e.laidFonts} {
					for _, f := range fonts {
						f.checked, f.lines = wrapCheckedFirst, 1 // no check by Core Text
					}
				}
				e.resetScratch()
				suggests, lines, reads, creates = suggests[:0], lines[:0], reads[:0], creates[:0]
				t0 := time.Now()
				got := e.shapeText(text, style, nil, width, tx.rtl, false)
				total = append(total, time.Since(t0))
				inSuggest = append(inSuggest, sum(suggests))
				inLine = append(inLine, sum(lines))
				inRead = append(inRead, sum(reads))
				var c time.Duration
				for _, d := range creates {
					c += d
				}
				inCreate = append(inCreate, c)
				sFirst, sLast = append(sFirst, mean(suggests, n, false)), append(sLast, mean(suggests, n, true))
				lFirst, lLast = append(lFirst, mean(lines, n, false)), append(lLast, mean(lines, n, true))
				rFirst, rLast = append(rFirst, mean(reads, n, false)), append(rLast, mean(reads, n, true))
				if run == runs-1 {
					made, nSuggest, nLine, nCreate = len(got), len(suggests), len(lines), len(creates)
				}
			}
			whole = append(whole, median(total))
			t.Logf("%-6s n=%6d width=%.0f lines=%d calls suggest/line/create=%d/%d/%d | median wrap=%v suggest=%v line=%v e.line=%v create=%v | per call first/last tenth: suggest %v/%v line %v/%v e.line %v/%v",
				tx.name, n, width, made, nSuggest, nLine, nCreate,
				median(total), median(inSuggest), median(inLine), median(inRead), median(inCreate),
				median(sFirst), median(sLast), median(lFirst), median(lLast), median(rFirst), median(rLast))
		}
		t.Logf("%-6s t(131072)/t(8192) = %.1f", tx.name, float64(whole[2])/float64(whole[0]))
	}
}
