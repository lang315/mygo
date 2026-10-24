package widget

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

var (
	badgeGreen = ui.RGB(0, 160, 0)
	badgeWhite = ui.RGB(255, 255, 255)
)

// badgeTable is a table of ann, bob and cat whose Name cells draw a green
// badge "Done" before the name, or the plain name when plain.
func badgeTable(plain bool, tail ...ui.Color) (*DataGridViewWidget[acct, int], *ui.Tester) {
	col := Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 300, Sortable: true, Filterable: true}
	if !plain {
		col.Spans = func(a *acct) []ui.Span {
			return []ui.Span{
				{Text: "Done", Color: badgeWhite, Background: badgeGreen, Radius: 4, Weight: 800},
				{Text: " " + a.name, Color: append(tail, ui.Color{})[0]},
			}
		}
	}
	tb := DataGridView(func(a *acct) int { return a.id }, col).
		SetRows([]acct{{1, "ann", 1}, {2, "bob", 2}, {3, "cat", 3}}).Expand()
	return tb, spansTester(tb)
}

func spansTester(tb Control) *ui.Tester {
	tt := ui.NewTester(content(tb), 800, 600)
	tt.Frame()
	return tt
}

// ink is the pixel of r farthest from bg: the core of the glyphs drawn.
func inkOf(img *image.RGBA, r ui.Rect, bg color.RGBA) color.RGBA {
	best, far := bg, -1
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := img.RGBAAt(x, y)
			d := absInt(int(c.R)-int(bg.R)) + absInt(int(c.G)-int(bg.G)) + absInt(int(c.B)-int(bg.B))
			if d > far {
				best, far = c, d
			}
		}
	}
	return best
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// badgeColor is the color most of the badge of the rich cell b is made of:
// its background, whichever way a platform blends the glyphs' edges.
func badgeColor(img *image.RGBA, b ui.Rect) color.RGBA {
	count := map[color.RGBA]int{}
	best, most := color.RGBA{}, 0
	for y := int(b.Y); y < int(b.Y+b.H); y++ {
		for x := int(b.X); x < int(b.X)+30; x++ {
			c := img.RGBAAt(x, y)
			if count[c]++; count[c] > most {
				best, most = c, count[c]
			}
		}
	}
	return best
}

// afterBadge is the last letters of the rich cell "Done ann" r, past the badge.
func afterBadge(t *testing.T, tt *ui.Tester, badge ui.Rect) ui.Rect {
	t.Helper()
	return ui.Rect{X: badge.X + badge.W - 20, Y: badge.Y, W: 18, H: badge.H}
}

func TestSpansBadgeBackgroundIsGreen(t *testing.T) {
	_, tt := badgeTable(false)
	b := find(t, tt, "Done ann")
	if c := badgeColor(tt.Image(), b); !near(c, badgeGreen) {
		t.Fatalf("badge background %v, want green", c)
	}
}

func TestSpansTextAfterTheBadgeIsInTheRowColor(t *testing.T) {
	_, tt := badgeTable(false)
	_, plain := badgeTable(true)
	b := find(t, tt, "Done ann")
	got := inkOf(tt.Image(), afterBadge(t, tt, b), page(tt))
	want := inkOf(plain.Image(), ui.Rect{X: b.X, Y: b.Y, W: 18, H: b.H}, page(plain)) // the plain cell's "ann"
	if !near(got, ui.RGB(want.R, want.G, want.B)) {
		t.Fatalf("text after the badge is %v, the plain cell's is %v", got, want)
	}
}

func TestSpansSelectedRowKeepsTheBadgeAndTakesTheSelectionTextColor(t *testing.T) {
	_, tt := badgeTable(false, red) // a span with a color of its own
	_, plain := badgeTable(true)
	click(t, tt, "ann")
	click(t, plain, "ann")
	b := find(t, tt, "Done ann")
	if c := badgeColor(tt.Image(), b); !near(c, badgeGreen) {
		t.Fatalf("selected row's badge %v, want green", c)
	}
	bg := plain.Image().RGBAAt(700, int(b.Y+b.H/2)) // the selection's fill
	got := inkOf(tt.Image(), afterBadge(t, tt, b), bg)
	want := inkOf(plain.Image(), ui.Rect{X: b.X, Y: b.Y, W: 18, H: b.H}, bg)
	if !near(got, ui.RGB(want.R, want.G, want.B)) {
		t.Fatalf("selected text after the badge is %v, the plain selected cell's is %v", got, want)
	}
}

func TestSpansSortFilterAndCopyFollowText(t *testing.T) {
	tb, tt := badgeTable(false)
	click(t, tt, "Name")
	click(t, tt, "Name") // descending by Text: cat, bob, ann
	inOrder(t, tt, "Done cat", "Done bob", "Done ann")
	tb.Search("Name", "bo") // the badge's "Done" has no "bo": Text is "bob"
	tt.Frame()
	if tt.HasText("Done ann") || !tt.HasText("Done bob") {
		t.Fatalf("search by Text: %q", tt.Texts())
	}
	tb.Search("Name", "Done") // the badge text is not Text
	tt.Frame()
	if tt.HasText("Done bob") {
		t.Fatalf("search matched the badge's text: %q", tt.Texts())
	}
	tb.Search("Name", "")
	tt.Frame()
	click(t, tt, "Done bob")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "bob" {
		t.Fatalf("copied %q, want Text", got)
	}
}

func TestNilSpansIsTheTextCell(t *testing.T) {
	_, tt := badgeTable(true)
	if !slices.Contains(tt.Texts(), "ann") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

// BenchmarkTableSpansFrame builds an idle frame of a table of 100,000 rows
// whose first column shows the name as a plain text, or as a badge and the
// name in spans.
func BenchmarkTableSpansFrame(b *testing.B) {
	rows := manyAccts(100_000)
	for _, spans := range []bool{false, true} {
		name := "text"
		if spans {
			name = "spans"
		}
		b.Run(name, func(b *testing.B) {
			col := Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 300, Sortable: true}
			if spans {
				// The app's own allocations are the fixture's: spans made
				// once per row, as an app keeping them with its rows has.
				made := make([][]ui.Span, len(rows))
				for i := range rows {
					made[i] = []ui.Span{{Text: "Done", Color: badgeWhite, Background: badgeGreen, Radius: 4, Weight: 800}, {Text: " " + rows[i].name}}
				}
				col.Spans = func(a *acct) []ui.Span { return made[a.id] }
			}
			tb := DataGridView(func(a *acct) int { return a.id }, col).CheckboxColumn().SetRows(rows)
			tt := ui.NewTester(content(tb), 400, 600)
			for range 10 {
				tt.Frame()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				tt.Frame()
			}
		})
	}
}
