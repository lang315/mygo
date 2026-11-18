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
// when MYGO_CORETEXT_MEASURE is set.
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
	var suggests, lines []call
	var creates []time.Duration
	suggest, createLine, create := ct.suggestLineBreak, ct.typesetterCreateLine, ct.typesetterCreate
	defer func() {
		ct.suggestLineBreak, ct.typesetterCreateLine, ct.typesetterCreate = suggest, createLine, create
	}()
	ct.suggestLineBreak = func(typesetter uintptr, start int, width float64) int {
		t0 := time.Now()
		n := suggest(typesetter, start, width)
		suggests = append(suggests, call{start, n, time.Since(t0)})
		return n
	}
	ct.typesetterCreateLine = func(typesetter uintptr, r cfRange) uintptr {
		t0 := time.Now()
		l := createLine(typesetter, r)
		lines = append(lines, call{r.location, r.length, time.Since(t0)})
		return l
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
	for _, tx := range texts {
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
			var total, inSuggest, inLine, inCreate []time.Duration
			var made, nSuggest, nLine, nCreate int
			var sFirst, sLast, lFirst, lLast time.Duration
			for run := range runs {
				for k, kt := range e.typesetters {
					e.dropTypesetter(k, kt)
				}
				for _, f := range e.wrapFonts {
					f.checked, f.lines = wrapCheckedFirst, 1 // no check by Core Text
				}
				e.resetScratch()
				suggests, lines, creates = suggests[:0], lines[:0], creates[:0]
				t0 := time.Now()
				got := e.shapeText(text, style, nil, width, tx.rtl, false)
				total = append(total, time.Since(t0))
				inSuggest = append(inSuggest, sum(suggests))
				inLine = append(inLine, sum(lines))
				var c time.Duration
				for _, d := range creates {
					c += d
				}
				inCreate = append(inCreate, c)
				if run == runs-1 {
					made, nSuggest, nLine, nCreate = len(got), len(suggests), len(lines), len(creates)
					sFirst, sLast = mean(suggests, n, false), mean(suggests, n, true)
					lFirst, lLast = mean(lines, n, false), mean(lines, n, true)
				}
			}
			whole = append(whole, median(total))
			t.Logf("%-6s n=%6d width=%.0f lines=%d calls suggest/line/create=%d/%d/%d | median wrap=%v suggest=%v line=%v create=%v | per call first/last tenth: suggest %v/%v line %v/%v",
				tx.name, n, width, made, nSuggest, nLine, nCreate,
				median(total), median(inSuggest), median(inLine), median(inCreate),
				sFirst, sLast, lFirst, lLast)
		}
		t.Logf("%-6s t(131072)/t(8192) = %.1f", tx.name, float64(whole[2])/float64(whole[0]))
	}
}
