package widget

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// file is a node of the tree of the cases: a path, the name the tree column
// shows, and the Size and Kind columns' texts.
type file struct{ Path, Name, Size, Kind string }

func sizeCol() Column[file] {
	return Column[file]{Title: "Size", Text: func(f *file) string { return f.Size }, Width: 80, Align: End}
}

func kindCol() Column[file] {
	return Column[file]{Title: "Kind", Text: func(f *file) string { return f.Kind }, Width: 100}
}

// plainFiles is the tree of the cases without columns: "src" (with "a.go"
// and "b.go") and "README".
func plainFiles() *TreeViewWidget[file, string] {
	tv := TreeView(func(f *file) string { return f.Path }, func(f *file) string { return f.Name })
	tv.SetRoots([]file{{"src", "src", "", "Folder"}, {"README", "README", "3 KB", "Text"}})
	tv.SetChildren("src", []file{{"src/a.go", "a.go", "1 KB", "Go"}, {"src/b.go", "b.go", "2 KB", "Go"}})
	return tv
}

// filesTree is the tree of the cases in the columns Name, Size and Kind.
func filesTree() *TreeViewWidget[file, string] {
	return plainFiles().SetColumns("Name", sizeCol(), kindCol())
}

// filesTester shows w in a window of 500 by 300.
func filesTester(w Control) *ui.Tester {
	tt := ui.NewTester(content(w), 500, 300)
	tt.Frame()
	return tt
}

// tcFind returns what assistive technology sees of the element of role r
// named name in the last frame, failing without one.
func tcFind(t *testing.T, tt *ui.Tester, r ui.Role, name string) ui.AccessInfo {
	t.Helper()
	a, ok := tt.AccessByRole(r, name)
	if !ok {
		t.Fatalf("no %v named %q in %q", r, name, tt.Texts())
	}
	return a
}

func tcHas(tt *ui.Tester, r ui.Role, name string) bool {
	_, ok := tt.AccessByRole(r, name)
	return ok
}

// tcHeaders returns the titles of the column headers shown, in order.
func tcHeaders(tt *ui.Tester) []string {
	var hs []string
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleColumnHeader {
			hs = append(hs, a.Name)
		}
	}
	return hs
}

// tcCell returns the bounds of the cell holding the text s.
func tcCell(t *testing.T, tt *ui.Tester, s string) ui.Rect {
	t.Helper()
	b := tcFind(t, tt, ui.RoleText, s).Bounds
	cx, cy := b.X+b.W/2, b.Y+b.H/2
	for _, a := range tt.AccessAll() {
		if r := a.Bounds; a.Role == ui.RoleCell && cx >= r.X && cx < r.X+r.W && cy >= r.Y && cy < r.Y+r.H {
			return r
		}
	}
	t.Fatalf("no cell holds %q at %v", s, b)
	return ui.Rect{}
}

// tcRowCells returns the cells of the row that holds the text s, left to right.
func tcRowCells(t *testing.T, tt *ui.Tester, s string) []ui.Rect {
	t.Helper()
	b := tcFind(t, tt, ui.RoleText, s).Bounds
	cy := b.Y + b.H/2
	var cells []ui.Rect
	for _, a := range tt.AccessAll() {
		if r := a.Bounds; a.Role == ui.RoleCell && cy >= r.Y && cy < r.Y+r.H {
			cells = append(cells, r)
		}
	}
	return cells
}

func withinDIP(a, b, tol float32) bool { return a-b <= tol && b-a <= tol }

// Case 1.
func TestTreeColumnsHeaderAndCells(t *testing.T) {
	tt := filesTester(filesTree())
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name Size Kind]" {
		t.Fatalf("headers %s, want [Name Size Kind]", got)
	}
	for _, row := range [][3]string{{"src", "", "Folder"}, {"README", "3 KB", "Text"}} {
		for i, s := range row {
			if s == "" {
				continue
			}
			h := tcFind(t, tt, ui.RoleColumnHeader, []string{"Name", "Size", "Kind"}[i]).Bounds
			if c := tcCell(t, tt, s); !withinDIP(c.X, h.X, 1) {
				t.Errorf("the cell of %q starts at x %v, its header %v: want within 1 DIP", s, c.X, h.X)
			}
		}
	}
}

