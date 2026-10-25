package ui

import (
	"slices"
	"strings"
)

// A TableGroup is a run of the rows a TableView shows that SetGroupBy put
// under one title: the rows at positions Start to Start+Len-1.
type TableGroup struct {
	Title string
	Start int
	Len   int
}

// tvGroups is what a TableView keeps of its grouping. A zero one is the view
// without groups, which costs it nothing.
type tvGroups[T any] struct {
	by    func(*T) string
	order []string // the titles the app named, as it gave them
	// gid is each row's group, an index into titles; rank is that group's
	// place among the others, so that sorting by it is a lookup. The first
	// len(named) ids are the named titles, whose rank is their id; the
	// others are ranked by title.
	gid    []int32
	rank   []int32
	titles []string
	ids    map[string]int32
	named  int
	// last is the title asked last and its id: rows of one group are often
	// side by side in the slice.
	last   string
	lastID int32
	// shown are the groups of the order shown; spare is the buffer the next
	// ones are made in.
	shown, spare TableGroups
	ranked       []int32 // scratch for rank
	// ends and dealt are the scratch sortRows deals rows to their groups in.
	ends  []int
	dealt []int
}

// SetGroupBy puts the rows under the groups key names, from the next Sync:
// the group is the first key rows are sorted by, before the columns of the
// sort, so that the rows of a group show together, in the order the sort
// gives them. The groups named in order come first, in that order, and the
// others after them, by title; a title no row has makes no group. key gets a
// pointer to a row of the slice, which it must not change nor keep; it is
// asked once for each row when all rows are placed, and for each row
// Changed names, never while sorting. As functions cannot be compared, the
// next Sync places all rows again: call it when the groups change, not in
// every frame. nil shows no groups.
//
// Groups lists the groups shown, and Places, PlaceOf and PosOf tell where a
// table that puts a header before each group shows the rows.
func (v *TableView[T]) SetGroupBy(key func(*T) string, order ...string) {
	v.groups.by, v.groups.order = key, slices.Clone(order)
	v.reset = true
}

// Groups returns the groups the rows shown are under, in order, each with the
// positions of its rows; nil without SetGroupBy. The slice is the view's, and
// is valid until the next Sync.
func (v *TableView[T]) Groups() TableGroups { return v.groups.shown }

// Places returns how many lines a table shows for the view: the rows, and a
// header before each group.
func (v *TableView[T]) Places() int { return len(v.perm) + len(v.groups.shown) }

// PlaceOf returns the place of the row shown at position pos: see
// TableGroups.PlaceOf.
func (v *TableView[T]) PlaceOf(pos int) int { return v.groups.shown.PlaceOf(pos) }

// PosOf returns the position of the row shown at place, and its group: see
// TableGroups.PosOf.
func (v *TableView[T]) PosOf(place int) (pos, group int) { return v.groups.shown.PosOf(place) }

// TableGroups are the groups of a TableView as Groups returns them, which a
// table keeps for the frame it shows when the view moves on.
type TableGroups []TableGroup

// PlaceOf returns the place of the row at position pos: after the headers of
// the groups up to its own. Without groups the place is the position.
func (gs TableGroups) PlaceOf(pos int) int {
	if len(gs) == 0 {
		return pos
	}
	// The last group that starts at pos or before.
	lo, hi := 0, len(gs)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if gs[m].Start <= pos {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return pos + lo
}

// PosOf returns the position of the row at place, and the group it is in; for
// a header the position is -1, and the group the one it heads. Without groups
// the place is the position and the group -1.
func (gs TableGroups) PosOf(place int) (pos, group int) {
	if len(gs) == 0 {
		return place, -1
	}
	// The last group whose header is at place or before: group k's header is
	// at gs[k].Start+k.
	lo, hi := 0, len(gs)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if gs[m].Start+m <= place {
			lo = m + 1
		} else {
			hi = m
		}
	}
	k := lo - 1
	if k < 0 {
		return -1, -1
	}
	if place == gs[k].Start+k {
		return -1, k
	}
	return place - k - 1, k
}

// order returns how rows sort: by the rank of their group, if any, then as
// columnsOrder says.
func (v *TableView[T]) order(rows []T) func(i, j int) int {
	by := v.columnsOrder(rows)
	g := &v.groups
	if g.by == nil {
		return by
	}
	gid, rank := g.gid, g.rank
	return func(i, j int) int {
		if d := rank[gid[i]] - rank[gid[j]]; d != 0 {
			return int(d)
		}
		return by(i, j)
	}
}

