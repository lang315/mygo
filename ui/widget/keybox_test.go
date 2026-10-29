package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
)

type kbRow struct {
	id   int
	sid  string
	name string
}

func kbRows(n int) []kbRow {
	rs := make([]kbRow, n)
	for i := range rs {
		rs[i] = kbRow{1000 + i, fmt.Sprintf("id%06d", i), fmt.Sprintf("row %d", i)}
	}
	return rs
}

// ui asks a list for the key of every row in view in every frame, and a key of
// 256 or more or a string boxed as an any is allocated each time: the widgets
// keep it boxed, so asking again allocates nothing, and the key is the same.
func TestListBoxAndGridKeysAreBoxedOnce(t *testing.T) {
	rs := kbRows(2000)
	type asker struct {
		name string
		ask  func(row int) any
		want func(row int) any
	}
	lb := ListBox(func(r *kbRow) int { return r.id }, func(r *kbRow) string { return r.name }).SetItems(rs)
	ls := ListBox(func(r *kbRow) string { return r.sid }, func(r *kbRow) string { return r.name }).SetItems(rs)
	src := NewRowSource(func(r *kbRow) int { return r.id }).SetRows(rs)
	dg := DataGridViewOf(src, Column[kbRow]{Title: "Name", Text: func(r *kbRow) string { return r.name }})
	for _, w := range []Control{lb, ls, dg} {
		tester(t, w)
	}
	for _, a := range []asker{
		{"ListBox with int keys", lb.keyAt, func(row int) any { return rs[row].id }},
		{"ListBox with string keys", ls.keyAt, func(row int) any { return rs[row].sid }},
		{"DataGridView with int keys", dg.keyAt, func(row int) any { return rs[row].id }},
	} {
		t.Run(a.name, func(t *testing.T) {
			rows := func(f func(row int)) {
				for row := 5; row < 25; row++ {
					f(row)
				}
			}
			rows(func(row int) { a.ask(row) }) // boxed once
			if n := allocrun.Least(10, func() { rows(func(row int) { a.ask(row) }) }); n != 0 {
				t.Errorf("asking for the keys of 20 rows in view again allocated %v times, want 0", n)
			}
			rows(func(row int) {
				if got, want := a.ask(row), a.want(row); got != want {
					t.Errorf("row %d: key %v, want %v", row, got, want)
				}
			})
		})
	}
}

// A keyBox answers for the value of the key, whatever the slot asked for: a row
// that moved takes the box of its key, and a slot whose key changed is boxed
// anew.
func TestKeyBoxFollowsTheValue(t *testing.T) {
	var b keyBox[int]
	one := b.of(3, 1000)
	if got := b.of(3, 1000); got != one || got != any(1000) {
		t.Errorf("same slot and key: %v", got)
	}
	if got := b.of(3, 2000); got != any(2000) {
		t.Errorf("same slot, another key: %v, want 2000", got)
	}
	if got := b.of(3+keyBoxSize, 2000); got != any(2000) {
		t.Errorf("a slot that maps to the same entry: %v, want 2000", got)
	}
	if got := b.of(7, 1000); got != any(1000) {
		t.Errorf("another slot: %v, want 1000", got)
	}
	var s keyBox[string]
	if got := s.of(0, "a"); got != any("a") {
		t.Errorf("string key: %v", got)
	}
}
