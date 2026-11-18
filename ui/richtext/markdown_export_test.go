package richtext

import (
	"math/rand/v2"
	"runtime"
	"strings"
	"testing"
)

func parasOf(t Text) []Paragraph {
	var ps []Paragraph
	for _, p := range t.Paragraphs() {
		ps = append(ps, p)
	}
	return ps
}

// An image's alt and a link's URL with blank lines in them don't break
// the paragraph out as Markdown: it reads back as one paragraph.
func TestMarkdownExportAltAndURLNewlines(t *testing.T) {
	for _, r := range []Run{
		{Text: string(ObjectReplacement), Attributes: Attributes{Attachment: &Attachment{Name: "a\n\n# H", URL: "cat.png"}}},
		{Text: "x", Attributes: Attributes{Link: "https://e.org/a\n\n# H"}},
		{Text: "x", Attributes: Attributes{Link: "https://e.org/a\r\n\r\n# H"}},
		{Text: "x", Attributes: Attributes{Link: "https://e.org/a\tb"}},
	} {
		md := TextOf(Paragraph{Runs: []Run{{Text: "see "}, r, {Text: " end"}}}).Markdown()
		got := ParseMarkdown(md)
		ps := parasOf(got)
		if len(ps) != 1 || r.Attributes.Link != "" && len(ps[0].Runs) != 3 {
			t.Errorf("%q reads as\n%s", md, dump(got))
		}
	}
}

// A long paragraph reads in memory linear in its length: four times the
// text allocates less than eight times the bytes.
func TestMarkdownImportLongParagraphLinear(t *testing.T) {
	alloc := func(n int) uint64 {
		src := strings.Repeat("word\n", n/5)
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		ParseMarkdown(src)
		runtime.ReadMemStats(&after)
		return after.TotalAlloc - before.TotalAlloc
	}
	small, big := alloc(110_000), alloc(440_000)
	if big >= 8*small {
		t.Errorf("%d bytes for 440 KB, %d for 110 KB: not linear", big, small)
	}
}

// A list item indented deeper than the item before it, or than none,
// stays an item: Markdown would read it as code.
func TestMarkdownExportOrphanIndentedItem(t *testing.T) {
	item := Paragraph{Style: ParagraphStyle{List: Bullets, Indent: 1}, Runs: []Run{{Text: "x"}}}
	for _, paras := range [][]Paragraph{
		{item},
		{{Runs: []Run{{Text: "intro"}}}, item},
		{{Style: ParagraphStyle{List: Bullets}, Runs: []Run{{Text: "a"}}}, {Style: ParagraphStyle{List: Bullets, Indent: 3}, Runs: []Run{{Text: "x"}}}},
	} {
		md := TextOf(paras...).Markdown()
		got := ParseMarkdown(md)
		ps := parasOf(got)
		last := ps[len(ps)-1]
		if len(ps) != len(paras) || last.Style.List != Bullets || last.Style.CodeBlock || last.Runs[0].Text != "x" {
			t.Errorf("%q reads\n%s", md, dump(got))
		}
	}
}

// URLs in text are written as they are, their underscores unescaped, and
// read back the same.
func TestMarkdownExportURLUnderscores(t *testing.T) {
	for _, s := range []string{"see https://example.com/a_b_c", "www.x.com/a_b", "snake_case_name", "at https://x.com/~u/_p*q."} {
		md := TextOf(Paragraph{Runs: []Run{{Text: s}}}).Markdown()
		if md != s {
			t.Errorf("%q written %q", s, md)
		}
		if got := ParseMarkdown(md).String(); got != s {
			t.Errorf("%q reads back %q", s, got)
		}
	}
}

// readsBackAs reports whether md reads as the one paragraph of want, its
// text and marks, links GitHub's Markdown makes of URLs aside.
func readsBackAs(md string, want Text) bool {
	got := parasOf(ParseMarkdown(md))
	ws := parasOf(want)
	if len(got) != len(ws) {
		return false
	}
	if len(got) == 0 {
		return true
	}
	return sameMarkRuns(got[0].Runs, ws[0].Runs)
}