// sortRows sorts idx, row indices in ascending order, as order says. With
// groups it deals the rows out to their groups first, in one pass that keeps
// their order, and sorts each group by the columns alone: no comparison reads
// a row's group, and the groups, smaller, take fewer comparisons than all.
func (v *TableView[T]) sortRows(idx []int, rows []T) {
	g := &v.groups
	if g.by == nil {
		slices.SortFunc(idx, v.columnsOrder(rows))
		return
	}
	n := len(g.rank)
	g.ends = slices.Grow(g.ends[:0], n)[:n]
	clear(g.ends)
	for _, i := range idx {
		g.ends[g.rank[g.gid[i]]]++
	}
	// ends[r] is where the rows of rank r start, then, as they are dealt, where
	// they end.
	at := 0
	for r, c := range g.ends {
		g.ends[r] = at
		at += c
	}
	g.dealt = slices.Grow(g.dealt[:0], len(idx))[:len(idx)]
	for _, i := range idx {
		r := g.rank[g.gid[i]]
		g.dealt[g.ends[r]] = i
		g.ends[r]++
	}
	copy(idx, g.dealt)
	if len(v.keys) == 0 {
		return // the rows of a group are in the slice's order already
	}
	by := v.columnsOrder(rows)
	from := 0
	for _, end := range g.ends {
		if end-from > 1 {
			slices.SortFunc(idx[from:end], by)
		}
		from = end
	}
}

// placeAll gives every row its group, from scratch, for a rebuild.
func (v *TableView[T]) placeAll(rows []T) {
	g := &v.groups
	if g.by == nil {
		g.gid, g.titles, g.ids, g.rank = nil, nil, nil, nil
		return
	}
	g.titles, g.last, g.lastID = g.titles[:0], "", -1
	if g.ids == nil {
		g.ids = map[string]int32{}
	} else {
		clear(g.ids)
	}
	for _, title := range g.order {
		g.id(title)
	}
	g.named = len(g.titles)
	g.gid = slices.Grow(g.gid[:0], len(rows))[:len(rows)]
	for i := range rows {
		g.gid[i] = g.id(g.by(&rows[i]))
	}
	g.rankTitles()
}

// placeDirty gives the rows changed (once each, in the slice) their group.
func (v *TableView[T]) placeDirty(rows []T, dirty []int) {
	g := &v.groups
	if g.by == nil {
		return
	}
	for len(g.gid) < len(rows) {
		g.gid = append(g.gid, 0)
	}
	titles := len(g.titles)
	for _, i := range dirty {
		g.gid[i] = g.id(g.by(&rows[i]))
	}
	if len(g.titles) != titles {
		g.rankTitles()
	}
}

// id returns the id of title, new if no row had it yet.
func (g *tvGroups[T]) id(title string) int32 {
	if g.lastID >= 0 && g.last == title {
		return g.lastID
	}
	id, ok := g.ids[title]
	if !ok {
		id = int32(len(g.titles))
		g.ids[title] = id
		g.titles = append(g.titles, title)
	}
	g.last, g.lastID = title, id
	return id
}

// rankTitles ranks the groups: the named first, in the order given, then the
// others by title, as a text column sorts. A title that came later does not
// change the order of those before it, so rows already placed stay placed.
func (g *tvGroups[T]) rankTitles() {
	n := len(g.titles)
	g.rank = slices.Grow(g.rank[:0], n)[:n]
	g.ranked = g.ranked[:0]
	for id := range int32(n) {
		if int(id) < g.named {
			g.rank[id] = id
		} else {
			g.ranked = append(g.ranked, id)
		}
	}
	slices.SortFunc(g.ranked, func(a, b int32) int { return strings.Compare(g.titles[a], g.titles[b]) })
	for k, id := range g.ranked {
		g.rank[id] = int32(g.named + k)
	}
}

// makeGroups makes the groups of the order perm, in spare: the rows of a
// group are side by side, so it looks for where each ends instead of going
// through the rows. It reports whether they differ from those shown.
func (v *TableView[T]) makeGroups() (changed bool) {
	g := &v.groups
	if g.by == nil {
		changed = g.shown != nil
		g.shown = nil
		return changed
	}
	perm, gid, rank := v.perm, g.gid, g.rank
	gs := g.spare[:0]
	for p := 0; p < len(perm); {
		r := rank[gid[perm[p]]]
		// The first row past p of another group.
		lo, hi := p+1, len(perm)
		for lo < hi {
			m := int(uint(lo+hi) >> 1)
			if rank[gid[perm[m]]] == r {
				lo = m + 1
			} else {
				hi = m
			}
		}
		gs = append(gs, TableGroup{Title: g.titles[gid[perm[p]]], Start: p, Len: lo - p})
		p = lo
	}
	changed = !slices.Equal(gs, g.shown)
	g.shown, g.spare = gs, g.shown
	return changed
}
