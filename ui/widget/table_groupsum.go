package widget

import "github.com/egoist/mygo/ui"

// enableGroupSummaries gives the groups' header lines their totals: a
// column has a GroupSummary. As enableSummaries, it is only the adapter that
// runs a column's GroupSummary over a group's rows; the state and the work
// are gridGroupSums'. Before the table is shared.
func (t *DataGridViewWidget[T, K]) enableGroupSummaries() {
	g := &gridGroupSums{}
	for i := range t.cols {
		g.titles = append(g.titles, t.cols[i].Title)
	}
	g.summary = func(start, n, i int) string {
		s := t.cols[i].GroupSummary
		if s == nil {
			return ""
		}
		return s(func(yield func(*T) bool) { // the group's rows shown, in their order
			for p := start; p < start+n; p++ {
				if !yield(&t.src.rows[t.view.Row(p)]) {
					return
				}
			}
		})
	}
	t.groupSums = g
}

// gridGroupSums is the state of the totals in a DataGridView's group header
// lines. Their cells are placed where the header's were laid out in the last
// frame, as the footer's (gridFooter); a paint that finds the header's cells,
// or the lines, laid out elsewhere asks for one more frame.
type gridGroupSums struct {
	heads []headCell // the header's cells of this build, in order (headerCell)
	// summary runs the GroupSummary of the grid's column i over the n rows
	// shown from position start, "" for none; titles are the columns' titles.
	summary func(start, n, i int) string
	titles  []string
	// texts are the totals by the place of the group, then by column, as of
	// the view's version atVer and the source's change atSeq.
	texts        [][]string
	known        bool
	atVer, atSeq uint64
	placed       []ui.Rect // the bounds of the header's cells the lines were placed by
	// xs are where each group's line started, by the group's place, as last
	// painted: a line pinned at the list's top does not scroll sideways with
	// the others. lineX is the start last painted of any; checks are the
	// lines' paints (checkPlaces), by place, each made once.
	xs     []float32
	lineX  float32
	checks []func(*ui.Painter, ui.Rect)
	c      *ui.Context
	asked  int // test hook: how many frames the paint asked for
	cells  int // test hook: how many totals' cells were built
}

// update works the totals out again when the rows shown changed, for the
// groups gs of the view; without groups none runs. A GroupSummary's panic is
// reported as a handler's, on the event goroutine, and leaves its text
// empty. Under mu, while the table builds, before its rows.
func (g *gridGroupSums) update(gs ui.TableGroups, ver, seq uint64) {
	if g.known && ver == g.atVer && seq == g.atSeq {
		return
	}
	g.known, g.atVer, g.atSeq = true, ver, seq
	g.texts = make([][]string, len(gs)) // new: a line ui builds late reads the frame's
	for k, gr := range gs {
		texts := make([]string, len(g.titles))
		for i := range texts {
			if v, stack, panicked := runColumnFn(func() { texts[i] = g.summary(gr.Start, gr.Len, i) }); panicked {
				Post(func() { report(v, stack) })
			}
		}
		g.texts[k] = texts
	}
}

// header builds the header line of the k-th group, as groupHeader does, with
// each total under its column's header cell, as wide. Under mu, while
// building.
func (g *gridGroupSums) header(c *ui.Context, k int, text string) {
	th := c.Theme()
	line := ui.Row(c).Grow(1).AlignItems(ui.Center).PaddingStart(th.Space(2)).Background(th.SurfaceHover).Radius(th.Radius)
	for len(g.checks) <= k { // each made once: a func made in every frame allocates
		j := len(g.checks)
		g.checks = append(g.checks, func(_ *ui.Painter, r ui.Rect) { g.checkPlaces(j, r) })
		g.xs = append(g.xs, g.lineX)
	}
	g.c = c
	if !c.Measuring() {
		line.Draw(g.checks[k])
	}
	// Where the line starts, as last painted (a line new in view, as the
	// lines last painted). Not line.Bounds, which allocates for an element
	// new in every frame.
	x := g.xs[k]
	g.placed = g.placed[:0]
	for i := range g.heads {
		g.placed = append(g.placed, g.heads[i].cell.Bounds())
	}
	line.Children(func() {
		ui.Text(c, text).Bold().TextColor(th.TextMuted).SingleLine()
		texts := g.texts[k] // made at this frame's build for its groups, which a line built late shows too
		for i := range g.heads {
			hc, b := &g.heads[i], g.placed[i]
			total := g.textOf(hc, texts)
			// An empty total builds no cell. Under a header cell not laid out
			// yet a cell is 0 wide: the paint asks for the frame that places it.
			if total == "" {
				continue
			}
			g.cells++
			ui.Row(c).Absolute().Left(b.X-x).Top(0).Bottom(0).Width(b.W).Padding(0, th.Space(2.5)).
				AlignItems(ui.Center).Justify(hc.align).Clip().Children(func() {
				ui.Text(c, total).Bold().TextColor(th.TextMuted).SingleLine()
			})
		}
	})
}

// textOf returns the total of hc's column among texts, the totals of a
// group by column; "" for the index column, whose title "#" no column has.
func (g *gridGroupSums) textOf(hc *headCell, texts []string) string {
	for i, title := range g.titles {
		if title == hc.title {
			return texts[i]
		}
	}
	return ""
}

// checkPlaces is the paint of the k-th group's header line, r its box: it
// asks for one more frame when the header's cells, or the line, are laid out
// elsewhere than the totals were placed by, as after a column was resized,
// moved or scrolled, or the line was pinned at the list's top, which does not
// scroll sideways.
func (g *gridGroupSums) checkPlaces(k int, r ui.Rect) {
	moved := r.X != g.xs[k] // placed holds a bound for each of heads: header made both
	for i := 0; !moved && i < len(g.heads); i++ {
		b, p := g.heads[i].cell.Bounds(), g.placed[i]
		moved = b.X != p.X || b.W != p.W
	}
	if moved {
		g.xs[k], g.lineX = r.X, r.X
		g.asked++
		g.c.Invalidate()
	}
}
