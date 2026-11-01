package ui

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"
)

// richSpans are spans by what they paint: a rich text cell of every kind a
// table shows.
var richSpans = map[string][]Span{
	"color":      {{Text: "Done", Color: RGB(255, 0, 0)}, {Text: " ok"}},
	"weight":     {{Text: "Done", Weight: 800}, {Text: " ok"}},
	"background": {{Text: "Done", Background: RGB(0, 150, 0)}, {Text: " ok"}},
	"badge":      {{Text: "Done", Color: RGB(255, 255, 255), Background: RGB(0, 150, 0), Radius: 4, Weight: 800}, {Text: " ok"}},
	"underline":  {{Text: "Done", Underline: true, DecorationColor: RGB(0, 0, 200)}, {Text: " ok", Strikethrough: true}},
	"none":       {{Text: "Done ok"}},
}

// richVersion is the version of every row of richTable: a test that changes
// the spans of rows kept (RowVersion) bumps it.
var richVersion uint64 = 1

// richTable is a table whose second column is a rich text of spans(r); its
// rows are kept (RowVersion) or built anew each frame.
func richTable(spans func(r int) []Span, kept bool) (*Tester, *ListState) {
	var list ListState
	if kept {
		list.RowVersion = func(int) uint64 { return richVersion }
	}
	tt := NewTester(func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}, {Title: "B"}}, 200, func(r, col int) {
			if col == 0 {
				Textf(c, "row %d", r)
				return
			}
			RichText(c, spans(r)...)
		}).Grow(1)
	}, 400, 400)
	for range 4 {
		tt.Frame()
	}
	return tt, &list
}

// A row holding a rich text is replayed in an idle frame, as one holding a
// text is (retained rows).
func TestRowsWithRichTextReplay(t *testing.T) {
	for name, spans := range richSpans {
		for _, kept := range []bool{false, true} {
			t.Run(fmt.Sprint(name, " kept ", kept), func(t *testing.T) {
				tt, list := richTable(func(int) []Span { return spans }, kept)
				tt.rt.replayCheck = true
				first, last := list.Visible()
				fresh := tt.rt.freshRows
				tt.Frame()
				if n := tt.rt.freshRows - fresh; n != 0 {
					t.Fatalf("an idle frame painted %d of %d rows anew", n, last-first+1)
				}
				if tt.rt.replayed == 0 || tt.rt.replayMismatch != 0 {
					t.Fatalf("replayed %d rows, %d of them differ from painting them", tt.rt.replayed, tt.rt.replayMismatch)
				}
			})
		}
	}
}

// What a span paints, changed in one span, paints the row anew.
func TestRowsWithRichTextPaintAgainWhenASpanChanges(t *testing.T) {
	base := func() []Span {
		return []Span{{Text: "Done", Color: RGB(255, 0, 0), Background: RGB(0, 150, 0), Radius: 4}, {Text: " ok"}}
	}
	for name, change := range map[string]func(s []Span) []Span{
		"color":             func(s []Span) []Span { s[0].Color = RGB(0, 0, 255); return s },
		"background":        func(s []Span) []Span { s[0].Background = RGB(200, 0, 0); return s },
		"radius":            func(s []Span) []Span { s[0].Radius = 8; return s },
		"underline":         func(s []Span) []Span { s[1].Underline = true; return s },
		"wavy underline":    func(s []Span) []Span { s[1].WavyUnderline = true; return s },
		"strikethrough":     func(s []Span) []Span { s[1].Strikethrough = true; return s },
		"decoration color":  func(s []Span) []Span { s[0].Underline, s[0].DecorationColor = true, RGB(1, 2, 3); return s },
		"decoration width":  func(s []Span) []Span { s[0].Underline, s[0].DecorationThickness = true, 3; return s },
		"weight":            func(s []Span) []Span { s[1].Weight = 800; return s },
		"text of one span":  func(s []Span) []Span { s[1].Text = " no"; return s },
		"a span added":      func(s []Span) []Span { return append(s, Span{Text: "!"}) },
		"a span removed":    func(s []Span) []Span { return s[:1] },
		"second span color": func(s []Span) []Span { s[1].Color = RGB(9, 9, 9); return s },
	} {
		for _, kept := range []bool{false, true} {
			t.Run(fmt.Sprint(name, " kept ", kept), func(t *testing.T) {
				changed := false
				tt, _ := richTable(func(int) []Span {
					s := base()
					if changed {
						s = change(s)
					}
					return s
				}, kept)
				tt.rt.replayCheck = true
				fresh := tt.rt.freshRows
				tt.Frame()
				if tt.rt.freshRows != fresh {
					t.Fatalf("an idle frame painted %d rows anew", tt.rt.freshRows-fresh)
				}
				changed = true
				richVersion++
				tt.Frame()
				if tt.rt.freshRows == fresh {
					t.Fatalf("after the change, no row painted anew")
				}
				if tt.rt.replayMismatch != 0 {
					t.Fatalf("%d replayed rows differ from painting them", tt.rt.replayMismatch)
				}
				tt.Frame() // the rows record what they painted
				fresh = tt.rt.freshRows
				tt.Frame()
				if tt.rt.freshRows != fresh {
					t.Fatalf("after the change, an idle frame painted %d rows anew", tt.rt.freshRows-fresh)
				}
			})
		}
	}
}

