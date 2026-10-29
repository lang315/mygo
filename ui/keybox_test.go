package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
)

// A DataTable's ListState.Key is asked for every row in view in every frame,
// and an id of 256 or more converted to an any is allocated each time: the
// table keeps it boxed, so asking again allocates nothing, and the key is the
// row's.
func TestDataTableKeysAreBoxedOnce(t *testing.T) {
	rows := make([]dtRow, 600)
	for i := range rows {
		rows[i] = dtRow{ID: 1000 + i, Name: "row"}
	}
	s := dtState()
	tt := dtTester(s, &rows)
	tt.Frame()
	ask := func() {
		for row := 0; row < 12; row++ {
			s.list.Key(row)
		}
	}
	ask()
	if n := allocrun.Least(10, ask); n != 0 {
		t.Errorf("asking for the keys of 12 rows in view again allocated %v times, want 0", n)
	}
	for row := 0; row < 12; row++ {
		if got, want := s.list.Key(row), any(s.At(row).ID); got != want {
			t.Errorf("row %d: key %v, want %v", row, got, want)
		}
	}
}
