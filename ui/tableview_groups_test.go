package ui

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// tvTitle is the group of a row in the tests: b, which the columns do not
// sort by in these tests' groups, so a row changes group without changing its
// place among the others of its group.
func tvTitle(x *tvRow) string { return "g" + strconv.Itoa(x.b) }

// tvWantGrouped returns the rows a grouped view shows and its groups, by
// brute force: by the rank of the group (those named in order first, the
// others by title), then as tvWantBy sorts.
func tvWantGrouped(rows []tvRow, by []SortOrder, keep func(*tvRow) bool, key func(*tvRow) string, order []string) ([]int, []TableGroup) {
	idx := tvWantBy(rows, by, keep)
	if key == nil {
		return idx, nil
	}
	rank := map[string]int{}
	for _, title := range order {
		if _, ok := rank[title]; !ok {
			rank[title] = len(rank)
		}
	}
	var others []string
	for _, i := range idx {
		title := key(&rows[i])
		if _, ok := rank[title]; !ok && !slices.Contains(others, title) {
			others = append(others, title)
		}
	}
	slices.SortFunc(others, strings.Compare)
	for _, title := range others {
		rank[title] = len(rank)
	}
	slices.SortStableFunc(idx, func(i, j int) int { return rank[key(&rows[i])] - rank[key(&rows[j])] })
	var groups []TableGroup
	for p, i := range idx {
		title := key(&rows[i])
		if n := len(groups); n > 0 && groups[n-1].Title == title {
			groups[n-1].Len++
		} else {
			groups = append(groups, TableGroup{Title: title, Start: p, Len: 1})
		}
	}
	return idx, groups
}

// checkPlaces checks Places, PlaceOf and PosOf against the groups: a header
// before the rows of each group.
func checkPlaces[T any](t *testing.T, v *TableView[T], groups []TableGroup, where string) {
	t.Helper()
	if got, want := v.Places(), v.Len()+len(groups); got != want {
		t.Fatalf("%s: Places() = %d, want %d", where, got, want)
	}
	place := 0
	for g, gr := range groups {
		if pos, grp := v.PosOf(place); pos != -1 || grp != g {
			t.Fatalf("%s: PosOf(%d) = %d, %d, want the header of group %d", where, place, pos, grp, g)
		}
		place++
		for p := gr.Start; p < gr.Start+gr.Len; p++ {
			if got := v.PlaceOf(p); got != place {
				t.Fatalf("%s: PlaceOf(%d) = %d, want %d", where, p, got, place)
			}
			if pos, grp := v.PosOf(place); pos != p || grp != g {
				t.Fatalf("%s: PosOf(%d) = %d, %d, want %d, %d", where, place, pos, grp, p, g)
			}
			place++
		}
	}
}

// A grouped view shows what brute force says, through changes in place,
// appended and removed rows, new sorts and filters, and new groups.
func TestTableViewGroupsMatchBruteForce(t *testing.T) {
	for seed := range 1000 {
		r := rand.New(rand.NewPCG(uint64(seed), 7))
		row := func() tvRow { return tvRow{r.IntN(10), r.IntN(5)} }
		rows := make([]tvRow, r.IntN(60))
		for i := range rows {
			rows[i] = row()
		}
		v := NewTableView(tvCompare)
		var by []SortOrder
		var keep func(*tvRow) bool
		var key func(*tvRow) string
		var order []string
		for step := range 60 {
			switch r.IntN(9) {
			case 0, 1:
				for range r.IntN(4) + 1 {
					if len(rows) > 0 {
						i := r.IntN(len(rows))
						rows[i] = row()
						v.Changed(i)
					}
				}
			case 2:
				for range r.IntN(3) + 1 {
					rows = append(rows, row())
				}
			case 3:
				by = nil
				for range r.IntN(2) + 1 {
					by = append(by, SortOrder{Column: []string{"a", "b", "none"}[r.IntN(3)], Descending: r.IntN(2) == 1})
				}
			case 4:
				if r.IntN(3) == 0 {
					keep = nil
				} else {
					lim := r.IntN(10)
					keep = func(x *tvRow) bool { return x.a < lim }
				}
				v.SetFilter(keep)
			case 5:
				rows = rows[:r.IntN(len(rows)+1)]
			case 6, 7: // groups on, with an order naming some titles, or off
				order = nil
				if r.IntN(4) == 0 {
					key = nil
				} else {
					key = tvTitle
					for range r.IntN(4) {
						order = append(order, "g"+strconv.Itoa(r.IntN(7)))
					}
				}
				v.SetGroupBy(key, order...)
			case 8:
				rows = slices.Clone(rows)
			}
			v.Sync(rows, by...)
			want, groups := tvWantGrouped(rows, by, keep, key, order)
			if got := tvGot(v); !slices.Equal(got, want) {
				t.Fatalf("seed %d, step %d: got %v, want %v", seed, step, got, want)
			}
			if got := v.Groups(); !slices.Equal(got, groups) {
				t.Fatalf("seed %d, step %d: Groups() = %v, want %v", seed, step, got, groups)
			}
			checkPlaces(t, v, groups, "seed "+strconv.Itoa(seed)+", step "+strconv.Itoa(step))
		}
	}
}