// Rows of rich text that differ row by row keep their own spans: a replay of
// one row never paints another's.
func TestRowsWithRichTextKeepTheirOwnSpans(t *testing.T) {
	shade := 0
	tt, _ := richTable(func(r int) []Span {
		return []Span{{Text: fmt.Sprint("r", r), Color: RGB(uint8(30*(r%3)+shade), 0, 0)}}
	}, true)
	tt.rt.replayCheck = true
	tt.Frame()
	shade = 7
	richVersion++
	tt.Frame()
	tt.Frame()
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows differ from painting them", tt.rt.replayMismatch)
	}
}

// Rows of rich text replay what painting them does, as the spans of random rows
// change frame after frame (colors, lines, backgrounds, texts, spans added and
// removed), the pointer moves and the table scrolls: each replayed row is
// painted as well (replayCheck), and a frame's operations are compared with a
// painting that replays nothing. The rows are built anew each frame, so the
// comparison of what they paint is what keeps a stale record from replaying.
func TestRichRowsReplayMatches(t *testing.T) {
	seeds, frames := 16, 60
	if testing.Short() || raceEnabled {
		seeds = 4
	}
	palette := []Color{{}, RGB(200, 0, 0), RGB(0, 0, 200), RGB(0, 150, 0), RGB(250, 220, 120)}
	replayed := 0
	for seed := range uint64(seeds) {
		rng := rand.New(rand.NewPCG(seed, 77))
		type rowSpans struct{ spans []Span }
		rows := make([]rowSpans, 60)
		randomSpan := func(i int) Span {
			return Span{
				Text:   fmt.Sprint("s", i, "-", rng.IntN(3)),
				Color:  palette[rng.IntN(len(palette))],
				Weight: []int{0, 400, 700}[rng.IntN(3)], Underline: rng.IntN(5) == 0, Strikethrough: rng.IntN(7) == 0,
				WavyUnderline: rng.IntN(9) == 0, Background: palette[rng.IntN(len(palette))],
				Radius: []float32{0, 3, 6}[rng.IntN(3)], DecorationColor: palette[rng.IntN(len(palette))],
			}
		}
		for r := range rows {
			for i := range 1 + rng.IntN(3) {
				rows[r].spans = append(rows[r].spans, randomSpan(i))
			}
		}
		var list ListState
		tt := lcTester(func(c *context) {
			coreTable(c, &list, []TableColumn{{Title: "A"}, {Title: "B"}}, len(rows), func(r, col int) {
				if col == 0 {
					coreTextf(c, "row %d", r)
					return
				}
				coreRichText(c, slices.Clone(rows[r].spans)...)
			}).Grow(1)
		}, 400, 300, time.Unix(1e9, 0))
		tt.rt.replayCheck = true
		var log []string
		for f := range frames {
			switch k := rng.IntN(8); {
			case k < 3:
				for range 1 + rng.IntN(3) {
					r := rng.IntN(len(rows))
					s := &rows[r].spans
					switch rng.IntN(4) {
					case 0:
						*s = append(*s, randomSpan(len(*s)))
					case 1:
						if len(*s) > 1 {
							*s = (*s)[:len(*s)-1]
						}
					default:
						i := rng.IntN(len(*s))
						(*s)[i] = randomSpan(i)
					}
					log = append(log, fmt.Sprint("row ", r))
				}
			case k == 3:
				tt.Move(float32(rng.IntN(400)), float32(rng.IntN(300)))
				log = append(log, "move")
			case k == 4:
				tt.Scroll(100, 100, 0, float32(rng.IntN(7)-3))
				log = append(log, "scroll")
			default:
				log = append(log, "unchanged")
			}
			for rep := range 1 + rng.IntN(4) {
				tt.Frame()
				replayed += tt.rt.replayed
				w, h, sc := tt.h.size()
				tt.rt.paint(tt.rt.c.root, w, h, sc)
				replayed += tt.rt.replayed
				got := withoutEmptyClips(tt.rt.scene.Ops)
				tt.rt.noReplay = true
				tt.rt.paint(tt.rt.c.root, w, h, sc)
				tt.rt.noReplay = false
				if len(log) > 6 {
					log = log[len(log)-6:]
				}
				if tt.rt.replayMismatch != 0 {
					t.Fatalf("seed %d, frame %d.%d, after %s: %d replayed rows differ from painting them", seed, f, rep, strings.Join(log, ", "), tt.rt.replayMismatch)
				}
				if d := sameOps(got, withoutEmptyClips(tt.rt.scene.Ops)); d != "" {
					t.Fatalf("seed %d, frame %d.%d, after %s: %s", seed, f, rep, strings.Join(log, ", "), d)
				}
			}
		}
	}
	if replayed == 0 {
		t.Error("no row replayed")
	}
}