// Case 2.
func TestTreeColumnsExpandIndentsInTheNameColumn(t *testing.T) {
	tv := filesTree()
	tt := filesTester(tv)
	tv.Expand("src")
	tt.Frame()
	src := tcFind(t, tt, ui.RoleText, "src").Bounds
	for _, kid := range [][3]string{{"a.go", "1 KB", "Go"}, {"b.go", "2 KB", "Go"}} {
		k := tcFind(t, tt, ui.RoleText, kid[0]).Bounds
		if k.Y <= src.Y || k.X < src.X+8 {
			t.Errorf("%q at %v, src at %v: want it below src, indented", kid[0], k, src)
		}
		// Both kids' Kind is Go: take the cells of the kid's own row.
		cells := tcRowCells(t, tt, kid[0])
		if len(cells) != 3 {
			t.Fatalf("the row of %q has %d cells, want 3", kid[0], len(cells))
		}
		for i, s := range kid[1:] {
			h := tcFind(t, tt, ui.RoleColumnHeader, []string{"Size", "Kind"}[i]).Bounds
			c := cells[i+1]
			b, ok := tcTextIn(tt, s, c)
			if !withinDIP(c.X, h.X, 1) || !ok {
				t.Errorf("the cell %v of %q's row, its header at %v: want it under the header, showing %q (%v)", c, kid[0], h, s, b)
			}
		}
	}
}

// Case 3: the Size text ends at its cell's end, less the cell's padding,
// ui's Space(2.5) of the light theme.
func TestTreeColumnsSizeIsEndAligned(t *testing.T) {
	tt := filesTester(filesTree())
	pad := ui.LightTheme().Space(2.5)
	b := tcFind(t, tt, ui.RoleText, "3 KB").Bounds
	c := tcCell(t, tt, "3 KB")
	if !withinDIP(b.X+b.W, c.X+c.W-pad, 1) {
		t.Errorf("3 KB ends at x %v, its cell at %v: want %v, the cell's end less %v", b.X+b.W, c.X+c.W, c.X+c.W-pad, pad)
	}
	k := tcFind(t, tt, ui.RoleText, "Text").Bounds
	if kc := tcCell(t, tt, "Text"); !withinDIP(k.X, kc.X+pad, 1) {
		t.Errorf("Kind's Text starts at x %v, its cell at %v: want it start-aligned", k.X, kc.X)
	}
}

// Case 4.
func TestTreeColumnsSelectionAndChecks(t *testing.T) {
	events := make(chan string, 8)
	tv := filesTree().CheckBoxes().
		OnSelect(func(k string, ok bool) { events <- fmt.Sprintf("select %s %v", k, ok) }).
		OnCheck(func(k string, on bool) { events <- fmt.Sprintf("check %s %v", k, on) })
	tt := filesTester(tv)
	readme := tcCell(t, tt, "README")
	tt.ClickAt(readme.X+readme.W-8, readme.Y+readme.H/2)
	tt.Frame()
	if got := soon(t, events); got != "select README true" {
		t.Fatalf("a click on README fired %q", got)
	}
	quiet(t, events, "one click selects once")
	box := tcFind(t, tt, ui.RoleCheckBox, "src").Bounds
	name := tcFind(t, tt, ui.RoleColumnHeader, "Name").Bounds
	if box.X < name.X || box.X+box.W > name.X+name.W {
		t.Fatalf("the check box of src at %v, the Name column at %v: want it inside", box, name)
	}
	tt.ClickAt(box.X+8, box.Y+box.H/2)
	tt.Frame()
	if got := soon(t, events); got != "check src true" {
		t.Fatalf("a click on src's box fired %q", got)
	}
	if !tv.IsChecked("src") {
		t.Fatal("src is not checked")
	}
}

// Case 5.
func TestTreeColumnsUpdateNodeShowsTheCell(t *testing.T) {
	tv := filesTree()
	tt := filesTester(tv)
	tv.UpdateNode("README", func(f *file) { f.Size = "4 KB" })
	tt.Frame()
	if !tcHas(tt, ui.RoleText, "4 KB") || tcHas(tt, ui.RoleText, "3 KB") {
		t.Fatalf("after UpdateNode: %q", tt.Texts())
	}
	h := tcFind(t, tt, ui.RoleColumnHeader, "Size").Bounds
	if c := tcCell(t, tt, "4 KB"); !withinDIP(c.X, h.X, 1) {
		t.Errorf("4 KB's cell at %v, Size's header at %v", c, h)
	}
}

