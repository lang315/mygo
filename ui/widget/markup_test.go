package widget

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestParseMarkupFhtoolStrings(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []ui.Span
	}{
		// fhtool's state.go: the Done and Error badges before a message.
		{`<span foreground='white' background='green' weight='800'>Done</span> msg`, []ui.Span{
			{Text: "Done", Color: ui.RGB(255, 255, 255), Background: ui.RGB(0, 255, 0), Weight: 800},
			{Text: " msg"},
		}},
		{`<span foreground='white' background='red' weight='800'>Error</span> checkpoint 282`, []ui.Span{
			{Text: "Error", Color: ui.RGB(255, 255, 255), Background: ui.RGB(255, 0, 0), Weight: 800},
			{Text: " checkpoint 282"},
		}},
		// The context menu's destructive items.
		{`<span foreground='red' weight='800'>Xóa 3 tài khoản</span>`, []ui.Span{
			{Text: "Xóa 3 tài khoản", Color: ui.RGB(255, 0, 0), Weight: 800},
		}},
		{`<span foreground='#B91C1C' weight='800'>Lọc trùng UID 2 tài khoản</span>`, []ui.Span{
			{Text: "Lọc trùng UID 2 tài khoản", Color: ui.RGB(0xB9, 0x1C, 0x1C), Weight: 800},
		}},
		{`<span foreground='yellow' background='black' weight='800'>Lưu ý</span>`, []ui.Span{
			{Text: "Lưu ý", Color: ui.RGB(255, 255, 0), Background: ui.RGB(0, 0, 0), Weight: 800},
		}},
		{`<b>Tên:</b> <i>Ada</i> <u>x</u>`, []ui.Span{
			{Text: "Tên:", Weight: 700}, {Text: " "}, {Text: "Ada", Italic: true}, {Text: " "}, {Text: "x", Underline: true},
		}},
		{`plain`, []ui.Span{{Text: "plain"}}},
		{``, nil},
	} {
		if got := ParseMarkup(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseMarkup(%q)\n got %+v\nwant %+v", c.in, got, c.want)
		}
	}
}

func TestParseMarkupNestingAndWeights(t *testing.T) {
	got := ParseMarkup(`<span weight="bold" foreground="#f00">a<i>b</i><span foreground="blue">c</span>d</span>`)
	want := []ui.Span{
		{Text: "a", Weight: 700, Color: ui.RGB(255, 0, 0)},
		{Text: "b", Weight: 700, Color: ui.RGB(255, 0, 0), Italic: true},
		{Text: "c", Weight: 700, Color: ui.RGB(0, 0, 255)},
		{Text: "d", Weight: 700, Color: ui.RGB(255, 0, 0)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseMarkupEntities(t *testing.T) {
	got := ParseMarkup(`a &amp; b &lt;c&gt; &apos;d&apos; &quot;e&quot; &unknown; &#65;`)
	if len(got) != 1 || got[0].Text != `a & b <c> 'd' "e" &unknown; A` {
		t.Fatalf("got %+v", got)
	}
}

func TestParseMarkupBadInputIsKeptAsTextAndNeverPanics(t *testing.T) {
	for _, in := range []string{
		`<`, `a < b`, `<span`, `<span foreground='red'`, `</span>`, `</b></b>`, `<b>unclosed`, `<span foreground='nope'>x</span>`,
		`<span weight='abc'>x</span>`, `<unknown>x</unknown>`, `<span foreground=>x</span>`, `<<b>>`, "<span\x00>", `&`, `&amp`, "\xff\xfe<b>",
		`<span foreground='#12'>x</span>`, `<span foreground='#GGGGGG'>x</span>`,
	} {
		spans := ParseMarkup(in)
		var text strings.Builder
		for _, s := range spans {
			text.WriteString(s.Text)
		}
		if in != "" && text.Len() == 0 && !strings.ContainsAny(in, "<&") {
			t.Errorf("ParseMarkup(%q) lost the text", in)
		}
	}
	// An unknown tag stays as text; an unclosed known one styles the rest.
	if got := ParseMarkup(`<unknown>x</unknown>`); len(got) != 1 || got[0].Text != "<unknown>x</unknown>" {
		t.Errorf("unknown tag: %+v", got)
	}
	if got := ParseMarkup(`<b>unclosed`); len(got) != 1 || got[0].Weight != 700 || got[0].Text != "unclosed" {
		t.Errorf("unclosed: %+v", got)
	}
	if got := ParseMarkup(`a < b`); len(got) != 1 || got[0].Text != "a < b" {
		t.Errorf("a stray <: %+v", got)
	}
	if got := ParseMarkup(`<span foreground='nope'>x</span>`); len(got) != 1 || got[0].Text != "x" || got[0].Color != (ui.Color{}) {
		t.Errorf("unknown color: %+v", got)
	}
}

func FuzzParseMarkup(f *testing.F) {
	for _, s := range []string{`<span foreground='white' background='green' weight='800'>Done</span> msg`, `<b>a</b>`, `&amp;`, `<`, `</`, `<span`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { ParseMarkup(s) })
}

func TestLabelRichTakesParsedMarkup(t *testing.T) {
	l := Label("").Rich(ParseMarkup(`<span foreground='white' background='green' weight='800'>Done</span> ok`)...)
	tt := tester(t, l)
	if l.Text() != "Done ok" || !tt.HasText("Done ok") {
		t.Fatalf("text %q, texts %q", l.Text(), tt.Texts())
	}
}

func TestParseMarkupUnknownTagsStayAsText(t *testing.T) {
	// fhtool's logger puts an unescaped message inside a span.
	for in, want := range map[string]string{
		"err: <nil>":                              "err: <nil>",
		"if x<y then <b>bold</b>":                 "if x<y then bold",
		"<span foreground='red'>a <nil> b</span>": "a <nil> b",
		"a </nil> b":                              "a </nil> b",
	} {
		var text strings.Builder
		for _, s := range ParseMarkup(in) {
			text.WriteString(s.Text)
		}
		if text.String() != want {
			t.Errorf("ParseMarkup(%q) text %q, want %q", in, text.String(), want)
		}
	}
	got := ParseMarkup("if x<y then <b>bold</b>")
	if len(got) != 2 || got[1].Weight != 700 || got[1].Text != "bold" {
		t.Errorf("the <b> after a stray < lost its style: %+v", got)
	}
}

func TestParseMarkupColorsWithAlpha(t *testing.T) {
	for in, want := range map[string]ui.Color{
		"#ff000080": {R: 255, A: 0x80},
		"#f008":     {R: 255, A: 0x88},
		"#102030":   {R: 0x10, G: 0x20, B: 0x30, A: 255},
	} {
		got := ParseMarkup("<span foreground='" + in + "'>x</span>")
		if len(got) != 1 || got[0].Color != want {
			t.Errorf("foreground %s: %+v, want %+v", in, got, want)
		}
	}
}

func TestParseMarkupWeightsAsPangoReadsThem(t *testing.T) {
	for in, want := range map[string]int{
		"1000": 1000, "ultraheavy": 1000, "book": 380, "semilight": 350, "100": 100, "heavy": 900,
	} {
		got := ParseMarkup("<span weight='" + in + "'>x</span>")
		if len(got) != 1 || got[0].Weight != want {
			t.Errorf("weight %s: %+v, want %d", in, got, want)
		}
	}
}
