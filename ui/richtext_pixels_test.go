package ui

import (
	"bytes"
	"fmt"
	"testing"
	"time"
	"unsafe"
)

// pixelTwin runs view in two Testers, one of which never replays a row
// (noReplay from the start, so its records never feed it): after each step
// the frames must have the same pixels.
type pixelTwin struct {
	t    *testing.T
	a, b *Tester
}

func (tw *pixelTwin) do(fn func(tt *Tester)) { fn(tw.a); fn(tw.b) }

// frame frames both and compares pixels; it returns how many rows a replayed.
func (tw *pixelTwin) frame(what string) int {
	tw.t.Helper()
	tw.a.rt.replayCheck = true
	tw.a.Frame()
	tw.b.Frame()
	if tw.a.rt.replayMismatch != 0 {
		tw.t.Fatalf("%s: replayCheck: %d rows differ", what, tw.a.rt.replayMismatch)
	}
	ia, ib := tw.a.Image(), tw.b.Image()
	if !bytes.Equal(ia.Pix, ib.Pix) {
		n := 0
		for i := range ia.Pix {
			if ia.Pix[i] != ib.Pix[i] {
				n++
			}
		}
		tw.t.Fatalf("%s: replayed frame differs from a frame painted anew in %d bytes", what, n)
	}
	return tw.a.rt.replayed
}

func richTableOf(c *context, list *ListState, n int, cell func(c *context, r int)) {
	coreTable(c, list, []TableColumn{{Title: "A"}, {Title: "B"}}, n, func(r, col int) {
		if col == 0 {
			coreTextf(c, "row %d", r)
			return
		}
		cell(c, r)
	}).Grow(1)
}

// 1. The app keeps one []Span and changes it in place between frames.
func TestRichRowsPixelsSpansMutatedInPlace(t *testing.T) {
	for _, kept := range []bool{false, true} {
		t.Run(fmt.Sprint("kept ", kept), func(t *testing.T) {
			shared := []Span{{Text: "Done", Color: RGB(255, 0, 0), Background: RGB(0, 150, 0)}, {Text: " ok"}}
			var version uint64 = 1
			views := [2]ListState{}
			mk := func(i int) func(c *context) {
				if kept {
					views[i].RowVersion = func(int) uint64 { return version }
				}
				return func(c *context) {
					richTableOf(c, &views[i], 100, func(c *context, r int) { coreRichText(c, shared...) })
				}
			}
			now := time.Unix(1e9, 0)
			tw := &pixelTwin{t: t, a: lcTester(mk(0), 400, 300, now), b: lcTester(mk(1), 400, 300, now)}
			tw.b.rt.noReplay = true
			for range 4 {
				tw.frame("warm")
			}
			if tw.frame("idle") == 0 {
				t.Fatal("no row replayed on an idle frame")
			}
			for i, c := range []Color{RGB(0, 0, 255), RGB(0, 200, 0), RGB(9, 9, 9)} {
				shared[0].Color = c
				shared[1].Underline = i%2 == 0
				version++
				tw.frame(fmt.Sprint("mutation ", i))
				tw.frame(fmt.Sprint("after mutation ", i))
			}
		})
	}
}

// 2. Span colors from the theme, and spans without a color (the text's), as
// the window turns dark and back.
func TestRichRowsPixelsSpansThemeChange(t *testing.T) {
	for _, kept := range []bool{false, true} {
		t.Run(fmt.Sprint("kept ", kept), func(t *testing.T) {
			var la, lb ListState
			if kept {
				la.RowVersion = func(int) uint64 { return 1 }
				lb.RowVersion = la.RowVersion
			}
			view := func(l *ListState) func(c *context) {
				return func(c *context) {
					richTableOf(c, l, 100, func(c *context, r int) {
						coreRichText(c, Span{Text: "Accent", Color: c.theme.Accent, Background: c.theme.Text}, Span{Text: " plain"})
					})
				}
			}
			now := time.Unix(1e9, 0)
			tw := &pixelTwin{t: t, a: lcTester(view(&la), 400, 300, now), b: lcTester(view(&lb), 400, 300, now)}
			tw.b.rt.noReplay = true
			for range 4 {
				tw.frame("warm")
			}
			for _, dark := range []bool{true, false, true} {
				tw.do(func(tt *Tester) { tt.SetDark(dark) })
				tw.frame(fmt.Sprint("dark ", dark))
				tw.frame(fmt.Sprint("dark ", dark, " idle"))
			}
		})
	}
}