// Case 6.
func TestTreeColumnsLoadingRowFillsOnlyTheNameColumn(t *testing.T) {
	tv := TreeView(func(f *file) string { return f.Path }, func(f *file) string { return f.Name }).
		SetRoots([]file{{"src", "src", "", "Folder"}}).
		SetColumns("Name", sizeCol(), kindCol()).
		SetHasChildren("src", true).Expand("src")
	tt := filesTester(tv)
	l := tcFind(t, tt, ui.RoleText, "Loading…").Bounds
	name := tcFind(t, tt, ui.RoleColumnHeader, "Name").Bounds
	if l.X < name.X || l.X+l.W > name.X+name.W {
		t.Fatalf("Loading… at %v, the Name column at %v", l, name)
	}
	cells := tcRowCells(t, tt, "Loading…")
	if len(cells) != 3 {
		t.Fatalf("the Loading row has %d cells, want 3", len(cells))
	}
	for _, a := range tt.AccessAll() {
		b := a.Bounds
		if a.Role != ui.RoleText || a.Name == "Loading…" {
			continue
		}
		if cy := b.Y + b.H/2; cy >= cells[1].Y && cy < cells[1].Y+cells[1].H && b.X >= cells[1].X {
			t.Errorf("the Loading row shows %q at %v outside the Name column", a.Name, b)
		}
	}
}

// Case 7.
func TestTreeColumnsHiddenColumnShowsNot(t *testing.T) {
	k := kindCol()
	k.Hidden = true
	tt := filesTester(plainFiles().SetColumns("Name", sizeCol(), k))
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name Size]" {
		t.Fatalf("headers %s, want [Name Size]", got)
	}
	if tcHas(tt, ui.RoleText, "Text") || tcHas(tt, ui.RoleText, "Folder") || !tcHas(tt, ui.RoleText, "3 KB") {
		t.Fatalf("shows %q: want no Kind cells", tt.Texts())
	}
	// Every column hidden: the tree column alone.
	s := sizeCol()
	s.Hidden = true
	tt = filesTester(plainFiles().SetColumns("Name", s, k))
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name]" {
		t.Fatalf("all hidden: headers %s, want [Name]", got)
	}
}

// tcTree is what a test compares of two element trees: their roles, names
// and bounds.
func tcTree(tt *ui.Tester) []string {
	var s []string
	for _, a := range tt.AccessAll() {
		s = append(s, fmt.Sprintf("%v %q %v", a.Role, a.Name, a.Bounds))
	}
	return s
}

// Case 8.
func TestTreeColumnsOffBuildsThePlainTree(t *testing.T) {
	tv := filesTree().Expand("src")
	tt := filesTester(tv)
	tv.SetColumns("")
	tt.Frame()
	if hs := tcHeaders(tt); len(hs) != 0 {
		t.Fatalf("SetColumns(\"\") shows the headers %q", hs)
	}
	if tv.inColumns != nil {
		t.Error("SetColumns(\"\") left the columns' hook set")
	}
	plain := filesTester(plainFiles().Expand("src"))
	got, want := tcTree(tt), tcTree(plain)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("after SetColumns(\"\"):\n%q\nnever in columns:\n%q", got, want)
	}
}

// Case 9.
func TestTreeColumnsSetAgainShowsTheNewColumns(t *testing.T) {
	tv := filesTree()
	tt := filesTester(tv)
	tv.SetColumns("File", Column[file]{Title: "Type", Text: func(f *file) string { return "type " + f.Kind }, Width: 120})
	tt.Frame()
	if got := fmt.Sprint(tcHeaders(tt)); got != "[File Type]" {
		t.Fatalf("headers %s, want [File Type]", got)
	}
	if !tcHas(tt, ui.RoleText, "type Text") || tcHas(tt, ui.RoleText, "3 KB") {
		t.Fatalf("shows %q: want the Type cells, no Size", tt.Texts())
	}
	h := tcFind(t, tt, ui.RoleColumnHeader, "Type").Bounds
	if c := tcCell(t, tt, "type Text"); !withinDIP(c.X, h.X, 1) || !withinDIP(c.W, h.W, 1) {
		t.Errorf("type Text's cell at %v, Type's header at %v", c, h)
	}
}

// A tree shown without columns takes them after its first frame.
func TestTreeColumnsSetAfterAPlainFrame(t *testing.T) {
	tv := plainFiles()
	tt := filesTester(tv)
	if hs := tcHeaders(tt); len(hs) != 0 {
		t.Fatalf("a plain tree shows the headers %q", hs)
	}
	tv.SetColumns("Name", sizeCol(), kindCol())
	tt.Frame()
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name Size Kind]" {
		t.Fatalf("headers %s, want [Name Size Kind]", got)
	}
	h := tcFind(t, tt, ui.RoleColumnHeader, "Size").Bounds
	if c := tcCell(t, tt, "3 KB"); !withinDIP(c.X, h.X, 1) {
		t.Errorf("3 KB's cell at %v, Size's header at %v", c, h)
	}
}

