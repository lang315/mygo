package ui

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// keepScene is a table of n rows with a version for each, as a list that
// keeps rows is given: the builds of each row's first cell are counted.
type keepScene struct {
	t       *testing.T
	tt      *Tester
	n       int
	vers    []uint64
	builds  []int
	checked Selection[int]
	chosen  Selection[int]
	list    ListState
	handle  Handle
	light   *Theme
	dark    *Theme
	theme   *Theme
	bind    int     // the row whose first cell binds handle, -1 for none
	hide    int     // 1: an element above the table is Invisible, 2: Hidden
	omit    bool    // the table is not built
	colw    float32 // the width of the first column, 0 for the room that is left
	rtl     bool    // the table is built right to left
}

// newKeepScene makes the scene, with the rows kept (keep) or built in every
// frame, and shows 3 frames, so that rows that can be kept are.
func newKeepScene(t *testing.T, n int, keep bool) *keepScene {
	s := &keepScene{t: t, n: n, vers: make([]uint64, n), builds: make([]int, n), bind: -1, light: LightTheme(), dark: DarkTheme()}
	s.theme = s.light
	s.list = ListState{
		Key:        func(i int) any { return i },
		Label:      func(i int) string { return fmt.Sprint("row", i) },
		Checked:    &s.checked,
		Selection:  &s.chosen,
		RowVersion: func(i int) uint64 { return s.vers[i] },
	}
	s.tt = coreNewTester(func(c *context) {
		c.SetTheme(s.theme)
		top := coreBox(c).Fill()
		switch s.hide {
		case 1:
			top.Invisible()
		case 2:
			top.Hidden(true)
		}
		if s.rtl {
			top.Direction(RightToLeft)
		}
		top.Children(func() {
			if s.omit {
				return
			}
			cols := []TableColumn{{Title: "Name", Width: s.colw}}
			coreTable(c, &s.list, cols, n, func(row, col int) {
				s.builds[row]++
				e := coreText(c, fmt.Sprint("row", row, " v", s.vers[row]))
				if row == s.bind {
					wrapElement(e).Bind(&s.handle)
				}
			}).Grow(1)
		})
	}, 400, 600)
	s.tt.rt.noRetain = !keep
	s.tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for range 3 {
		s.tt.Frame()
	}
	return s
}

// frame runs a frame and returns the rows built in it, with how many times.
func (s *keepScene) frame() (built map[int]int) {
	before := slicesClone(s.builds)
	s.tt.Frame()
	return s.diff(before)
}

func slicesClone(v []int) []int { return append([]int(nil), v...) }

func (s *keepScene) diff(before []int) map[int]int {
	built := map[int]int{}
	for i := range before {
		if d := s.builds[i] - before[i]; d != 0 {
			built[i] = d
		}
	}
	return built
}

func (s *keepScene) pixels() []byte { return s.tt.Image().Pix }

// shown returns how many rows are in view.
func (s *keepScene) shown() int {
	n := 0
	for i := range s.n {
		if _, ok := s.tt.Find(fmt.Sprint("row", i, " v", s.vers[i])); ok {
			n++
		}
	}
	return n
}

// checkBox returns the middle of the check box of row i.
func (s *keepScene) checkBox(i int) (x, y float32) {
	s.t.Helper()
	for _, n := range s.tt.h.access.Nodes {
		if n.Role == platform.RoleCheckBox && n.Label == fmt.Sprint("Check row", i) {
			return float32(n.Bounds.X + n.Bounds.W/2), float32(n.Bounds.Y + n.Bounds.H/2)
		}
	}
	s.t.Fatalf("no check box of row %d", i)
	return 0, 0
}

// rowAt returns the middle of the text of row i.
func (s *keepScene) rowAt(i int) (x, y float32) {
	s.t.Helper()
	r, ok := s.tt.Find(fmt.Sprint("row", i, " v", s.vers[i]))
	if !ok {
		s.t.Fatalf("row %d is not in view", i)
	}
	return r.X + r.W/2, r.Y + r.H/2
}

// same fails unless the scenes show the same picture.
func same(t *testing.T, what string, a, b *keepScene) {
	t.Helper()
	if !bytes.Equal(a.pixels(), b.pixels()) {
		t.Fatalf("%s: the rows kept show another picture than the rows built", what)
	}
}

