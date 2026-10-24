package widget

import (
	"runtime"
	"slices"
	"sync"
	"testing"
	"weak"

	"github.com/egoist/mygo/ui"
)

func acctKey(a *acct) int { return a.id }

// threeTables returns a source of ann, bob and cat and three tables over it.
func threeTables(t *testing.T) (*RowSource[acct, int], [3]*DataGridViewWidget[acct, int]) {
	t.Helper()
	src := NewRowSource(acctKey)
	var tbs [3]*DataGridViewWidget[acct, int]
	for i := range tbs {
		tbs[i] = DataGridViewOf(src,
			Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true, Filterable: true},
		).CheckboxColumn()
	}
	src.SetRows([]acct{{1, "ann", 1}, {2, "bob", 2}, {3, "cat", 3}})
	return src, tbs
}

func TestRowSourceUpdateShowsInAllTables(t *testing.T) {
	src, tbs := threeTables(t)
	var tts [3]*ui.Tester
	for i, tb := range tbs {
		tts[i] = tester(t, tb)
	}
	src.UpdateRow(2, func(a *acct) { a.name = "zed" })
	for i, tt := range tts {
		tt.Frame()
		if !tt.HasText("zed") || tt.HasText("bob") {
			t.Errorf("table %d shows %q after UpdateRow", i, tt.Texts())
		}
	}
	if r, ok := tbs[1].Row(2); !ok || r.name != "zed" || src.Len() != 3 {
		t.Fatalf("Row %v %v, Len %d", r, ok, src.Len())
	}
}

func TestRowSourceUpdateReordersASortedTable(t *testing.T) {
	src := NewRowSource(acctKey)
	tb := DataGridViewOf(src,
		Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Sortable: true},
		Column[acct]{Title: "Score", Text: func(a *acct) string { return itoa(a.score) }, Sortable: true,
			Compare: func(a, b *acct) int { return a.score - b.score }})
	src.SetRows([]acct{{1, "ann", 1}, {2, "bob", 2}, {3, "cat", 3}})
	tt := tester(t, tb)
	click(t, tt, "Score")
	inOrder(t, tt, "ann", "bob", "cat")
	src.UpdateRow(1, func(a *acct) { a.score = 9 })
	tt.Frame()
	inOrder(t, tt, "bob", "cat", "ann")
}

func TestRowSourceDeleteAndReAddKeepsNoOldCheckOrSelection(t *testing.T) {
	for name, del := range map[string]func(src *RowSource[acct, int], tb *DataGridViewWidget[acct, int]){
		"source": func(src *RowSource[acct, int], _ *DataGridViewWidget[acct, int]) { src.DeleteRows(2) },
		"table":  func(_ *RowSource[acct, int], tb *DataGridViewWidget[acct, int]) { tb.DeleteRows(2) },
	} {
		t.Run(name, func(t *testing.T) {
			src, tbs := threeTables(t)
			tt := tester(t, tbs[0])
			click(t, tt, "bob")
			tbs[0].SetChecked([]int{2}, true)
			tbs[1].SetChecked([]int{2}, true) // a table never drawn
			tt.Frame()
			del(src, tbs[0])
			src.AddRows([]acct{{2, "bob", 2}})
			if tbs[0].IsChecked(2) || tbs[1].IsChecked(2) || tbs[0].CountSelected() != 0 {
				t.Fatalf("the old check or selection of a deleted row came back with its key: %v %v %d",
					tbs[0].IsChecked(2), tbs[1].IsChecked(2), tbs[0].CountSelected())
			}
		})
	}
}

func TestRowSourceInputOnAFrameWhoseRowsAnotherTableDeleted(t *testing.T) {
	_, tbs := threeTables(t)
	tt := tester(t, tbs[0])
	tbs[1].DeleteRows(2) // bob goes; tbs[0]'s frame still shows him
	if k := tbs[0].keyAt(1); k != 2 {
		t.Fatalf("row 1 of the frame shown is %v, want bob (2)", k)
	}
	tbs[0].chosen.shown.Add(2) // the user chooses bob in that frame
	tbs[0].commit(&tbs[0].chosen)
	if tbs[0].CountSelected() != 0 {
		t.Fatalf("selected %d: bob is gone", tbs[0].CountSelected())
	}
	tt.Frame()
}

func TestRowSourceShownTableNeverResetsWhileAHiddenOneLags(t *testing.T) {
	src := NewRowSource(acctKey)
	src.SetRows(manyAccts(100))
	cols := Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }}
	shown, hidden := DataGridViewOf(src, cols), DataGridViewOf(src, cols)
	_ = hidden
	for i := range 1000 {
		src.UpdateRow(i%100, func(a *acct) { a.score++ })
		mu.Lock()
		shown.catchUp()
		mu.Unlock()
	}
	if shown.resets != 0 {
		t.Fatalf("the shown table started over %d times because a hidden one lags", shown.resets)
	}
	runtime.KeepAlive(hidden)
}

func TestRowSourceTablesKeepTheirOwnOrder(t *testing.T) {
	_, tbs := threeTables(t)
	tts := [3]*ui.Tester{tester(t, tbs[0]), tester(t, tbs[1]), tester(t, tbs[2])}
	click(t, tts[1], "Name")
	click(t, tts[1], "Name") // descending
	inOrder(t, tts[0], "ann", "bob", "cat")
	inOrder(t, tts[1], "cat", "bob", "ann")
	inOrder(t, tts[2], "ann", "bob", "cat")
	tbs[0].AddRow(acct{4, "dan", 4}) // forwards to the source
	for _, tt := range tts {
		tt.Frame()
	}
	inOrder(t, tts[1], "dan", "cat", "bob", "ann")
	inOrder(t, tts[2], "ann", "bob", "cat", "dan")
}

