package ui

import "testing"

// replayTable is a table of n rows in a 300×400 tester, warmed by two frames, for the replay tests.
func replayTable(t *testing.T, n int, cell func(c *Context, r int)) (*Tester, *ListState) {
	t.Helper()
	var list ListState
	tt := NewTester(func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}}, n, func(r, col int) { cell(c, r) }).Grow(1)
	}, 300, 400)
	for range 4 {
		tt.Frame()
	}
	return tt, &list
}

func rowText(c *Context, r int) { Textf(c, "row %d", r) }

// Each row in view records what it painted.
func TestRowsRecordWhatTheyPainted(t *testing.T) {
	tt, list := replayTable(t, 100, rowText)
	first, last := list.Visible()
	if got := tt.rt.recorded + tt.rt.replayed; got != last-first+1 {
		t.Fatalf("%d rows recorded or replayed, %d rows in view", got, last-first+1)
	}
}

// What a row paints that is not in the scene's operations and glyphs alone
// keeps it from recording.
func TestRowsThatCannotBeReplayedDoNotRecord(t *testing.T) {
	plain, _ := replayTable(t, 100, rowText)
	base := plain.rt.recorded + plain.rt.replayed
	for name, cell := range map[string]func(c *Context, r int){
		"a Draw callback": func(c *Context, r int) {
			if r == 0 {
				Box(c).Size(10, 10).Draw(func(p *Painter, rc Rect) { p.Fill(rc, RGB(1, 2, 3), 0) })
			} else {
				rowText(c, r)
			}
		},
		"a gradient": func(c *Context, r int) {
			if r == 0 {
				Box(c).Size(10, 10).Gradient(RGB(1, 2, 3), RGB(4, 5, 6), 45)
			} else {
				rowText(c, r)
			}
		},
		"a wide color": func(c *Context, r int) {
			if r == 0 {
				Text(c, "wide").TextColor(Oklch(0.8, 0.4, 145))
			} else {
				rowText(c, r)
			}
		},
	} {
		tt, _ := replayTable(t, 100, cell)
		if got := tt.rt.recorded + tt.rt.replayed; got >= base {
			t.Errorf("%s: %d rows recorded, %d in a plain table", name, got, base)
		}
	}
}

// In an idle frame, the rows recorded in the frame before replay, and what
// the frame paints is what it paints without replaying.
func TestIdleRowsReplay(t *testing.T) {
	tt, list := replayTable(t, 100, rowText)
	tt.rt.replayCheck = true
	tt.Frame()
	_, last := list.Visible()
	if tt.rt.replayed != last+1 || tt.rt.recorded != 0 {
		t.Fatalf("%d rows replayed and %d recorded, %d rows in view", tt.rt.replayed, tt.rt.recorded, last+1)
	}
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows differ from painting them", tt.rt.replayMismatch)
	}
}

// What makes a row paint again, though it lays out and is built as before.
func TestRowsPaintAgainWhenWhatTheyReadChanges(t *testing.T) {
	for name, change := range map[string]func(tt *Tester){
		"hover":        func(tt *Tester) { tt.Move(100, 60) },
		"focus":        func(tt *Tester) { tt.Key(0, KeyTab) },
		"window focus": func(tt *Tester) { tt.SetFocused(false) },
		"scale":        func(tt *Tester) { tt.SetScale(2) },
		"scroll":       func(tt *Tester) { tt.Scroll(100, 200, 0, 1) },
		"theme":        func(tt *Tester) { tt.rt.themeGen++ },
	} {
		t.Run(name, func(t *testing.T) {
			tt, _ := replayTable(t, 100, rowText)
			tt.rt.replayCheck = true
			tt.Frame()
			if tt.rt.replayed == 0 || tt.rt.freshRows == 0 {
				t.Fatal("no row replays in an idle frame")
			}
			fresh := tt.rt.freshRows
			tt.Frame()
			if tt.rt.freshRows != fresh {
				t.Fatalf("an idle frame painted %d rows anew", tt.rt.freshRows-fresh)
			}
			change(tt)
			tt.Frame()
			if tt.rt.freshRows == fresh {
				t.Fatalf("after %s, no row painted anew", name)
			}
			if tt.rt.replayMismatch != 0 {
				t.Fatalf("after %s, %d replayed rows differ from painting them", name, tt.rt.replayMismatch)
			}
		})
	}
}