// What looks like a URL but isn't linked, a URL in a link's text or an
// image's alt, and one right after a mark keep their text and marks.
func TestMarkdownExportURLsKeepText(t *testing.T) {
	bold := Attributes{Weight: 700}
	cases := [][]Run{
		{{Text: "https://x.com/a_b*c*", Attributes: Attributes{Link: "https://x.com/a_b*c*"}}},
		{{Text: string(ObjectReplacement), Attributes: Attributes{Attachment: &Attachment{Name: "https://x/a]b", URL: "a.png"}}}},
		{{Text: "f()", Attributes: Attributes{Code: true}}, {Text: "https://x.com/*a*"}},
		{{Text: "b", Attributes: bold}, {Text: "www.x.com/_a_"}},
		{{Text: "u", Attributes: Attributes{Underline: true}}, {Text: " www.x.com/*a*"}},
		{{Text: "x "}, {Text: "www.x.com/*a*", Attributes: Attributes{Underline: true}}},
	}
	for _, s := range []string{"https://*a*", "www.*a*", "www._a_", "x https://*a* y",
		"https://x.com/a_", "https://x.com/a*", "https://x.com/a~", "www.x_y.com/*a*", "https://x.com/[a](b)", "https://x.com/`a`"} {
		cases = append(cases, []Run{{Text: s}})
	}
	for _, runs := range cases {
		want := TextOf(Paragraph{Runs: runs})
		if md := want.Markdown(); !readsBackAs(md, want) {
			t.Errorf("%q written %q reads\n%s", want.String(), md, dump(ParseMarkdown(md)))
		}
	}
	// A URL not starting a word links with the escapes written in it, as
	// before: what read back escaped still does.
	rng := rand.New(rand.NewPCG(3, 4))
	words := []string{"https://", "http://", "www.", "x.com", "a_b", "/", "*", "_", "~", "a", "(", ")", "[", "]", "<", ".", "\\", "`", "&amp;", "!", " "}
	failures := 0
	for range 3000 {
		var runs []Run
		for range 1 + rng.IntN(4) {
			var s strings.Builder
			for k := range 1 + rng.IntN(4) {
				if k > 0 && rng.IntN(3) == 0 {
					s.WriteByte(' ')
				}
				s.WriteString(words[rng.IntN(len(words))])
			}
			var a Attributes
			switch rng.IntN(6) {
			case 0:
				a.Weight = 700
			case 1:
				a.Underline = true
			case 2:
				a.Link = "https://e.org/"
			}
			runs = append(runs, Run{Text: s.String(), Attributes: a})
		}
		want := TextOf(normalize(TextOf(Paragraph{Runs: runs}))...)
		ps := parasOf(want)
		if len(ps) == 0 {
			continue
		}
		escaped := markdownRuns(ps[0].Runs, false)
		if md := want.Markdown(); readsBackAs(escaped, want) && !readsBackAs(md, want) {
			failures++
			if failures <= 10 {
				t.Errorf("%q written %q reads\n%s", want.String(), md, dump(ParseMarkdown(md)))
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of 3000 failed", failures)
	}
}

// Writing a document as Markdown, reading it and writing it again gives
// the same Markdown.
func TestMarkdownRoundTripStable(t *testing.T) {
	bold, italic := Attributes{Weight: 700}, Attributes{Italic: true}
	para := func(st ParagraphStyle, runs ...Run) Paragraph { return Paragraph{Style: st, Runs: runs} }
	doc := TextOf(
		para(ParagraphStyle{Heading: 1}, Run{Text: "Title"}),
		para(ParagraphStyle{}, Run{Text: "Some "}, Run{Text: "bold", Attributes: bold}, Run{Text: " and "},
			Run{Text: "italic", Attributes: italic}, Run{Text: ", a "}, Run{Text: "link", Attributes: Attributes{Link: "https://e.org/a_b?q=(1)"}},
			Run{Text: " and "}, Run{Text: string(ObjectReplacement), Attributes: Attributes{Attachment: &Attachment{Name: "a cat", URL: "cat.png"}}},
			Run{Text: " in some_dir."}),
		para(ParagraphStyle{List: Bullets}, Run{Text: "one"}),
		para(ParagraphStyle{List: Bullets, Indent: 1}, Run{Text: "nested "}, Run{Text: "two", Attributes: bold}),
		para(ParagraphStyle{List: Numbers, Indent: 2}, Run{Text: "three"}),
		para(ParagraphStyle{List: Tasks, Checked: true}, Run{Text: "done"}),
		para(ParagraphStyle{Quote: true}, Run{Text: "quoted snake_case"}),
		para(ParagraphStyle{CodeBlock: true}, Run{Text: "x := a_b * 2"}),
	)
	md := doc.Markdown()
	if again := ParseMarkdown(md).Markdown(); again != md {
		t.Errorf("written\n%s\nthen\n%s", md, again)
	}
}