// Case 9, with the same headers: a column of the same title, width and
// alignment whose Text differs builds every row again (ui sees no change in
// its columns), so the cells show the new Text on the next frame.
func TestTreeColumnsSetAgainSameHeadersNewText(t *testing.T) {
	tv := filesTree()
	tt := filesTester(tv)
	rows := tv.state.Rows()
	version := func() []uint64 {
		v := make([]uint64, rows)
		for i := range v {
			v[i] = tv.state.RowVersion(tv.state.Item(i))
		}
		return v
	}
	before := version()
	kind := kindCol()
	kind.Text = func(f *file) string { return "kind " + f.Kind }
	tv.SetColumns("Name", sizeCol(), kind)
	tt.Frame()
	if got := changed(before, version()); len(got) != rows {
		t.Errorf("SetColumns: %d of %d rows changed their version, want all", len(got), rows)
	}
	if !tcHas(tt, ui.RoleText, "kind Text") || tcHas(tt, ui.RoleText, "Text") {
		t.Fatalf("shows %q: want the new Text of Kind", tt.Texts())
	}
}

// The Name cell spaces the arrow, check box, image and text as a row without
// columns does: ui's Space(1) between each.
func TestTreeColumnsNameCellSpacedAsWithoutColumns(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	il := ImageList(16)
	if err := il.Add("doc", pngOf(t, 16, blue)); err != nil {
		t.Fatal(err)
	}
	icon := func(*file) string { return "doc" }
	gap := ui.LightTheme().Space(1)
	// The image has no role: the box and the text, 16 + gap + 16 + gap apart.
	plain := filesTester(plainFiles().CheckBoxes().SetImageList(il, icon))
	pb, pt := tcFind(t, plain, ui.RoleCheckBox, "README").Bounds, tcFind(t, plain, ui.RoleText, "README").Bounds
	tt := filesTester(filesTree().CheckBoxes().SetImageList(il, icon))
	b, x := tcFind(t, tt, ui.RoleCheckBox, "README").Bounds, tcFind(t, tt, ui.RoleText, "README").Bounds
	if want := 32 + 2*gap; !withinDIP(x.X-b.X, want, 0.5) || !withinDIP(pt.X-pb.X, want, 0.5) {
		t.Errorf("the text starts %v after the box in columns, %v without: want %v", x.X-b.X, pt.X-pb.X, want)
	}
	// After the arrow, which starts at the cell's padding (Space(2.5)) and is
	// Space(4) wide, the gap.
	c := tcCell(t, tt, "README")
	th := ui.LightTheme()
	if want := c.X + th.Space(2.5) + th.Space(4) + gap; !withinDIP(b.X, want, 0.5) {
		t.Errorf("the check box starts at x %v, want %v: the arrow, then a gap of %v", b.X, want, gap)
	}
}

