package ui

import "sort"

// headerRows is where the rows of a grid whose items include headers
// (GridState.Header) begin. The items fall in blocks: the items before the
// first header, then each header with the items after it up to the next. A
// header takes a row, and the items of a block fill rows of cols, the last of
// which may be short. It is made of the headers' places only, so a grid of a
// million items with a few headers keeps a few numbers.
type headerRows struct {
	known   bool  // heads is the headers of n items
	given   bool  // SetHeaders gave them: they are not asked of Header again
	ok      bool  // start was made for cols
	n, cols int   // the items and columns it was made for
	heads   []int // the headers, in order
	start   []int // start[b] is the first row of block b: len(heads)+1 of them
	rows    int
	ver     int // changes whenever the rows were made again
}

// isHeader reports whether item i heads a group.
func (s *GridState) isHeader(i int) bool { return s.Header != nil && s.Header(i) }

// ReloadHeaders makes the grid ask Header for every item again at the next
// frame. It is needed only when an item that was not a header becomes one
// and the number of items stays: the grid finds the others itself.
func (s *GridState) ReloadHeaders() { s.layout.known, s.layout.given = false, false }

// SetHeaders tells the grid which of n items are headers, in ascending order,
// each less than n (otherwise the layout is wrong, without a panic), so that it does not ask Header for every item to find them: for an app that
// knows, a million items with a few headers cost the headers, not a pass over
// the items. The grid takes a copy and trusts it, asking Header for none of
// them, until the number of items is not n (it then asks for every item, as
// without SetHeaders), ReloadHeaders is called, or SetHeaders is called again:
// call it whenever a header moves. Header is still required, and answers for
// the items the grid builds.
func (s *GridState) SetHeaders(items []int, n int) {
	l := &s.layout
	l.heads = append(l.heads[:0], items...)
	l.n, l.known, l.given, l.ok = n, true, true, false
}

// prepare makes the layout of the grid's rows for n items in cols columns.
// It asks Header for every item only when the number of items changed, or a
// header it knew is none, or ReloadHeaders was called; a header that moved
// leaves a place where there is none now, so the grid finds it by asking
// for the headers it knew.
func (s *GridState) prepare(n, cols int) {
	l := &s.layout
	if s.Header == nil {
		l.known, l.ok = false, false
		return
	}
	if l.known && l.n == n {
		for _, h := range l.heads {
			if l.given || s.Header(h) {
				continue
			}
			l.known = false
			break
		}
	} else {
		l.known, l.given = false, false
	}
	if !l.known {
		l.heads = l.heads[:0]
		for i := range n {
			if s.Header(i) {
				l.heads = append(l.heads, i)
			}
		}
		l.n, l.known, l.ok = n, true, false
	}
	if l.ok && l.cols == cols {
		return
	}
	l.cols = cols
	l.start = l.start[:0]
	row := 0
	for b := 0; b <= len(l.heads); b++ {
		l.start = append(l.start, row)
		lo, hi := l.segment(b)
		if b > 0 {
			row++
		}
		row += (hi - lo + cols - 1) / cols
	}
	l.rows, l.ok = row, true
	l.ver++
}

// segment returns the items of block b that are not its header: from lo to
// hi, exclusive.
func (l *headerRows) segment(b int) (lo, hi int) {
	hi = l.n
	if b < len(l.heads) {
		hi = l.heads[b]
	}
	if b > 0 {
		lo = l.heads[b-1] + 1
	}
	return lo, hi
}

// rowCount returns the number of rows of the grid of n items in cols columns.
func (s *GridState) rowCount(n, cols int) int {
	if l := &s.layout; s.Header != nil && l.ok {
		return l.rows
	}
	return (n + cols - 1) / cols
}

// itemRow returns the row of item i.
func (s *GridState) itemRow(i int) int {
	cols := max(s.cols, 1)
	l := &s.layout
	if s.Header == nil || !l.ok {
		return i / cols
	}
	i = max(0, min(i, l.n-1))
	k := sort.SearchInts(l.heads, i)
	if k < len(l.heads) && l.heads[k] == i {
		return l.start[k+1]
	}
	lo, _ := l.segment(k)
	row := l.start[k] + (i-lo)/cols
	if k > 0 {
		row++
	}
	return row
}

// rowItems returns the items of row r: from first to end, exclusive, and
// whether the row is a header's.
func (s *GridState) rowItems(r, n int) (first, end int, header bool) {
	cols := max(s.cols, 1)
	l := &s.layout
	if s.Header == nil || !l.ok {
		first = r * cols
		return first, min(first+cols, n), false
	}
	b := sort.SearchInts(l.start, r+1) - 1
	off := r - l.start[b]
	if b > 0 {
		if off == 0 {
			return l.heads[b-1], l.heads[b-1] + 1, true
		}
		off--
	}
	lo, hi := l.segment(b)
	first = lo + off*cols
	return first, min(first+cols, hi), false
}

// firstItem and lastItem return the first and the last item that is not a
// header, or -1.
func (s *GridState) firstItem(n int) int { return s.skip(-1, 1, n) }
func (s *GridState) lastItem(n int) int  { return s.skip(n, -1, n) }

// skip returns the first item from i on by d that is not a header, or -1.
func (s *GridState) skip(i, d, n int) int {
	for i += d; i >= 0 && i < n; i += d {
		if !s.isHeader(i) {
			return i
		}
	}
	return -1
}

// beside returns the item next to item at, d (1 or -1) places on, passing
// the headers by, or -1.
func (s *GridState) beside(at, d, n int) int { return s.skip(at, d, n) }

// vertical returns the item in the row below (d 1) or above (d -1) the row
// of item at, in the same column or the last of a shorter row, passing the
// header rows by, or -1 when there is none.
func (s *GridState) vertical(at, d, n int) int {
	first, _, _ := s.rowItems(s.itemRow(at), n)
	col := at - first
	rows := s.rowCount(n, max(s.cols, 1))
	for r := s.itemRow(at) + d; r >= 0 && r < rows; r += d {
		if f, end, header := s.rowItems(r, n); !header && end > f {
			return min(f+col, end-1)
		}
	}
	return -1
}
