package ui

import (
	"strings"
	"testing"
)

// An outline keeping its rows (Generation) walks its items again when
// Open changes through CopyFrom, which changes a Selection in one step, and
// with RowVersion set, its kept rows still show the items opened: Open's
// generation is the one both read.
func TestOutlineGenerationSeesCopyFrom(t *testing.T) {
	tree := map[string][]string{"a": {"a/1"}, "c": {"c/1"}}
	roots := []string{"a", "c"}
	s := OutlineState[string]{Generation: 1, RowVersion: func(string) uint64 { return 1 }}
	tt := coreNewTester(func(c *context) {
		coreOutline(c, &s, roots, func(k string) []string { return tree[k] }, func(k string) {
			coreText(c, k)
		}).Grow(1)
	}, 400, 400)
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	rows := func(want string) {
		t.Helper()
		got := make([]string, s.Rows())
		for i := range got {
			got[i] = s.Item(i)
		}
		if shown := strings.Join(got, " "); shown != want {
			t.Fatalf("rows %q, want %q", shown, want)
		}
		for _, k := range got {
			if _, ok := tt.Find(k); !ok {
				t.Fatalf("row %s not shown", k)
			}
		}
	}
	tt.Frame()
	rows("a c")
	var open Selection[string]
	open.Add("c")
	s.Open.CopyFrom(&open)
	tt.Frame()
	rows("a c c/1")
	open.Clear()
	open.Add("a")
	s.Open.CopyFrom(&open)
	tt.Frame()
	rows("a a/1 c")
}