// Case 10: the keys are ui's outline's.
func TestTreeColumnsKeyboard(t *testing.T) {
	tv := filesTree()
	tt := filesTester(tv)
	src := tcCell(t, tt, "src")
	tt.ClickAt(src.X+src.W-8, src.Y+src.H/2)
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if !tv.IsExpanded("src") || !tcHas(tt, ui.RoleText, "a.go") {
		t.Fatalf("Right did not open src: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if k, _ := tv.Selected(); k != "src/a.go" {
		t.Fatalf("Right on an open src selected %q, want src/a.go", k)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if k, _ := tv.Selected(); k != "src" {
		t.Fatalf("Left on a.go selected %q, want src", k)
	}
}

// In columns too, typing finds a row by its text and the keys skip the
// Loading row (ui's ListState Label and Skip).
func TestTreeColumnsTypingAndTheLoadingRow(t *testing.T) {
	tv := plainFiles().SetColumns("Name", sizeCol()).SetHasChildren("README", true).Expand("README")
	tt := filesTester(tv)
	src := tcCell(t, tt, "src")
	tt.ClickAt(src.X+src.W-8, src.Y+src.H/2)
	tt.Frame()
	tt.Key(0, ui.KeyR)
	tt.Key(0, ui.KeyE)
	tt.Frame()
	if k, _ := tv.Selected(); k != "README" {
		t.Fatalf("typing re selected %q, want README", k)
	}
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	if k, ok := tv.Selected(); !ok || k != "README" {
		t.Fatalf("End selected %q, %v: want README, the Loading row below it skipped", k, ok)
	}
}

// Case 11: a frame at rest of the tree in columns allocates no more than one
// of a bare ui.OutlineTable of the same rows and columns.
func TestTreeColumnsFrameAllocatesAsABareOutlineTable(t *testing.T) {
	tv := filesTree().Expand("src")
	tt := filesTester(tv)
	for range 3 {
		tt.Frame()
	}
	tree := allocrun.Least(20, tt.Frame)

	type row struct{ name, size, kind string }
	data := map[string]row{"src": {"src", "", "Folder"}, "src/a.go": {"a.go", "1 KB", "Go"},
		"src/b.go": {"b.go", "2 KB", "Go"}, "README": {"README", "3 KB", "Text"}}
	cols := []ui.TableColumn{{Title: "Name", MinWidth: 120}, {Title: "Size", Width: 80, Align: ui.End}, {Title: "Kind", Width: 100}}
	roots, kids := []string{"src", "README"}, []string{"src/a.go", "src/b.go"}
	var s ui.OutlineState[string]
	s.Open.Add("src")
	s.RowVersion = func(string) uint64 { return 1 } // its rows keep, as the tree's do
	// Rows chosen as the tree's, none chosen: ui's list costs more with it.
	cursor := -1
	s.List.Selected = &cursor
	// Its functions are made once, as the tree's are: a frame makes none.
	var ctx *ui.Context
	children := func(k string) []string {
		if k == "src" {
			return kids
		}
		return nil
	}
	cell := func(k string, col int) {
		r := data[k]
		ui.Text(ctx, [...]string{r.name, r.size, r.kind}[col]).SingleLine()
	}
	// In the same view as the tree, whose own cost (content's) both pay.
	bare := ui.NewTester(content(Custom(func(c *ui.Context) ui.Element {
		ctx = c
		return ui.OutlineTable(c, &s, cols, roots, children, cell).Grow(1)
	})), 500, 300)
	for range 3 {
		bare.Frame()
	}
	ref := allocrun.Least(20, bare.Frame)
	t.Logf("a frame at rest: tree in columns %.0f allocs, bare OutlineTable %.0f", tree, ref)
	if tree > ref {
		t.Errorf("a frame at rest of the tree in columns allocates %.0f objects, a bare OutlineTable %.0f: want no more", tree, ref)
	}
}

// Case 12: SetColumns never called leaves the hook nil, so the tree builds
// as on main; the existing allocation tests of TreeView hold it.
func TestTreeColumnsHookNilWithoutSetColumns(t *testing.T) {
	if tv := plainFiles(); tv.inColumns != nil {
		t.Fatal("a TreeView never given columns has the columns' hook set")
	}
	tv := filesTree()
	if tv.inColumns == nil {
		t.Fatal("SetColumns did not set the hook")
	}
	if tv.SetColumns(""); tv.inColumns != nil {
		t.Fatal("SetColumns(\"\") left the hook set")
	}
}

// Case 13: SetColumns, UpdateNode and Expand from another goroutine while
// frames run (run it with -race) show on a following frame.
func TestTreeColumnsFromAnotherGoroutine(t *testing.T) {
	tv := filesTree()
	tt := filesTester(tv)
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		defer close(done)
		for i := range 40 {
			if i%2 == 0 {
				tv.SetColumns("Name", kindCol())
			} else {
				tv.SetColumns("Name", sizeCol(), kindCol())
			}
			tv.UpdateNode("README", func(f *file) { f.Size = strconv.Itoa(i) + " KB" })
			tv.Expand("src")
		}
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	wg.Wait()
	tt.Frame()
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name Size Kind]" {
		t.Errorf("headers %s, want the last SetColumns'", got)
	}
	if !tcHas(tt, ui.RoleText, "39 KB") || !tcHas(tt, ui.RoleText, "a.go") {
		t.Errorf("shows %q: want 39 KB and src open", tt.Texts())
	}
}

