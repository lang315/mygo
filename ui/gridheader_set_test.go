package ui

import (
	"fmt"
	"slices"
	"testing"
)

// An app that knows where the headers are says so with SetHeaders, and the
// grid does not ask Header for every item to find them.
func TestGridViewSetHeadersSparesTheScan(t *testing.T) {
	const n = 20000
	heads := []int{0, 5000, 10000, 15000}
	calls := 0
	sel := -1
	s := &GridState{Selected: &sel, Selection: &Selection[int]{}, Key: func(i int) any { return i }}
	s.Header = func(i int) bool { calls++; return slices.Contains(heads, i) }
	s.Label = func(i int) string {
		if slices.Contains(heads, i) {
			return fmt.Sprintf("H%d", i)
		}
		return fmt.Sprintf("Item %d", i)
	}
	s.SetHeaders(heads, n)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, n, 100, 40, func(i int) { coreText(c, s.Label(i)) }).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	tt.Frame()
	if calls > n/20 {
		t.Fatalf("Header was asked %d times for %d items, headers given", calls, n)
	}
	if !slices.Equal(s.layout.heads, heads) {
		t.Fatalf("the grid knows the headers %v, want %v", s.layout.heads, heads)
	}
	// The layout is the one the scan makes: the item after the first header
	// starts a row of its own, under it.
	ref, _, _ := headerGrid(n, heads)
	for _, text := range []string{"Item 1", "Item 5"} {
		if got, want := rowY(t, tt, text), rowY(t, ref, text); got != want {
			t.Errorf("%s is at y=%v, with the scan at y=%v", text, got, want)
		}
	}
}

// Headers given are trusted: a frame at rest does not ask Header for each of
// them again to see that it is one still, as the grid does for those it found.
func TestGridViewSetHeadersAreNotCheckedEveryFrame(t *testing.T) {
	const n = 20000
	var heads []int
	for h := 0; h < n; h += 20 {
		heads = append(heads, h) // a thousand headers
	}
	isHead := make(map[int]bool, len(heads))
	for _, h := range heads {
		isHead[h] = true
	}
	calls := 0
	sel := -1
	s := &GridState{Selected: &sel, Selection: &Selection[int]{}, Key: func(i int) any { return i }}
	s.Header = func(i int) bool { calls++; return isHead[i] }
	s.SetHeaders(heads, n)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, n, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	tt.Frame()
	calls = 0
	for range 10 {
		tt.Frame()
	}
	if calls > 10*200 { // what a frame builds: about forty items, not a thousand headers
		t.Fatalf("ten frames at rest asked Header %d times, with %d headers given", calls, len(heads))
	}
}

// The headers given are checked and replaced when the number of items changes:
// the grid asks Header for every item again, as without them.
func TestGridViewSetHeadersIsForgottenWhenTheItemsChange(t *testing.T) {
	heads := []int{0, 3}
	sel := -1
	n := 10
	s := &GridState{Selected: &sel, Key: func(i int) any { return i }}
	s.Header = func(i int) bool { return slices.Contains(heads, i) }
	s.SetHeaders(heads, n)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreGridView(c, s, n, 100, 40, func(i int) { coreTextf(c, "Item %d", i) }).Grow(1)
		})
	}, 448, 400)
	tt.Frame()
	heads, n = []int{0, 3, 6}, 12 // the app did not call SetHeaders
	tt.Frame()
	if !slices.Equal(s.layout.heads, heads) {
		t.Fatalf("the grid knows the headers %v after the items changed, want %v", s.layout.heads, heads)
	}
}