func TestRowSourceDeletePrunesEveryTable(t *testing.T) {
	src, tbs := threeTables(t)
	for i, tb := range tbs {
		tt := tester(t, tb)
		click(t, tt, "bob")
		tb.SetChecked([]int{2, 3}, true)
		tt.Frame()
		if tb.CountSelected() != 1 || !tb.IsChecked(2) {
			t.Fatalf("table %d: setup", i)
		}
	}
	src.DeleteRows(2)
	for i, tb := range tbs {
		if tb.CountSelected() != 0 || tb.IsChecked(2) || !slices.Equal(tb.Checked(), []int{3}) {
			t.Errorf("table %d: selected %d, checked %v", i, tb.CountSelected(), tb.Checked())
		}
	}
}

func TestRowSourceTableBuiltAfterManyChangesShowsTheCurrentRows(t *testing.T) {
	src := NewRowSource(acctKey)
	src.SetRows(manyAccts(20))
	for i := range 10_000 {
		src.UpdateRow(i%20, func(a *acct) { a.score = i })
	}
	src.DeleteRows(5)
	tb := DataGridViewOf(src, Column[acct]{Title: "Score", Text: func(a *acct) string { return itoa(a.score) }})
	tt := tester(t, tb)
	if !tt.HasText("9999") || tb.Len() != 19 {
		t.Fatalf("shows %q, Len %d", tt.Texts(), tb.Len())
	}
}

func itoa(n int) string {
	return string(rune('0'+n/1000%10)) + string(rune('0'+n/100%10)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
}

func TestRowSourceDroppedTableIsCollectedAndTheLogTrims(t *testing.T) {
	src := NewRowSource(acctKey)
	src.SetRows(manyAccts(1000))
	keep := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }})
	var gone weak.Pointer[DataGridViewWidget[acct, int]]
	func() {
		lag := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }})
		gone = weak.Make(lag)
	}()
	for i := range 50 {
		src.UpdateRow(i, func(a *acct) { a.score++ })
	}
	keep.sync() // under mu in real use; the test is alone
	if n := src.logLen(); n == 0 {
		t.Fatal("log empty while the dropped table may still lag")
	}
	for range 5 {
		runtime.GC()
	}
	if gone.Value() != nil {
		t.Fatal("the dropped table was not collected: the source holds it")
	}
	mu.Lock()
	keep.catchUp()
	mu.Unlock()
	src.UpdateRow(60, func(a *acct) { a.score++ })
	if n, live := src.logLen(), src.attached(); n != 1 || live != 1 {
		t.Fatalf("log holds %d entries, %d tables attached: want 1 and 1", n, live)
	}
}

func TestRowSourceLogIsBounded(t *testing.T) {
	src := NewRowSource(acctKey)
	src.SetRows(manyAccts(100))
	lag := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }})
	for i := range 10_000 {
		src.UpdateRow(i%100, func(a *acct) { a.score++ })
	}
	if n := src.logLen(); n > 64 {
		t.Fatalf("log holds %d entries for a lagging table", n)
	}
	tt := tester(t, lag) // catches up through a Reset
	if !tt.HasText("acct000000") || lag.Len() != 100 {
		t.Fatalf("shows %q", tt.Texts())
	}
}

func TestRowSourceRace(t *testing.T) {
	src, tbs := threeTables(t)
	var tts [3]*ui.Tester
	for i, tb := range tbs {
		tts[i] = tester(t, tb)
	}
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				src.UpdateRow(1+i%3, func(a *acct) { a.score = i })
				if i%50 == g {
					src.AddRows([]acct{{10 + g*1000 + i, "n", i}})
				}
				if i%70 == 0 {
					src.DeleteRows(10 + g*1000 + i - 20)
				}
			}
		}()
	}
	for range 30 {
		for _, tt := range tts {
			tt.Frame()
		}
	}
	wg.Wait()
}

func TestRowSourceChangeRedrawsTheWindowsOfItsTables(t *testing.T) {
	src, tbs := threeTables(t)
	_, w0 := showFrame(t, tbs[0])
	_, w1 := showFrame(t, tbs[1])
	_, wOther := showFrame(t, Label("other"))
	_ = wOther
	redrawn := recordInvalidations(t)
	src.UpdateRow(1, func(a *acct) { a.score++ })
	wantRedrawn(t, redrawn(), w0, w1)
	tbs[2].AddRow(acct{9, "z", 9}) // a table's setter reaches the others
	wantRedrawn(t, redrawn(), w0, w1)
}

// BenchmarkRowSourceUpdate updates one of a million rows and draws the
// frame of the table shown, with one table over the source and with three.
func BenchmarkRowSourceUpdate(b *testing.B) {
	const n = 1_000_000
	rows := manyAccts(n)
	for _, tables := range []int{1, 3} {
		b.Run(itoa(tables)[3:]+"tables", func(b *testing.B) {
			src := NewRowSource(acctKey)
			src.SetRows(rows)
			var shown *DataGridViewWidget[acct, int]
			var all []*DataGridViewWidget[acct, int]
			for range tables {
				tb := DataGridViewOf(src, Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200, Sortable: true})
				all = append(all, tb)
				if shown == nil {
					shown = tb
				}
			}
			tt := ui.NewTester(content(shown), 1280, 800)
			tt.Frame()
			if err := tt.Click("Name"); err != nil { // sorted: updates place the row again
				b.Fatal(err)
			}
			tt.Frame()
			defer runtime.KeepAlive(all)
			b.ResetTimer()
			for i := range b.N {
				src.UpdateRow(i%50, func(a *acct) { a.score++ })
				tt.Frame()
			}
		})
	}
}