// Review Focus 1: at a text scale of 1.5 in a tree 300 DIPs wide, a deep
// node's long name clips in the Name column, which keeps its MinWidth, and
// never inks the Size cell.
func TestTreeColumnsLongNameClipsInTheNameColumn(t *testing.T) {
	tv := plainFiles().SetColumns("Name", sizeCol(), kindCol())
	tv.SetChildren("src/a.go", []file{{"deep", "a-very-long-folder-name-that-overflows", "", "Folder"}})
	tv.Expand("src", "src/a.go")
	tt := ui.NewTester(content(tv), 300, 300)
	tt.SetPreferences(ui.Preferences{TextScale: 1.5})
	tt.Frame()
	tt.Frame()
	name := tcFind(t, tt, ui.RoleColumnHeader, "Name").Bounds
	if name.W < 120 {
		t.Errorf("the Name column is %v wide, want its MinWidth of 120", name.W)
	}
	// The scale applies: a line is 15.3 DIPs tall at 1.
	if h := tcFind(t, tt, ui.RoleText, "README").Bounds.H; h < 15.3*1.4 {
		t.Fatalf("README is %v tall: the text scale of 1.5 did not apply", h)
	}
	cells := tcRowCells(t, tt, "a-very-long-folder-name-that-overflows")
	if len(cells) < 2 {
		t.Fatalf("the deep row has %d cells", len(cells))
	}
	size := cells[1]
	img := tt.Image()
	sx := float32(img.Bounds().Dx()) / 300
	bg := img.RGBAAt(int((size.X+size.W/2)*sx), int((size.Y+size.H/2)*sx))
	for y := int((size.Y + 2) * sx); y < int((size.Y+size.H-2)*sx); y++ {
		for x := int((size.X + 2) * sx); x < int((size.X+size.W-2)*sx); x++ {
			if img.RGBAAt(x, y) != bg {
				t.Fatalf("the deep row's Size cell %v has ink at (%d, %d): the name spills over it", size, x, y)
			}
		}
	}
}

// Review Focus 2: dragging the Size header's edge resizes the column, and
// the cells follow the header.
func TestTreeColumnsResizeByDragging(t *testing.T) {
	tt := filesTester(filesTree())
	h := tcFind(t, tt, ui.RoleColumnHeader, "Size").Bounds
	x, y := h.X+h.W-1, h.Y+h.H/2
	tt.Press(x, y)
	tt.Move(x+20, y)
	tt.Move(x+40, y)
	tt.Release(x+40, y)
	tt.Frame()
	h2 := tcFind(t, tt, ui.RoleColumnHeader, "Size").Bounds
	if withinDIP(h2.W, h.W, 10) {
		t.Fatalf("Size's header is %v wide after the drag, %v before: want it wider", h2.W, h.W)
	}
	c := tcCell(t, tt, "3 KB")
	if !withinDIP(c.X, h2.X, 1) || !withinDIP(c.W, h2.W, 1) {
		t.Errorf("3 KB's cell at %v, Size's header at %v: want the cell under it", c, h2)
	}
	k := tcFind(t, tt, ui.RoleColumnHeader, "Kind").Bounds
	if kc := tcCell(t, tt, "Text"); !withinDIP(kc.X, k.X, 1) {
		t.Errorf("Text's cell at %v, Kind's header at %v", kc, k)
	}
}

// Review Focus 3: with 2,000 nodes open, only the rows in view build, and
// scrolling keeps the cells under their headers.
func TestTreeColumnsManyOpenNodesBuildTheRowsInView(t *testing.T) {
	tv := TreeView(func(f *file) string { return f.Path }, func(f *file) string { return f.Name }).
		SetColumns("Name", sizeCol(), kindCol())
	roots := make([]file, 20)
	for i := range roots {
		roots[i] = file{"r" + strconv.Itoa(i), "Root " + strconv.Itoa(i), "", "Folder"}
	}
	tv.SetRoots(roots)
	for i := range roots {
		kids := make([]file, 100)
		for j := range kids {
			n := strconv.Itoa(i*100 + j)
			kids[j] = file{"k" + n, "Node " + n, n + " B", "Data"}
		}
		tv.SetChildren(roots[i].Path, kids)
	}
	tv.ExpandAll()
	tt := filesTester(tv)
	if n := tv.state.Rows(); n != 2020 {
		t.Fatalf("%d rows, want 2020", n)
	}
	items := 0
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleTreeItem {
			items++
		}
	}
	if items == 0 || items > 60 {
		t.Fatalf("%d rows built of 2020: want those in view only", items)
	}
	tt.Scroll(250, 150, 0, 3000)
	tt.Frame()
	tt.Frame()
	if tcHas(tt, ui.RoleText, "Root 0") {
		t.Fatal("the scroll did not move the rows")
	}
	h := tcFind(t, tt, ui.RoleColumnHeader, "Kind").Bounds
	seen := 0
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleText && a.Name == "Data" && a.Bounds.H > 0 {
			seen++
			if c := tcCellAt(tt, a.Bounds); !withinDIP(c.X, h.X, 1) {
				t.Errorf("a Kind cell at %v after the scroll, its header at %v", c, h)
			}
		}
	}
	if seen == 0 {
		t.Fatalf("no Kind cell in view after the scroll: %q", tt.Texts())
	}
}