// The nodes' flags after a frame that replays are those after one that
// paints: Hovered and Pressed set flags as the row's styles run.
func TestReplayKeepsTheFlagsPaintSets(t *testing.T) {
	flags := func(noReplay bool) map[uint64]uint32 {
		tt, _ := replayTable(t, 100, rowText)
		tt.rt.noReplay = noReplay
		tt.Frame()
		got := map[uint64]uint32{}
		walkRow(tt.rt.c.root, func(e *node) { got[e.id] |= e.flags })
		return got
	}
	painted, replayed := flags(true), flags(false)
	for id, f := range painted {
		if replayed[id] != f {
			t.Errorf("element %x: flags %b replaying, %b painting", id, replayed[id], f)
		}
	}
}

// A check box reads the hover of the row around it (checkboxStyle): the row
// of a hovered box's parent paints again.
func TestHoverOfTheParentOfABoxRepaintsTheRow(t *testing.T) {
	var list ListState
	var checked Selection[int]
	list.Checked = &checked
	tt := NewTester(func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}}, 100, func(r, col int) { rowText(c, r) }).Grow(1)
	}, 300, 400)
	tt.Frame()
	tt.Frame()
	tt.rt.replayCheck = true
	tt.Frame()
	fresh := tt.rt.freshRows
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame")
	}
	tt.Move(100, 60)
	tt.Frame()
	if tt.rt.freshRows == fresh || tt.rt.replayMismatch != 0 {
		t.Fatalf("hover: %d rows painted anew, %d mismatches", tt.rt.freshRows-fresh, tt.rt.replayMismatch)
	}
}

// A disabled ancestor fades the rows (opacity in the key): they paint again.
func TestDisabledAncestorRepaintsRows(t *testing.T) {
	dis := false
	var list ListState
	tt := NewTester(func(c *Context) {
		Box(c).Grow(1).Disabled(dis).Children(func() {
			Table(c, &list, []TableColumn{{Title: "A"}}, 100, func(r, col int) { rowText(c, r) }).Grow(1)
		})
	}, 300, 400)
	tt.Frame()
	tt.Frame()
	tt.rt.replayCheck = true
	tt.Frame()
	fresh := tt.rt.freshRows
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame")
	}
	dis = true
	tt.Frame()
	if tt.rt.freshRows == fresh || tt.rt.replayMismatch != 0 {
		t.Fatalf("disabled: %d rows painted anew, %d mismatches", tt.rt.freshRows-fresh, tt.rt.replayMismatch)
	}
}

// A replayed row marks the atlas entries it draws from used, as painting it
// would: MakeRoom keeps what the frame being painted used.
func TestReplayedRowsKeepTheirGlyphsUsed(t *testing.T) {
	tt, _ := replayTable(t, 100, rowText)
	rows := 0
	walkRow(tt.rt.c.root, func(e *node) {
		if r := e.st.replay; e.listRow && r != nil && r.ok {
			rows++
			if !tt.rt.text.UsedAgo(&r.uses, 1) {
				t.Errorf("row %d: its glyphs are not marked used in the last frame", e.rowIndex)
			}
		}
	})
	if rows == 0 {
		t.Fatal("no row replayed")
	}
}

// What the painter reads from the elements around the rows: an opaque
// background behind them (glyphs land on it), and the stack of such boxes.
func TestBackgroundBehindRowsIsInTheKey(t *testing.T) {
	opaque := true
	var list ListState
	tt := NewTester(func(c *Context) {
		bg := Transparent
		if opaque {
			bg = RGB(240, 240, 240)
		}
		Box(c).Grow(1).Background(bg).Children(func() {
			Table(c, &list, []TableColumn{{Title: "A"}}, 100, func(r, col int) { rowText(c, r) }).Grow(1)
		})
	}, 300, 400)
	for range 3 {
		tt.Frame()
	}
	tt.rt.replayCheck = true
	tt.Frame()
	fresh := tt.rt.freshRows
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame")
	}
	opaque = false
	tt.Frame()
	if tt.rt.freshRows == fresh || tt.rt.replayMismatch != 0 {
		t.Fatalf("background: %d rows painted anew, %d mismatches", tt.rt.freshRows-fresh, tt.rt.replayMismatch)
	}
}

// A moved atlas sends the rows fresh by its generation (the glyphs they
// recorded point into the old one). The shared atlas may be at its largest
// after other tests and unable to move, so the recordings are made to say
// they were made at another generation.
func TestRowsPaintAgainWhenTheAtlasMoves(t *testing.T) {
	tt, _ := replayTable(t, 100, rowText)
	tt.rt.replayCheck = true
	tt.Frame()
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays in an idle frame")
	}
	fresh := tt.rt.freshRows
	walkRow(tt.rt.c.root, func(e *node) {
		if r := e.st.replay; e.listRow && r != nil {
			r.key.atlases[0]++
		}
	})
	tt.Frame()
	if tt.rt.replayed != 0 || tt.rt.freshRows == fresh || tt.rt.replayMismatch != 0 {
		t.Fatalf("with another atlas generation, %d rows replayed, %d painted anew, %d mismatches",
			tt.rt.replayed, tt.rt.freshRows-fresh, tt.rt.replayMismatch)
	}
}