func onlyRows(t *testing.T, what string, got map[int]int, want ...int) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: rows built %v, want %v", what, got, want)
		return
	}
	for _, r := range want {
		if got[r] != 1 {
			t.Errorf("%s: rows built %v, want %v", what, got, want)
			return
		}
	}
}

func TestRetainIdleKeepsEveryRowInView(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	shown := a.shown()
	if shown < 10 {
		t.Fatalf("%d rows in view: the test shows nothing", shown)
	}
	kept := a.tt.rt.rowsKept
	built := a.frame()
	b.frame()
	if len(built) != 0 {
		t.Errorf("an idle frame built the rows %v", built)
	}
	if got := a.tt.rt.rowsKept - kept; got < shown {
		t.Errorf("an idle frame kept %d rows of the %d in view", got, shown)
	}
	same(t, "idle", a, b)
}

func TestRetainABumpedVersionBuildsThatRowOnly(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	a.vers[3]++
	b.vers[3]++
	onlyRows(t, "version of row 3 bumped", a.frame(), 3)
	b.frame()
	same(t, "bump", a, b)
	if !a.tt.HasText("row3 v1") {
		t.Errorf("row 3 does not show its new content: %q", a.tt.Texts())
	}
	onlyRows(t, "the frame after", a.frame())
}

func TestRetainHoverBuildsTheRowsHoveredAndLeft(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	x, y := a.rowAt(5)
	a.tt.Move(x, y)
	b.tt.Move(x, y)
	before := slicesClone(a.builds)
	a.tt.Frame()
	b.tt.Frame()
	if got := a.diff(before); got[5] == 0 {
		t.Errorf("row 5 hovered: rows built %v", got)
	}
	x, y = a.rowAt(6)
	before = slicesClone(a.builds)
	a.tt.Move(x, y)
	b.tt.Move(x, y)
	got := a.diff(before)
	for r := range got {
		if r != 5 && r != 6 {
			t.Errorf("the pointer went from row 5 to row 6: row %d built too (%v)", r, got)
		}
	}
	same(t, "hover", a, b)
	a.frame()
	onlyRows(t, "the pointer stays on row 6", a.frame(), 6)
}

// A click on the check box of a row that is kept, and Space on the row, check
// it as they do for a row built in every frame.
func TestRetainACheckBoxOfAKeptRowChecks(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	x, y := a.checkBox(2)
	a.tt.ClickAt(x, y)
	b.tt.ClickAt(x, y)
	if !a.checked.Has(2) || !b.checked.Has(2) {
		t.Fatalf("the click on the check box did not check row 2: kept %v, built %v", a.checked.Has(2), b.checked.Has(2))
	}
	same(t, "click", a, b)
	a.frame()
	a.frame()
	b.frame()
	b.frame()
	// Space on the row chosen.
	x, y = a.rowAt(4)
	a.tt.ClickAt(x, y)
	b.tt.ClickAt(x, y)
	a.tt.Key(0, KeySpace)
	b.tt.Key(0, KeySpace)
	if a.checked.Has(4) != b.checked.Has(4) || !a.checked.Has(4) {
		t.Fatalf("Space on row 4: kept %v, built %v, want both checked", a.checked.Has(4), b.checked.Has(4))
	}
	same(t, "space", a, b)
	// And the box can be unchecked again, from a row kept.
	a.frame()
	a.frame()
	b.frame()
	b.frame()
	x, y = a.checkBox(2)
	a.tt.ClickAt(x, y)
	b.tt.ClickAt(x, y)
	if a.checked.Has(2) || b.checked.Has(2) {
		t.Fatalf("a second click on row 2's check box left it checked: kept %v, built %v", a.checked.Has(2), b.checked.Has(2))
	}
	same(t, "second click", a, b)
}

// A row out of view for one frame had its states pruned: it is built when it
// comes back, not linked.
func TestRetainARowOutOfViewForAFrameIsBuiltWhenItComesBack(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	x, y := a.rowAt(15) // the pointer stays on another row, which is hovered in view
	a.tt.Scroll(x, y, 0, 3000)
	b.tt.Scroll(x, y, 0, 3000)
	if _, ok := a.tt.Find("row5 v0"); ok {
		t.Fatal("row 5 is still in view: the test shows nothing")
	}
	before := slicesClone(a.builds)
	a.tt.Scroll(x, y, 0, -3000)
	b.tt.Scroll(x, y, 0, -3000)
	got := a.diff(before)
	if got[5] == 0 {
		t.Errorf("row 5 came back after a frame out of view: rows built %v, want it built", got)
	}
	same(t, "back", a, b)
	onlyRows(t, "the frame after (row 15 is hovered)", a.frame(), 15)
}

