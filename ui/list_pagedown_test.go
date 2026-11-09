package ui

import (
	"runtime"
	"testing"
)

// pageTable is a table of 2000 rows in a 400×300 window whose height leaves
// its last row in view cut by the bottom (and, with wide, by the track of
// the horizontal bar too).
func pageTable(wide bool) (*Tester, *ListState) {
	s := &ListState{Selection: &Selection[int]{}}
	w := float32(300)
	if wide {
		w = 900
	}
	cols := []TableColumn{{Title: "A", Width: w, Fixed: true}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, s, cols, 2000, func(row, col int) { coreTextf(c, "r%d", row) }).Grow(1)
	}, 400, 230)
	tt.Frame()
	tt.Frame()
	return tt, s
}

// fullRows returns the first and the last row that show whole: not cut by
// the top of the list, its bottom, or the track of its horizontal bar.
func fullRows(t *testing.T, tt *Tester, s *ListState) (first, last int) {
	t.Helper()
	st := listStateOf(s)
	top, bottom := st.y, st.y+st.h
	if bt, ok := hBarTop(tt, s); ok {
		bottom = bt
	}
	first, last = -1, -1
	v0, v1 := s.Visible()
	for i := v0; i <= v1; i++ {
		r, ok := rowBox(tt, s, i)
		if !ok || r.Y < top || r.Y+r.H > bottom+0.01 {
			continue
		}
		if first < 0 {
			first = i
		}
		last = i
	}
	if first < 0 {
		t.Fatalf("no row shows whole between %v and %v", top, bottom)
	}
	return first, last
}

func TestPageKeysFromTheEdgeRowsInViewGoAPageFurther(t *testing.T) {
	for _, wide := range []bool{false, true} {
		name := "no horizontal bar"
		if wide {
			name = "horizontal bar"
		}
		t.Run(name, func(t *testing.T) {
			tt, s := pageTable(wide)
			first, last := fullRows(t, tt, s)
			if v1 := 5; wide != (last == v1-1) {
				t.Fatalf("setup: the last whole row is %d: the bar's track should hide row 5 only when wide", last)
			}
			if _, v1 := s.Visible(); v1 == last {
				t.Fatalf("setup: the last row in view, %d, is not cut", v1)
			}
			s.lead = last
			if to := s.frame.page(1); to-last < last-first {
				t.Errorf("PageDown from the last whole row %d went to %d, less than the %d rows that show", last, to, last-first)
			}
			tt.Scroll(200, 100, 0, 500+7) // so that a row is cut at the top
			tt.Frame()
			tt.Frame()
			first, last = fullRows(t, tt, s)
			if v0, _ := s.Visible(); v0 == first {
				t.Fatalf("setup: the first row in view, %d, is not cut", v0)
			}
			s.lead = first
			if to := s.frame.page(-1); first-to < last-first {
				t.Errorf("PageUp from the first whole row %d went to %d, less than the %d rows that show", first, to, last-first)
			}
		})
	}
}

// On macOS lists leave PageDown and PageUp to scrolling (navigate), so the
// Tester's keys move nothing there: the test above calls page, and this one
// presses the keys where they move the choice.
func TestPageDownTwiceMovesAPageEachTime(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("lists scroll on PageDown and PageUp on macOS; TestPageKeysFromTheEdgeRowsInViewGoAPageFurther covers page")
	}
	tt, s := pageTable(true)
	tt.Click("r0")
	tt.Key(0, KeyPageDown)
	first := s.lead
	tt.Key(0, KeyPageDown)
	if s.lead-first <= 1 {
		t.Errorf("the second PageDown went from row %d to %d", first, s.lead)
	}
	tt.Key(0, KeyPageUp)
	if first-s.lead <= 1 {
		t.Errorf("PageUp went from row %d to %d", first+(s.lead-first), s.lead)
	}
}