// A row that painted anew without recording must not give way to an older
// recording once it stands still again: frame A records a red background,
// B makes it blue (not paint-same, so it paints without recording), C changes
// nothing (paint-same with B, and its key equals A's).
func TestRowDoesNotReplayARecordingOlderThanItsLastPainting(t *testing.T) {
	col := RGB(255, 0, 0)
	tt, _ := replayTable(t, 20, func(c *Context, r int) { Box(c).Size(50, 10).Background(col) })
	tt.rt.replayCheck = true
	tt.Frame()
	if tt.rt.replayed == 0 {
		t.Fatal("no row replays while the color stands")
	}
	col = RGB(0, 0, 255)
	tt.Frame()
	for range 3 {
		tt.Frame()
	}
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows show the color before the change", tt.rt.replayMismatch)
	}
}

// The same with an ancestor that is Invisible while the row changes: its
// painting never runs then.
func TestRowDoesNotReplayARecordingOlderThanAChangeWhileInvisible(t *testing.T) {
	var list ListState
	col, invisible := RGB(255, 0, 0), false
	tt := NewTester(func(c *Context) {
		b := Box(c).Grow(1)
		if invisible {
			b.Invisible()
		}
		b.Children(func() {
			Table(c, &list, []TableColumn{{Title: "A"}}, 30, func(r, k int) { Box(c).Size(50, 10).Background(col) }).Grow(1)
		})
	}, 300, 400)
	for range 5 {
		tt.Frame()
	}
	tt.rt.replayCheck = true
	invisible = true
	tt.Frame()
	col = RGB(0, 0, 255)
	tt.Frame()
	tt.Frame()
	invisible = false
	for range 3 {
		tt.Frame()
	}
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows show the color before the change", tt.rt.replayMismatch)
	}
}

// Moving the chosen row with the keys: the row chosen and the row left paint
// as they are now, not as a recording of before.
func TestKeyboardSelectionRepaintsTheRows(t *testing.T) {
	var list ListState
	list.Selection = &Selection[int]{}
	tt := NewTester(func(c *Context) {
		Table(c, &list, []TableColumn{{Title: "A"}}, 30, func(r, col int) { rowText(c, r) }).Grow(1)
	}, 300, 400)
	for range 5 {
		tt.Frame()
	}
	tt.rt.replayCheck = true
	tt.Key(0, KeyTab)
	for range 3 {
		tt.Frame()
	}
	for range 4 {
		tt.Key(0, KeyDown)
		for range 3 {
			tt.Frame()
		}
	}
	if tt.rt.replayMismatch != 0 {
		t.Fatalf("%d replayed rows differ from painting them", tt.rt.replayMismatch)
	}
}

// An icon is drawn from a mask made for the frame alone the first time: its
// row does not record.
func TestRowWithAnIconDoesNotRecord(t *testing.T) {
	svg, err := ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8"><path d="M1 1h6v6H1z"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := replayTable(t, 100, rowText)
	base := plain.rt.recorded + plain.rt.replayed
	tt, _ := replayTable(t, 100, func(c *Context, r int) {
		if r == 0 {
			Icon(c, svg)
		} else {
			rowText(c, r)
		}
	})
	if got := tt.rt.recorded + tt.rt.replayed; got >= base {
		t.Errorf("%d rows recorded or replayed with an icon in one, %d in a plain table", got, base)
	}
}

// The same with an atlas that really grows (skipped when the shared atlas is
// at its largest after other tests).
func TestRowsPaintAgainWhenTheAtlasGrows(t *testing.T) {
	tt, _ := replayTable(t, 100, rowText)
	tt.rt.replayCheck = true
	tt.Frame()
	fresh := tt.rt.freshRows
	if !tt.rt.text.MaskAtlas.Grow() {
		t.Skip("the shared atlas is at its largest")
	}
	tt.Frame()
	if tt.rt.freshRows == fresh || tt.rt.replayMismatch != 0 {
		t.Fatalf("after the atlas grew, %d rows painted anew, %d mismatches", tt.rt.freshRows-fresh, tt.rt.replayMismatch)
	}
}