// The groups the app names come first, in its order, once each; the others
// follow by title, whatever the sort; a title no row has makes no group.
func TestTableViewGroupsOrder(t *testing.T) {
	rows := []tvRow{{1, 2}, {2, 0}, {3, 1}, {4, 2}, {5, 3}}
	v := NewTableView(tvCompare)
	v.SetGroupBy(tvTitle, "g3", "g9", "g3", "g1")
	v.Sync(rows, SortOrder{Column: "a", Descending: true})
	want := []TableGroup{{"g3", 0, 1}, {"g1", 1, 1}, {"g0", 2, 1}, {"g2", 3, 2}}
	if got := v.Groups(); !slices.Equal(got, want) {
		t.Fatalf("Groups() = %v, want %v", got, want)
	}
	if got, wantRows := tvGot(v), []int{4, 2, 1, 3, 0}; !slices.Equal(got, wantRows) {
		t.Fatalf("rows = %v, want %v", got, wantRows)
	}
	v.Sync(rows, SortOrder{Column: "a"})
	if got := v.Groups(); !slices.Equal(got, want) {
		t.Fatalf("after a new sort, Groups() = %v: the groups moved", got)
	}
	if got, wantRows := tvGot(v), []int{4, 2, 1, 0, 3}; !slices.Equal(got, wantRows) {
		t.Fatalf("after a new sort, rows = %v, want %v", got, wantRows)
	}
}

// A view without groups has none, and its places are its positions.
func TestTableViewWithoutGroups(t *testing.T) {
	rows := []tvRow{{1, 2}, {2, 0}}
	v := NewTableView(tvCompare)
	v.Sync(rows)
	if v.Groups() != nil || v.Places() != 2 || v.PlaceOf(1) != 1 {
		t.Fatalf("Groups() = %v, Places() = %d, PlaceOf(1) = %d", v.Groups(), v.Places(), v.PlaceOf(1))
	}
	if pos, g := v.PosOf(1); pos != 1 || g != -1 {
		t.Fatalf("PosOf(1) = %d, %d", pos, g)
	}
}

// The version changes whenever the groups do: a row that changes group and
// stays at the same position moves the boundary, not the order.
func TestTableViewGroupsVersion(t *testing.T) {
	rows := []tvRow{{1, 0}, {2, 0}, {3, 1}}
	v := NewTableView(tvCompare)
	v.SetGroupBy(tvTitle)
	v.Sync(rows, SortOrder{Column: "a"})
	ver := v.Version()
	v.Sync(rows, SortOrder{Column: "a"})
	if v.Version() != ver {
		t.Fatal("the version changed with nothing")
	}
	rows[1].b = 1
	v.Changed(1)
	v.Sync(rows, SortOrder{Column: "a"})
	if got := tvGot(v); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("setup: rows = %v, want the order kept", got)
	}
	if v.Version() == ver {
		t.Fatal("a row changed group, the boundary moved and the version did not change")
	}
	ver = v.Version()
	v.SetGroupBy(nil)
	v.Sync(rows, SortOrder{Column: "a"})
	if v.Version() == ver || v.Groups() != nil {
		t.Fatalf("SetGroupBy(nil): version %d -> %d, Groups() = %v", ver, v.Version(), v.Groups())
	}
	ver = v.Version()
	v.SetGroupBy(tvTitle, "g1")
	v.Sync(rows, SortOrder{Column: "a"})
	if v.Version() == ver {
		t.Fatal("new groups did not change the version")
	}
}

// The key of the group is asked once for each row when they are placed and
// once for each row changed, never while sorting.
func TestTableViewGroupsAskTheKeyOncePerRow(t *testing.T) {
	rows := tvBenchRows(1000)
	calls := 0
	v := NewTableView(tvCompare)
	v.SetGroupBy(func(x *tvRow) string { calls++; return tvTitle(x) })
	v.Sync(rows, SortOrder{Column: "a"})
	if calls != len(rows) {
		t.Fatalf("the key was asked %d times for %d rows", calls, len(rows))
	}
	calls = 0
	rows[10].b, rows[20].b = 6, 6
	v.Changed(10, 20)
	v.Sync(rows, SortOrder{Column: "a"})
	if calls != 2 {
		t.Fatalf("two rows changed and the key was asked %d times", calls)
	}
	calls = 0
	v.Sync(rows, SortOrder{Column: "a"})
	if calls != 0 {
		t.Fatalf("a Sync with nothing changed asked the key %d times", calls)
	}
}

// A Sync with nothing changed allocates nothing, with groups either.
func TestTableViewGroupsIdleSyncAllocatesNothing(t *testing.T) {
	rows := tvBenchRows(1000)
	v := NewTableView(tvCompare)
	v.SetGroupBy(tvTitle)
	v.Sync(rows, SortOrder{Column: "a"})
	if n := testing.AllocsPerRun(100, func() { v.Sync(rows, SortOrder{Column: "a"}) }); n != 0 {
		t.Fatalf("an idle Sync allocated %v times", n)
	}
}