func TestRetainARowWithAnAppHandleIsNeverKept(t *testing.T) {
	a := newKeepScene(t, 200, true)
	a.bind = 2
	a.vers[2]++ // so that it is built, and binds the Handle
	a.tt.Frame()
	for range 3 {
		got := a.frame()
		if got[2] != 1 || len(got) != 1 {
			t.Errorf("a row binds the app's Handle: rows built %v, want only row 2", got)
		}
	}
}

func TestRetainATheThemeChangeBuildsEveryRow(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	shown := a.shown()
	a.theme, b.theme = a.dark, b.dark
	got := a.frame()
	b.frame()
	if len(got) < shown {
		t.Errorf("a new theme: %d rows built of the %d in view", len(got), shown)
	}
	same(t, "dark", a, b)
	// And the engine's themes starting over does the same.
	a.theme, b.theme = a.light, b.light
	a.tt.Frame()
	b.tt.Frame()
	a.frame()
	a.tt.rt.themeGen++
	if got := a.frame(); len(got) < shown {
		t.Errorf("themes started over: %d rows built of the %d in view", len(got), shown)
	}
}

func TestRetainANeighbourChosenBuildsTheRowsAround(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	a.chosen.Add(7)
	b.chosen.Add(7)
	onlyRows(t, "row 7 chosen", a.frame(), 6, 7, 8)
	b.frame()
	same(t, "chosen", a, b)
	a.chosen.Add(8)
	b.chosen.Add(8)
	onlyRows(t, "row 8 chosen too", a.frame(), 7, 8, 9)
	b.frame()
	same(t, "chosen two", a, b)
}

func TestRetainNothingIsKeptWhileTheInspectorIsOpen(t *testing.T) {
	a := newKeepScene(t, 200, true)
	shown := a.shown()
	a.tt.rt.insp.open = true
	kept := a.tt.rt.rowsKept
	if got := a.frame(); len(got) < shown {
		t.Errorf("the inspector is open: %d rows built of the %d in view", len(got), shown)
	}
	if a.tt.rt.rowsKept != kept {
		t.Errorf("the inspector is open and %d rows were kept", a.tt.rt.rowsKept-kept)
	}
	a.tt.rt.insp.open = false
	a.tt.Frame()
	a.tt.Frame()
	if got := a.frame(); len(got) != 0 {
		t.Errorf("the inspector closed and rows are still built: %v", got)
	}
}

func TestRetainWithoutARowVersionNothingIsKept(t *testing.T) {
	a := newKeepScene(t, 200, true)
	a.list.RowVersion = nil
	shown := a.shown()
	for range 3 {
		a.tt.Frame()
	}
	kept := a.tt.rt.rowsKept
	if got := a.frame(); len(got) < shown {
		t.Errorf("no RowVersion: %d rows built of the %d in view", len(got), shown)
	}
	if a.tt.rt.rowsKept != kept {
		t.Errorf("no RowVersion and %d rows were kept", a.tt.rt.rowsKept-kept)
	}
}

// An app that gives two rows a key makes them share a state; a row that is
// kept does not overwrite the copy of the other with its own.
func TestRetainRowsThatShareAKeyKeepNoWrongCopy(t *testing.T) {
	log.SetOutput(io.Discard) // the repeated keys are logged
	defer log.SetOutput(os.Stderr)
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	for _, s := range []*keepScene{a, b} {
		s.tt.rt.strict = false
		s.list.Key = func(i int) any { return i / 2 }
	}
	for range 4 {
		a.tt.Frame()
		b.tt.Frame()
		same(t, "rows that share keys", a, b)
	}
	for _, s := range []*keepScene{a, b} {
		s.list.Key = func(i int) any { return i }
	}
	for range 4 {
		a.tt.Frame()
		b.tt.Frame()
		same(t, "keys back", a, b)
	}
}