// 3. Scale change, and hover moving over rows.
func TestRichRowsPixelsSpansScaleAndHover(t *testing.T) {
	var la, lb ListState
	view := func(l *ListState) func(c *context) {
		return func(c *context) {
			richTableOf(c, l, 100, func(c *context, r int) {
				coreRichText(c, Span{Text: "Done", Color: RGB(255, 255, 255), Background: RGB(0, 150, 0), Radius: 4, Underline: true}, Span{Text: " ok", WavyUnderline: true})
			})
		}
	}
	now := time.Unix(1e9, 0)
	tw := &pixelTwin{t: t, a: lcTester(view(&la), 400, 300, now), b: lcTester(view(&lb), 400, 300, now)}
	tw.b.rt.noReplay = true
	for range 4 {
		tw.frame("warm")
	}
	for i, y := range []float32{40, 70, 100, 70, 400} {
		tw.do(func(tt *Tester) { tt.Move(200, y) })
		tw.frame(fmt.Sprint("hover ", i))
	}
	tw.do(func(tt *Tester) { tt.SetScale(2) })
	tw.frame("scale 2")
	tw.frame("scale 2 idle")
	tw.do(func(tt *Tester) { tt.SetScale(1.5) })
	tw.frame("scale 1.5")
	tw.do(func(tt *Tester) { tt.SetZoom(1.25) })
	tw.frame("zoom")
	tw.frame("zoom idle")
}

// 4. Text selected inside a Selectable region holding rich text rows, with
// plain text as the control.
func TestRichRowsPixelsSpansSelectableRegion(t *testing.T) {
	for _, rich := range []bool{false, true} {
		t.Run(fmt.Sprint("rich ", rich), func(t *testing.T) {
			var la, lb ListState
			view := func(l *ListState) func(c *context) {
				return func(c *context) {
					coreColumn(c).Grow(1).Selectable().Children(func() {
						coreList(c, l, 100, func(r int) {
							coreRow(c).Height(24).Children(func() {
								if rich {
									coreRichText(c, Span{Text: "Selectable rich ", Color: RGB(200, 0, 0)}, Span{Text: fmt.Sprint(r)})
								} else {
									coreText(c, fmt.Sprint("Selectable rich ", r))
								}
							})
						}).Grow(1)
					})
				}
			}
			now := time.Unix(1e9, 0)
			tw := &pixelTwin{t: t, a: lcTester(view(&la), 400, 300, now), b: lcTester(view(&lb), 400, 300, now)}
			tw.b.rt.noReplay = true
			for range 4 {
				tw.frame("warm")
			}
			tw.do(func(tt *Tester) { tt.Press(5, 10) })
			tw.frame("press")
			tw.do(func(tt *Tester) { tt.Move(80, 60) })
			tw.frame("drag")
			tw.do(func(tt *Tester) { tt.Release(80, 60) })
			tw.frame("release")
			tw.frame("idle with selection")
			tw.do(func(tt *Tester) { tt.Key(0, KeyRight) })
			tw.frame("collapse selection")
			tw.do(func(tt *Tester) { tt.ClickAt(300, 280) })
			tw.frame("click away")
			tw.frame("idle")
		})
	}
}

// 5. The element's own color changes under spans without a color.
func TestRichRowsPixelsSpanInheritsElementColor(t *testing.T) {
	col := RGB(0, 0, 0)
	var la, lb ListState
	view := func(l *ListState) func(c *context) {
		return func(c *context) {
			richTableOf(c, l, 100, func(c *context, r int) {
				coreRichText(c, Span{Text: "inherit "}, Span{Text: "red", Color: RGB(200, 0, 0)}).TextColor(col)
			})
		}
	}
	now := time.Unix(1e9, 0)
	tw := &pixelTwin{t: t, a: lcTester(view(&la), 400, 300, now), b: lcTester(view(&lb), 400, 300, now)}
	tw.b.rt.noReplay = true
	for range 4 {
		tw.frame("warm")
	}
	col = RGB(0, 120, 220)
	tw.frame("color")
	tw.frame("idle")
}