// tcTextIn returns the bounds of a text s whose center is inside r.
func tcTextIn(tt *ui.Tester, s string, r ui.Rect) (ui.Rect, bool) {
	for _, a := range tt.AccessAll() {
		b := a.Bounds
		if cx, cy := b.X+b.W/2, b.Y+b.H/2; a.Role == ui.RoleText && a.Name == s && cx >= r.X && cx < r.X+r.W && cy >= r.Y && cy < r.Y+r.H {
			return b, true
		}
	}
	return ui.Rect{}, false
}

// tcCellAt returns the cell holding the center of b.
func tcCellAt(tt *ui.Tester, b ui.Rect) ui.Rect {
	cx, cy := b.X+b.W/2, b.Y+b.H/2
	for _, a := range tt.AccessAll() {
		if r := a.Bounds; a.Role == ui.RoleCell && cx >= r.X && cx < r.X+r.W && cy >= r.Y && cy < r.Y+r.H {
			return r
		}
	}
	return ui.Rect{}
}

// tcInk returns how many pixels across the ink inside r spans: the pixels
// that are not bg, 2 pixels in from its edges.
func tcInk(tt *ui.Tester, r ui.Rect, bg color.RGBA) int {
	img := tt.Image()
	lo, hi := -1, -1
	for y := int(r.Y) + 2; y < int(r.Y+r.H)-2; y++ {
		for x := int(r.X) + 2; x < int(r.X+r.W)-2; x++ {
			if img.RGBAAt(x, y) != bg {
				if lo < 0 || x < lo {
					lo = x
				}
				hi = max(hi, x)
			}
		}
	}
	if lo < 0 {
		return 0
	}
	return hi - lo + 1
}

// tcContrast returns the highest contrast of a pixel inside r against bg.
func tcContrast(tt *ui.Tester, r ui.Rect, bg color.RGBA) float64 {
	img := tt.Image()
	best := 1.0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := img.RGBAAt(x, y)
			best = math.Max(best, ratio(ui.RGB(p.R, p.G, p.B), ui.RGB(bg.R, bg.G, bg.B)))
		}
	}
	return best
}

// Review Focus 4: under each built-in skin, light and dark, the header takes
// the grid header's look, as DataGridView's does, and the rows' text reaches
// 4.5:1 on their background.
func TestTreeColumnsHeaderLookAndContrastUnderTheSkins(t *testing.T) {
	for _, skin := range []string{"", "Calm", "Classic", "HighContrast"} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dark=%v", skin, dark), func(t *testing.T) {
				if skin != "" {
					useSkinNamed(t, skin, false)
				} else {
					calmOff()
				}
				tt := filesTester(filesTree())
				tt.SetDark(dark)
				tt.Advance(time.Second)
				tt.Frame()
				grid := filesTester(DataGridView(func(f *file) string { return f.Path },
					Column[file]{Title: "Name", Text: func(f *file) string { return f.Name }, MinWidth: 120}, sizeCol(), kindCol()).
					SetRows([]file{{"README", "README", "3 KB", "Text"}}))
				grid.SetDark(dark)
				grid.Advance(time.Second)
				grid.Frame()
				h := tcFind(t, tt, ui.RoleColumnHeader, "Kind").Bounds
				g := tcFind(t, grid, ui.RoleColumnHeader, "Kind").Bounds
				px, gx := tt.Image().RGBAAt(int(h.X+h.W-12), int(h.Y+h.H/2)), grid.Image().RGBAAt(int(g.X+g.W-12), int(g.Y+g.H/2))
				if px != gx {
					t.Errorf("the tree's header fills with %v, a DataGridView's with %v", px, gx)
				}
				// The header's text takes the skin's size, as the grid's.
				if hw, gw := tcInk(tt, h, px), tcInk(grid, g, gx); hw == 0 || hw != gw {
					t.Errorf("the tree's header text inks %d pixels across, the grid's %d", hw, gw)
				}
				c := tcCell(t, tt, "README")
				bg := tt.Image().RGBAAt(int(c.X+c.W-4), int(c.Y+c.H/2))
				for _, s := range []string{"README", "3 KB", "Text"} {
					if got := tcContrast(tt, tcFind(t, tt, ui.RoleText, s).Bounds, bg); got < 4.5 {
						t.Errorf("%q reaches %.2f:1 on its row, want 4.5:1", s, got)
					}
				}
			})
		}
	}
}

// Review Focus 4, a skin's own GridHeader look: the tree's header takes it.
func TestTreeColumnsHeaderTakesTheGridHeaderLook(t *testing.T) {
	useLooks(t, map[string]string{"GridHeader": "bg-rose-600"})
	tt := filesTester(filesTree())
	h := tcFind(t, tt, ui.RoleColumnHeader, "Kind").Bounds
	rose600 := color.RGBA{0xe1, 0x1d, 0x48, 0xff}
	if got := tt.Image().RGBAAt(int(h.X+h.W-12), int(h.Y+h.H/2)); got != rose600 {
		t.Errorf("the header fills with %v, want GridHeader's rose-600 %v", got, rose600)
	}
}

