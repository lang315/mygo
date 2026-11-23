package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A wide row with an empty label in SettingsRows (emptywide): the row is as
// high as its control asks, at least a normal row, and the control sits in
// the middle of its band, as a normal row's does.

// ewLong wraps in five lines or so, more than a Calm row's least height holds.
const ewLong = "Danh sách các khóa được phép, mỗi dòng một khóa, dài hơn mọi nhãn khác của biểu mẫu này, và còn dài hơn thế nữa để nó xuống dòng. " +
	"Danh sách các khóa được phép, mỗi dòng một khóa, dài hơn mọi nhãn khác của biểu mẫu này, và còn dài hơn thế nữa để nó xuống dòng. " +
	"Danh sách các khóa được phép, mỗi dòng một khóa, dài hơn mọi nhãn khác của biểu mẫu này, và còn dài hơn thế nữa để nó xuống dòng."

// ewForm is a card of the rows Alpha, n (wide, with no label), Gamma and
// Delta, whose labels are one line, the rows' gap 10. Alpha, Gamma and Delta
// have bands of one height, their labels in the middle of them.
func ewForm(n Control) *FieldsWidget {
	return Fields().SettingsRows().Gap(10, 16).
		Row("Alpha", Label("p")).Row("", n).Wide().Row("Gamma", Label("q")).Row("Delta", Label("r"))
}

// ewMid returns the middle of the text s.
func ewMid(t *testing.T, tt *Tester, s string) float32 {
	t.Helper()
	r := find(t, tt.Tester, s)
	return r.Y + r.H/2
}

func ewForSkins(t *testing.T, test func(t *testing.T)) {
	t.Run("calm", func(t *testing.T) { useCalm(t, true); t.Cleanup(calmOff); test(t) })
	t.Run("noskin", func(t *testing.T) { useCalm(t, false); test(t) })
}

// ewTester shows f in the card, settled.
func ewTester(t *testing.T, f *FieldsWidget) *Tester {
	tt := NewTester(t, f, 400, 500)
	tt.Settle()
	return tt
}

// Case 1 and 6: a one-line note is as high as a normal row, its text in the
// middle of its band: the rows are as far from each other as normal ones, and
// the note is midway between Alpha and Gamma.
func TestWideEmptyLabelOneLineBand(t *testing.T) {
	ewForSkins(t, func(t *testing.T) {
		tt := ewTester(t, ewForm(Label("Note one").Align(Start)))
		a, n, g, d := ewMid(t, tt, "Alpha"), ewMid(t, tt, "Note one"), ewMid(t, tt, "Gamma"), ewMid(t, tt, "Delta")
		if !approx(n-a, g-n) || !approx(g-n, d-g) {
			t.Errorf("Alpha, the note, Gamma and Delta are at %v, %v, %v and %v: the note is not as high as a row, in its middle", a, n, g, d)
		}
	})
}

// Case 2: first in its card, the control is as far below the card's top as a
// normal first row's, and the next row as far as after a normal one.
func TestWideEmptyLabelFirstInCard(t *testing.T) {
	ewForSkins(t, func(t *testing.T) {
		build := func(label string, wide bool) (*Tester, cardBox) {
			f := Fields().SettingsRows().Gap(10, 16).Row(label, wbar("w", 60, 24))
			if wide {
				f.Wide()
			}
			tt := ewTester(t, f.Row("Below", Label("b")))
			return tt, cardsOf(t, tt.Tester, cardX, ui.LightTheme().Border)[0]
		}
		note, kn := build("", true)
		norm, kr := build("Switch", false)
		if a, b := find(t, note.Tester, "w").Y-float32(kn.top), find(t, norm.Tester, "w").Y-float32(kr.top); !approx(a, b) {
			t.Errorf("the control is %v below the card's top, a normal row's %v", a, b)
		}
		if a, b := find(t, note.Tester, "Below").Y-float32(kn.top), find(t, norm.Tester, "Below").Y-float32(kr.top); !approx(a, b) {
			t.Errorf("the next row is %v below the card's top, after a normal row %v", a, b)
		}
	})
}

// Case 1 as the pilot app measured it: the label tops of Note, Mid and Last.
func TestWideEmptyLabelNoteToMid(t *testing.T) {
	ewForSkins(t, func(t *testing.T) {
		f := Fields().SettingsRows().Gap(10, 16).Section("Link").
			Row("", Label("Note one").Align(Start)).Wide().Row("Mid", Label("y")).Row("Last", Label("x"))
		tt := NewTester(t, f, 560, 500)
		tt.Settle()
		n, m, l := find(t, tt.Tester, "Note one"), find(t, tt.Tester, "Mid"), find(t, tt.Tester, "Last")
		if a, b := m.Y-n.Y, l.Y-m.Y; !approx(a, b) {
			t.Errorf("Note to Mid %v, Mid to Last %v", a, b)
		}
	})
}

// Case 3: a note of two lines or more makes its band grow, with the same
// room above and below its text (it is midway between Alpha and Gamma, whose
// bands are alike), and not less than a normal row's.
func TestWideEmptyLabelWrapped(t *testing.T) {
	ewForSkins(t, func(t *testing.T) {
		tt := ewTester(t, ewForm(Label(ewLong)))
		note := find(t, tt.Tester, ewLong)
		if one := find(t, tt.Tester, "Alpha").H; note.H < 1.5*one {
			t.Fatalf("the note is %v high, one line %v: it did not wrap", note.H, one)
		}
		a, n, g, d := ewMid(t, tt, "Alpha"), ewMid(t, tt, ewLong), ewMid(t, tt, "Gamma"), ewMid(t, tt, "Delta")
		if !approx(n-a, g-n) {
			t.Errorf("the note's middle is %v below Alpha's and %v above Gamma's", n-a, g-n)
		}
		if g-a <= 2*(d-g) {
			t.Errorf("Alpha to Gamma is %v, a normal row is %v: the note's band did not grow", g-a, d-g)
		}
	})
}

// Case 4: a text box of 140 with no label has the band of its height, and no
// empty label line above it: Alpha to Gamma is two normal rows plus the box
// and a gap, a divider and a gap on each side of it (one more of them than
// two normal rows have).
func TestWideEmptyLabelTextBox(t *testing.T) {
	ewForSkins(t, func(t *testing.T) {
		tt := ewTester(t, ewForm(MultilineTextBox().SetSize(0, 140)))
		if m := fieldBox(tt, 0); !approx(m.H, 140) {
			t.Fatalf("the text box is %v high", m.H)
		}
		a, n, g, d := ewMid(t, tt, "Alpha"), fieldBox(tt, 0), ewMid(t, tt, "Gamma"), ewMid(t, tt, "Delta")
		pitch := d - g // a normal row's band, its gaps and divider
		if want := pitch + 140 + 21; !approx(g-a, want) {
			t.Errorf("Alpha to Gamma is %v, want %v: a band of the box", g-a, want)
		}
		if !approx(n.Y+n.H/2-a, g-(n.Y+n.H/2)) {
			t.Errorf("the box is not midway between Alpha and Gamma")
		}
	})
}