// 6. Spans that change only in what the layout reads, the joined text and
// styles otherwise the same: the boundary between spans moves.
func TestRichRowsPixelsSpanBoundaryMoves(t *testing.T) {
	split := 2
	var la, lb ListState
	view := func(l *ListState) func(c *context) {
		return func(c *context) {
			richTableOf(c, l, 100, func(c *context, r int) {
				s := "Boundary"
				coreRichText(c, Span{Text: s[:split], Color: RGB(200, 0, 0)}, Span{Text: s[split:], Background: RGB(0, 150, 0)})
			})
		}
	}
	now := time.Unix(1e9, 0)
	tw := &pixelTwin{t: t, a: lcTester(view(&la), 400, 300, now), b: lcTester(view(&lb), 400, 300, now)}
	tw.b.rt.noReplay = true
	for range 4 {
		tw.frame("warm")
	}
	for _, s := range []int{3, 5, 1} {
		split = s
		tw.frame(fmt.Sprint("split ", s))
	}
}

// 7. A row's state changes kind: rich text with spans, then a text, then
// rich text again (Match-like toggling at one key).
func TestRichRowsPixelsRichToPlainAtOneState(t *testing.T) {
	rich := true
	var la, lb ListState
	view := func(l *ListState) func(c *context) {
		return func(c *context) {
			richTableOf(c, l, 100, func(c *context, r int) {
				if rich {
					coreRichText(c, Span{Text: "Same", Color: RGB(200, 0, 0)})
				} else {
					coreText(c, "Same")
				}
			})
		}
	}
	now := time.Unix(1e9, 0)
	tw := &pixelTwin{t: t, a: lcTester(view(&la), 400, 300, now), b: lcTester(view(&lb), 400, 300, now)}
	tw.b.rt.noReplay = true
	for range 4 {
		tw.frame("warm")
	}
	for i := range 4 {
		rich = !rich
		tw.frame(fmt.Sprint("toggle ", i))
		tw.frame(fmt.Sprint("toggle idle ", i))
	}
}

// 8. Node size.
func TestRichRowsPixelsNodeSize(t *testing.T) {
	t.Logf("node %d bytes, paintStyle %d, layoutRecord %d", unsafe.Sizeof(node{}), unsafe.Sizeof(paintStyle{}), unsafe.Sizeof(layoutRecord{}))
}

// 9. Allocations of an idle frame of a 100k table with a rich column.
func TestRichRowsPixelsIdleAllocs(t *testing.T) {
	for _, kept := range []bool{false, true} {
		t.Run(fmt.Sprint("kept ", kept), func(t *testing.T) {
			var list ListState
			if kept {
				list.RowVersion = func(int) uint64 { return 1 }
			}
			tt := lcTester(func(c *context) {
				richTableOf(c, &list, 100000, func(c *context, r int) {
					coreRichText(c, Span{Text: "Done", Color: RGB(255, 255, 255), Background: RGB(0, 150, 0), Radius: 4}, Span{Text: " ok"})
				})
			}, 400, 800, time.Unix(1e9, 0))
			for range 6 {
				tt.Frame()
			}
			fresh := tt.rt.freshRows
			allocs := testing.AllocsPerRun(20, tt.Frame)
			t.Logf("idle frame: %.0f allocs, %d rows painted anew over 21 frames", allocs, tt.rt.freshRows-fresh)
		})
	}
}

// 10. Allocations of frames that scroll a 100k table with a rich column, and
// of frames whose spans all change.
func TestRichRowsPixelsBusyAllocs(t *testing.T) {
	var list ListState
	shade := uint8(0)
	tt := lcTester(func(c *context) {
		richTableOf(c, &list, 100000, func(c *context, r int) {
			coreRichText(c, Span{Text: "Done", Color: RGB(255, 255, 255), Background: RGB(0, shade, 0), Radius: 4}, Span{Text: " ok"})
		})
	}, 400, 800, time.Unix(1e9, 0))
	for range 6 {
		tt.Frame()
	}
	scroll := testing.AllocsPerRun(20, func() { tt.Scroll(100, 100, 0, -3); tt.Frame() })
	change := testing.AllocsPerRun(20, func() { shade++; tt.Frame() })
	t.Logf("scroll frame: %.0f allocs; spans-change frame: %.0f allocs", scroll, change)
}
