package richtext

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPastedHTMLImageBudget(t *testing.T) {
	// 5792² pixels is just under ui.MaxImagePixels; four of them fill the
	// budget.
	const side = 5792
	if side*side > ui.MaxImagePixels || (side+1)*(side+1) <= ui.MaxImagePixels {
		t.Fatalf("%d² is not just under the cap", side)
	}
	var buf bytes.Buffer
	m := image.NewPaletted(image.Rect(0, 0, side, side), color.Palette{color.Black, color.White})
	if err := png.Encode(&buf, m); err != nil {
		t.Fatal(err)
	}
	src := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	html := strings.Repeat(`<img src="`+src+`">`, 8)

	var decoded, refused int
	for _, p := range ParseHTML(html).Paragraphs() {
		for _, r := range p.Runs {
			at := r.Attributes.Attachment
			if at == nil {
				continue
			}
			switch {
			case at.Image != nil && at.URL == "":
				decoded++
			case at.Image == nil && at.URL == src:
				refused++ // as an image too large to decode
			default:
				t.Errorf("an image neither decoded nor refused: %+v", at)
			}
		}
	}
	if decoded != 4 || refused != 4 {
		t.Errorf("%d images decoded, %d refused; want 4 and 4", decoded, refused)
	}

	// A picture cut short fails only once its pixels are allocated: it
	// takes its share of the budget, so eight cost what four do.
	cut := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()[:buf.Len()/2])
	parse := func(n int) uint64 {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		ParseHTML(strings.Repeat(`<img src="`+cut+`">`, n))
		runtime.ReadMemStats(&after)
		return after.TotalAlloc - before.TotalAlloc
	}
	if four, eight := parse(4), parse(8); eight > four+four/2 {
		t.Errorf("8 pictures cut short allocated %d MB, 4 %d MB", eight>>20, four>>20)
	}
}

// rtfAlloc returns the bytes ParseRTF allocates for src.
func rtfAlloc(src string) uint64 {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	ParseRTF([]byte(src))
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func TestRTFLinearFieldAndDepth(t *testing.T) {
	// A field's instruction of n bytes, read a character at a time.
	field := func(n int) string {
		return `{\rtf1{\field{\*\fldinst HYPERLINK "` + strings.Repeat("a\n", n/2) + `"}{\fldrslt x}}}`
	}
	small, large := rtfAlloc(field(100_000)), rtfAlloc(field(400_000))
	if large > 8*small {
		t.Errorf("a field of 100 KB allocated %d bytes, of 400 KB %d", small, large)
	}

	r := rtfReader{s: strings.Repeat("{", 1<<20), uc: 1, state: rtfState{uc: 1}}
	r.read()
	if len(r.stack) > 256 {
		t.Errorf("1 MB of { made a stack %d deep", len(r.stack))
	}
	// Groups past the cap close without closing those they are in: bold
	// set 100 deep holds until its group closes.
	src := `{\rtf1 ` + strings.Repeat("{", 100) + `\b ` + strings.Repeat("{", 200) + "a" + strings.Repeat("}", 200) + "b" + strings.Repeat("}", 100) + "c}"
	var got []Paragraph
	for _, p := range ParseRTF([]byte(src)).Paragraphs() {
		got = append(got, p)
	}
	if len(got) != 1 || len(got[0].Runs) != 2 || got[0].Runs[0].Text != "ab" || !got[0].Runs[0].Attributes.Bold() || got[0].Runs[1].Text != "c" || got[0].Runs[1].Attributes.Bold() {
		t.Errorf("groups 300 deep read as %+v", got)
	}
}