// Review Focus 5: an ImageList's images show in the Name column at the
// list's size, as without columns.
func TestTreeColumnsImageListInTheNameColumn(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	il := ImageList(24)
	if err := il.Add("doc", pngOf(t, 48, blue)); err != nil {
		t.Fatal(err)
	}
	tv := filesTree().SetImageList(il, func(*file) string { return "doc" })
	tt := filesTester(tv)
	name := tcCell(t, tt, "README")
	if n := bluePixels2(tt, name); n < 24*24 || n > 24*24+4*24 {
		t.Errorf("README's Name cell has %d blue pixels, want about %d", n, 24*24)
	}
	for _, s := range []string{"3 KB", "Text"} {
		if n := bluePixels2(tt, tcCell(t, tt, s)); n != 0 {
			t.Errorf("the cell of %q has %d blue pixels, want none", s, n)
		}
	}
}

// No rows: the header shows alone.
func TestTreeColumnsEmptyTreeShowsTheHeader(t *testing.T) {
	tv := TreeView(func(f *file) string { return f.Path }, func(f *file) string { return f.Name }).
		SetColumns("Name", sizeCol())
	tt := filesTester(tv)
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name Size]" {
		t.Fatalf("headers %s, want [Name Size]", got)
	}
	if tv.state.Rows() != 0 {
		t.Fatalf("%d rows, want none", tv.state.Rows())
	}
}

// One column: SetColumns with a title and no columns shows the tree column
// alone, under its header; SetColumns("", cols...) shows the columns, the
// tree column untitled.
func TestTreeColumnsTreeColumnAloneAndUntitled(t *testing.T) {
	tt := filesTester(plainFiles().SetColumns("Name"))
	if got := fmt.Sprint(tcHeaders(tt)); got != "[Name]" {
		t.Fatalf("headers %s, want [Name]", got)
	}
	tt = filesTester(plainFiles().SetColumns("", sizeCol()))
	if got := fmt.Sprint(tcHeaders(tt)); got != "[ Size]" {
		t.Fatalf("headers %s, want an untitled tree column and Size", got)
	}
	if !tcHas(tt, ui.RoleText, "3 KB") {
		t.Fatalf("shows %q: want the Size cells", tt.Texts())
	}
}

// ui builds the rows a taller window shows late, while it lays the frame
// out, without the build's lock: a cell takes it, so Text still runs under
// it and a goroutine changing nodes meanwhile does not race the cells.
func TestTreeColumnsCellsBuiltWhileLayingOutHoldTheLock(t *testing.T) {
	var unlocked atomicBool
	tv := TreeView(func(f *file) string { return f.Path }, func(f *file) string { return f.Name }).
		SetColumns("Name", Column[file]{Title: "Size", Text: func(f *file) string {
			if !lockHeld() {
				unlocked.set()
			}
			return f.Size
		}})
	nodes := make([]file, 2000)
	for i := range nodes {
		n := strconv.Itoa(i)
		nodes[i] = file{n, "Node " + n, n + " B", ""}
	}
	tv.SetRoots(nodes)
	tt := filesTester(tv)
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			tv.UpdateNode(strconv.Itoa(i%2000), func(f *file) { f.Size = strconv.Itoa(i) })
		}
	}()
	for i := range 40 {
		tt.SetSize(500, 300+i%2*2000)
		tt.Frame()
	}
	close(stop)
	<-done
	if unlocked.get() {
		t.Fatal("a cell built while laying out ran Text without the package's lock")
	}
}

// atomicBool is a flag set from the main thread and read by the test.
type atomicBool struct {
	mu sync.Mutex
	v  bool
}

func (b *atomicBool) set()      { b.mu.Lock(); b.v = true; b.mu.Unlock() }
func (b *atomicBool) get() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.v }

// A column's Text runs under the package's lock, as the docs say.
func TestTreeColumnsTextRunsUnderTheLock(t *testing.T) {
	calls, locked := 0, true
	tv := plainFiles().SetColumns("Name", Column[file]{Title: "Size", Text: func(f *file) string {
		calls++
		locked = locked && lockHeld()
		return f.Size
	}})
	filesTester(tv)
	if calls == 0 || !locked {
		t.Fatalf("Text ran %d times, always under the lock: %v", calls, locked)
	}
}