// A row captured, then built without being captured (the pointer is on it, and
// its check changes meanwhile), has no record when the pointer leaves: it is
// built, not linked from the capture before.
func TestRetainARowBuiltWithoutCaptureLeavesNoOlderRecord(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	x, y := a.rowAt(5)
	for _, s := range []*keepScene{a, b} {
		s.tt.Move(x, y) // B: hovered, so not captured
		s.checked.Add(5)
		s.tt.Frame()
	}
	if a.list.frame.kept[5] != nil {
		t.Fatal("row 5 was built hovered, and the record of the frame before is still there")
	}
	same(t, "B", a, b)
	// C: the pointer leaves, to the header.
	before := slicesClone(a.builds)
	a.tt.Move(200, 8)
	b.tt.Move(200, 8)
	if got := a.diff(before); got[5] == 0 {
		t.Errorf("the pointer left row 5: rows built %v, want row 5 built", got)
	}
	same(t, "C", a, b)
	if !a.checked.Has(5) {
		t.Error("row 5 lost its check")
	}
	// And a frame later it is kept again.
	a.frame()
	onlyRows(t, "D", a.frame())
}

// An element above the table is Invisible or Hidden while a row's data changes
// with its version: the row shows the change when the element shows again.
func TestRetainARowChangedWhileAnAncestorIsHidden(t *testing.T) {
	for _, hide := range []int{1, 2} {
		a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
		for _, s := range []*keepScene{a, b} {
			s.hide = hide
			s.tt.Frame()
			s.tt.Frame()
			s.vers[4]++
			s.tt.Frame()
			s.tt.Frame()
			s.hide = 0
			s.tt.Frame()
		}
		same(t, fmt.Sprint("shown again after hide ", hide), a, b)
		if !a.tt.HasText("row4 v1") {
			t.Errorf("hide %d: row 4 does not show its new content: %q", hide, a.tt.Texts())
		}
		a.frame()
		onlyRows(t, "the frame after", a.frame())
	}
}

// A request of assistive technology to open a row (state.expand) is read as
// the row builds: a row it is made of is built, not linked.
func TestRetainARowAskedToExpandIsBuilt(t *testing.T) {
	a := newKeepScene(t, 200, true)
	var holder *state
	for _, r := range a.list.frame.rows {
		if r.i == 5 {
			holder = r.e.st
		}
	}
	if holder == nil {
		t.Fatal("row 5 is not in view")
	}
	holder.expand = 1
	if got := a.frame(); got[5] == 0 {
		t.Errorf("row 5 was asked to expand: rows built %v, want row 5 built", got)
	}
}

// A table that is not built for a frame (its page is not shown) had its rows'
// states pruned: when it is built again its rows are built, not linked.
func TestRetainATableNotBuiltForAFrameBuildsItsRowsAgain(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	shown := a.shown()
	for _, s := range []*keepScene{a, b} {
		s.omit = true
		s.tt.Frame()
		s.omit = false
	}
	if got := a.frame(); len(got) < shown {
		t.Errorf("the table is built again: %d rows built of the %d in view", len(got), shown)
	}
	b.tt.Frame()
	same(t, "built again", a, b)
	a.frame()
	onlyRows(t, "the frame after", a.frame())
}

// What a Table builds its rows with besides their data, the width of a column,
// is part of what a kept row was built with.
func TestRetainAColumnWidthChangeBuildsTheRows(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	shown := a.shown()
	for _, s := range []*keepScene{a, b} {
		s.colw = 120
	}
	if got := a.frame(); len(got) < shown {
		t.Errorf("a column got a width: %d rows built of the %d in view", len(got), shown)
	}
	b.tt.Frame()
	same(t, "width", a, b)
	a.frame()
	onlyRows(t, "the frame after", a.frame())
}

// The direction an element passes on to its children is copied as they are
// built.
func TestRetainADirectionChangeBuildsTheRows(t *testing.T) {
	a, b := newKeepScene(t, 200, true), newKeepScene(t, 200, false)
	shown := a.shown()
	for _, s := range []*keepScene{a, b} {
		s.rtl = true
	}
	if got := a.frame(); len(got) < shown {
		t.Errorf("right to left: %d rows built of the %d in view", len(got), shown)
	}
	b.tt.Frame()
	same(t, "rtl", a, b)
}
