package widget

import (
	"fmt"
	"image"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// galleryKit is the ribbon of the gallery's cases: a Home tab whose Styles
// group holds a gallery named "Gallery" of n items, "Style 0" to
// "Style n-1", 4 columns wide, over a text box, in a form 800 × 400 with
// access keys on. picks and hovers receive what OnSelect and OnHover give.
type galleryKit struct {
	r             *RibbonWidget
	g             *RibbonGalleryWidget
	tt            *ui.Tester
	picks, hovers chan int
}

func galleryItems(n int) []GalleryItem {
	items := make([]GalleryItem, n)
	for i := range items {
		items[i] = GalleryItem{Text: fmt.Sprintf("Style %d", i)}
	}
	return items
}

func newGalleryKit(t *testing.T, n int) *galleryKit {
	t.Helper()
	withMac(t, false)
	k := &galleryKit{picks: make(chan int, 32), hovers: make(chan int, 32)}
	k.r = Ribbon().AddTab("Home").AddGroup("Styles")
	k.g = k.r.AddGallery(galleryItems(n), 4)
	k.g.SetAccessibleName("Gallery")
	k.g.OnSelect(func(i int) { k.picks <- i })
	k.g.OnHover(func(i int) { k.hovers <- i })
	form := NewForm("Gallery", 800, 400).SetContent(VBox(0).Add(k.r, TextBox().SetAccessibleName("Body")))
	k.tt = ui.NewTester(form.view, 800, 400)
	k.tt.Frame()
	return k
}

// gotPosted returns what ch received so far, once the handlers posted have run.
func gotPosted(ch chan int) []int {
	Update(func() {})
	var got []int
	for {
		select {
		case i := <-ch:
			got = append(got, i)
		default:
			return got
		}
	}
}

// stripBottom is the bottom edge of the gallery in the ribbon.
func (k *galleryKit) stripBottom(t *testing.T) float32 {
	t.Helper()
	g, ok := k.tt.AccessByRole(ui.RoleGroup, "Gallery")
	if !ok {
		t.Fatalf("no gallery: %q", k.tt.Texts())
	}
	return g.Bounds.Y + g.Bounds.H
}

// items returns the items of the gallery shown in the ribbon (in the strip)
// or under it (in the drop-down), by name.
func (k *galleryKit) items(t *testing.T, below bool) map[string]ui.AccessInfo {
	t.Helper()
	bottom := k.stripBottom(t)
	m := map[string]ui.AccessInfo{}
	for _, a := range k.tt.AccessAll() {
		var i int
		if _, err := fmt.Sscanf(a.Name, "Style %d", &i); err != nil || a.Bounds.W == 0 {
			continue
		}
		if (a.Bounds.Y >= bottom-0.5) == below {
			m[a.Name] = a
		}
	}
	return m
}

// wantStrip checks that the strip shows the items from to to-1 alone.
func (k *galleryKit) wantStrip(t *testing.T, from, to int) {
	t.Helper()
	got := k.items(t, false)
	want := map[string]bool{}
	for i := from; i < to; i++ {
		want[fmt.Sprintf("Style %d", i)] = true
	}
	if len(got) != len(want) {
		t.Fatalf("the strip shows %d items %v, want Style %d to %d", len(got), slices.Sorted(maps.Keys(got)), from, to-1)
	}
	for name := range want {
		if _, ok := got[name]; !ok {
			t.Fatalf("the strip does not show %q: %v", name, slices.Sorted(maps.Keys(got)))
		}
	}
}

// button returns the gallery's button named name.
func (k *galleryKit) button(t *testing.T, name string) ui.AccessInfo {
	t.Helper()
	b, ok := k.tt.AccessByRole(ui.RoleButton, name)
	if !ok {
		t.Fatalf("no button %q: %q", name, k.tt.Texts())
	}
	return b
}

func (k *galleryKit) click(t *testing.T, a ui.AccessInfo) {
	t.Helper()
	k.tt.ClickAt(centre(a.Bounds))
	k.tt.Frame()
}

// openDropDown opens the drop-down with a click on its button.
func (k *galleryKit) openDropDown(t *testing.T) {
	t.Helper()
	k.click(t, k.button(t, "More"))
	if n := len(k.items(t, true)); n == 0 {
		t.Fatalf("a click on More opens no drop-down: %q", k.tt.Texts())
	}
}

// 1. 10 items with 4 columns show items 0–3. The down arrow shows 4–7, and
// the up arrow at row 0 is disabled.
func TestRibbonGalleryRow(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.wantStrip(t, 0, 4)
	if !k.button(t, "Previous").Disabled {
		t.Fatal("the up arrow is enabled at row 0")
	}
	if k.button(t, "Next").Disabled {
		t.Fatal("the down arrow is disabled at row 0 of 3")
	}
	k.click(t, k.button(t, "Next"))
	k.wantStrip(t, 4, 8)
	if k.button(t, "Previous").Disabled {
		t.Fatal("the up arrow is disabled at row 1")
	}
	// Own case: the last row holds what is left, and the down arrow is
	// disabled there.
	k.click(t, k.button(t, "Next"))
	k.wantStrip(t, 8, 10)
	if !k.button(t, "Next").Disabled {
		t.Fatal("the down arrow is enabled at the last row")
	}
	k.click(t, k.button(t, "Next")) // disabled: nothing
	k.wantStrip(t, 8, 10)
	k.click(t, k.button(t, "Previous"))
	k.wantStrip(t, 4, 8)
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("scrolling ran OnSelect %v", got)
	}
}

// Own case: one row of items shows no row to scroll to, and an item takes
// the ribbon's whole group height.
func TestRibbonGalleryOneRow(t *testing.T) {
	k := newGalleryKit(t, 3)
	k.wantStrip(t, 0, 3)
	if !k.button(t, "Previous").Disabled || !k.button(t, "Next").Disabled {
		t.Fatal("an arrow is enabled with one row of items")
	}
	g, _ := k.tt.AccessByRole(ui.RoleGroup, "Gallery")
	item := k.items(t, false)["Style 0"].Bounds
	if item.H < g.Bounds.H/2 {
		t.Fatalf("an item is %v high in a gallery %v high: not one row high", item.H, g.Bounds.H)
	}
}

// 2. The drop-down button opens a grid of all 10, with the selected one
// marked. A click picks it, fires OnSelect once and closes the grid.
func TestRibbonGalleryDropDown(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.g.SetSelected(6)
	k.tt.Frame()
	if k.button(t, "More").Expanded {
		t.Fatal("More reads expanded with the drop-down closed")
	}
	k.openDropDown(t)
	if !k.button(t, "More").Expanded {
		t.Fatal("More does not read expanded with the drop-down open")
	}
	grid := k.items(t, true)
	if len(grid) != 10 {
		t.Fatalf("the drop-down holds %d items, want 10: %v", len(grid), slices.Sorted(maps.Keys(grid)))
	}
	for name, a := range grid {
		if a.Checked != (name == "Style 6") {
			t.Fatalf("%q in the drop-down: Checked %v, want only Style 6 marked", name, a.Checked)
		}
	}
	// In 4 columns: Style 4 starts the second row, under Style 0.
	s0, s4, s5 := grid["Style 0"].Bounds, grid["Style 4"].Bounds, grid["Style 5"].Bounds
	if !near1(s0.X, s4.X) || s4.Y < s0.Y+s0.H-0.5 || !near1(s4.Y, s5.Y) || s5.X <= s4.X {
		t.Fatalf("Style 0 %v, Style 4 %v, Style 5 %v: not a grid of 4 columns", s0, s4, s5)
	}
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("SetSelected and opening ran OnSelect %v", got)
	}
	k.click(t, grid["Style 2"])
	if got := gotPosted(k.picks); !slices.Equal(got, []int{2}) {
		t.Fatalf("a click on Style 2 ran OnSelect %v, want [2]", got)
	}
	if k.g.Selected() != 2 {
		t.Fatalf("Selected() = %d after the click on Style 2", k.g.Selected())
	}
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("the drop-down still shows %d items after the pick", n)
	}
	// The strip shows the row of the pick, marked.
	k.wantStrip(t, 0, 4)
	if !k.items(t, false)["Style 2"].Checked {
		t.Fatal("Style 2 is not marked in the strip")
	}
	// Own case: a click on More toggles: it closes the drop-down open.
	k.openDropDown(t)
	k.click(t, k.button(t, "More"))
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("a second click on More leaves the drop-down open (%d items)", n)
	}
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("closing ran OnSelect %v", got)
	}
}

// Own case: a click on an item in the strip picks it too.
func TestRibbonGalleryStripClickPicks(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.click(t, k.items(t, false)["Style 1"])
	if got := gotPosted(k.picks); !slices.Equal(got, []int{1}) || k.g.Selected() != 1 {
		t.Fatalf("a click on Style 1 in the strip: OnSelect %v, Selected %d", got, k.g.Selected())
	}
}

// 3. Moving over items 2 then 3 posts OnHover(2) then OnHover(3), and
// leaving posts OnHover(-1). A keyboard move posts the same.
func TestRibbonGalleryHover(t *testing.T) {
	k := newGalleryKit(t, 10)
	strip := k.items(t, false)
	k.tt.Move(centre(strip["Style 2"].Bounds))
	k.tt.Frame()
	k.tt.Move(centre(strip["Style 3"].Bounds))
	k.tt.Frame()
	body, _ := k.tt.Access("Body")
	k.tt.Move(centre(body.Bounds))
	k.tt.Frame()
	if got := gotPosted(k.hovers); !slices.Equal(got, []int{2, 3, -1}) {
		t.Fatalf("the pointer over Style 2, Style 3, then off: OnHover %v, want [2 3 -1]", got)
	}
	k.g.SetSelected(1)
	k.tt.Frame()
	k.openDropDown(t)
	if got := gotPosted(k.hovers); len(got) != 0 {
		t.Fatalf("opening the drop-down posted OnHover %v", got)
	}
	typeTip(k.tt, ui.KeyRight)
	typeTip(k.tt, ui.KeyRight)
	typeTip(k.tt, ui.KeyEscape)
	if got := gotPosted(k.hovers); !slices.Equal(got, []int{2, 3, -1}) {
		t.Fatalf("Right, Right, Escape from Style 1: OnHover %v, want [2 3 -1]", got)
	}
	// Own case: the pointer in the drop-down posts as in the strip.
	k.openDropDown(t)
	k.tt.Move(centre(k.items(t, true)["Style 9"].Bounds))
	k.tt.Frame()
	k.tt.Move(790, 395) // off the items and the drop-down, the drop-down open
	k.tt.Frame()
	if got := gotPosted(k.hovers); !slices.Equal(got, []int{9, -1}) {
		t.Fatalf("the pointer over Style 9 in the drop-down, then off: OnHover %v, want [9 -1]", got)
	}
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("hovering ran OnSelect %v", got)
	}
}

// 4. Arrows move in the grid, Enter picks, Escape closes with no pick. The
// KeyTip opens it.
func TestRibbonGalleryKeys(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.g.SetSelected(0)
	typeTip(k.tt, ui.KeyF10)
	typeTip(k.tt, ui.KeyH)
	wantTips(t, k.tt, map[string]string{"Gallery": "G"})
	typeTip(k.tt, ui.KeyG)
	k.tt.Frame()
	if n := len(k.items(t, true)); n != 10 {
		t.Fatalf("the KeyTip G opens a drop-down of %d items, want 10: %q", n, k.tt.Texts())
	}
	wantTips(t, k.tt, map[string]string{})
	typeTip(k.tt, ui.KeyRight) // 1
	typeTip(k.tt, ui.KeyDown)  // 5
	typeTip(k.tt, ui.KeyRight) // 6
	typeTip(k.tt, ui.KeyLeft)  // 5
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{5}) || k.g.Selected() != 5 {
		t.Fatalf("Right, Down, Right, Left, Enter from Style 0: OnSelect %v, Selected %d, want 5", got, k.g.Selected())
	}
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("Enter leaves the drop-down open (%d items)", n)
	}
	k.wantStrip(t, 4, 8) // the row of the pick
	k.openDropDown(t)
	typeTip(k.tt, ui.KeyUp) // 1
	typeTip(k.tt, ui.KeyEscape)
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("Escape leaves the drop-down open (%d items)", n)
	}
	if got := gotPosted(k.picks); len(got) != 0 || k.g.Selected() != 5 {
		t.Fatalf("Escape: OnSelect %v, Selected %d, want none and 5", got, k.g.Selected())
	}
}

// Own case: the arrows stop at the grid's edges: Down from the last row,
// Right from the last item, Up from the first row and Left from the first
// item move nowhere.
func TestRibbonGalleryKeysStopAtTheEdges(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.g.SetSelected(9)
	k.tt.Frame()
	k.openDropDown(t)
	typeTip(k.tt, ui.KeyDown)
	typeTip(k.tt, ui.KeyRight)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{9}) {
		t.Fatalf("Down, Right, Enter from Style 9: OnSelect %v, want [9]", got)
	}
	k.g.SetSelected(1)
	k.tt.Frame()
	k.openDropDown(t)
	typeTip(k.tt, ui.KeyUp)
	typeTip(k.tt, ui.KeyLeft) // 0
	typeTip(k.tt, ui.KeyLeft)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{0}) {
		t.Fatalf("Up, Left, Left, Enter from Style 1: OnSelect %v, want [0]", got)
	}
	// With nothing selected, Enter picks nothing, and the first arrow goes
	// to the first item.
	k.g.SetSelected(-1)
	k.tt.Frame()
	k.openDropDown(t)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); len(got) != 0 || len(k.items(t, true)) != 10 {
		t.Fatalf("Enter with none selected: OnSelect %v, %d items in the drop-down, want none and 10", got, len(k.items(t, true)))
	}
	typeTip(k.tt, ui.KeyDown)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{0}) {
		t.Fatalf("Down, Enter with none selected: OnSelect %v, want [0]", got)
	}
}

// 5. SetItems with fewer items clamps the selection to −1 when it is gone.
func TestRibbonGallerySetItems(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.g.SetSelected(7)
	k.g.SetItems(galleryItems(5))
	k.tt.Frame()
	if s := k.g.Selected(); s != -1 {
		t.Fatalf("Selected() = %d after SetItems of 5 items with 7 selected, want -1", s)
	}
	// The first item gone is no longer selected; the last item kept stays.
	k.g.SetItems(galleryItems(10)).SetSelected(5)
	k.g.SetItems(galleryItems(5))
	if s := k.g.Selected(); s != -1 {
		t.Fatalf("Selected() = %d after SetItems of 5 items with 5 selected, want -1", s)
	}
	k.g.SetSelected(4)
	k.g.SetItems(galleryItems(5))
	if s := k.g.Selected(); s != 4 {
		t.Fatalf("Selected() = %d after SetItems of 5 items with 4 selected, want 4", s)
	}
	// The row shown is clamped too: row 1 of 5 items, then 2 items.
	k.tt.Frame()
	k.wantStrip(t, 4, 5)
	k.g.SetItems(galleryItems(2))
	k.tt.Frame()
	k.wantStrip(t, 0, 2)
	if k.g.Selected() != -1 {
		t.Fatalf("Selected() = %d with 2 items", k.g.Selected())
	}
	k.g.SetItems(nil)
	k.tt.Frame()
	k.wantStrip(t, 0, 0)
	if !k.button(t, "Next").Disabled || !k.button(t, "Previous").Disabled {
		t.Fatal("an arrow is enabled with no items")
	}
	// Own case: SetSelected out of range selects none.
	k.g.SetItems(galleryItems(3)).SetSelected(3)
	if s := k.g.Selected(); s != -1 {
		t.Fatalf("SetSelected(3) of 3 items: Selected() = %d, want -1", s)
	}
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("the setters ran OnSelect %v", got)
	}
}

// 6. In RTL the items run from the right, the arrows at the left end, and
// Right moves back in the grid.
func TestRibbonGalleryRTL(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.tt.SetDirection(ui.RightToLeft)
	k.tt.Frame()
	strip := k.items(t, false)
	s0, s1, s3 := strip["Style 0"].Bounds, strip["Style 1"].Bounds, strip["Style 3"].Bounds
	if s0.X <= s1.X {
		t.Fatalf("in RTL Style 0 at %v is not right of Style 1 at %v", s0, s1)
	}
	if more := k.button(t, "More").Bounds; more.X+more.W > s3.X+0.5 {
		t.Fatalf("in RTL More at %v is not left of the last item %v", more, s3)
	}
	k.g.SetSelected(1)
	k.tt.Frame()
	k.openDropDown(t)
	grid := k.items(t, true)
	if g0, g1 := grid["Style 0"].Bounds, grid["Style 1"].Bounds; g0.X <= g1.X {
		t.Fatalf("in RTL the drop-down's Style 0 at %v is not right of Style 1 at %v", g0, g1)
	}
	typeTip(k.tt, ui.KeyRight)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{0}) {
		t.Fatalf("in RTL Right, Enter from Style 1: OnSelect %v, want [0]", got)
	}
	k.openDropDown(t)
	typeTip(k.tt, ui.KeyLeft)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{1}) {
		t.Fatalf("in RTL Left, Enter from Style 0: OnSelect %v, want [1]", got)
	}
}

// 7. The setters from another goroutine while frames run show on a
// following frame (run it under -race).
func TestRibbonGalleryFromGoroutine(t *testing.T) {
	k := newGalleryKit(t, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		k.g.SetItems(galleryItems(6))
		k.g.SetSelected(5)
		k.g.OnSelect(func(i int) { k.picks <- 100 + i })
		k.g.OnHover(func(i int) {})
		_ = k.g.Selected()
	}()
	for range 20 {
		k.tt.Frame()
	}
	wg.Wait()
	k.tt.Frame()
	k.wantStrip(t, 4, 6)
	if !k.items(t, false)["Style 5"].Checked || k.g.Selected() != 5 {
		t.Fatalf("Style 5 is not selected: Selected %d", k.g.Selected())
	}
	k.click(t, k.items(t, false)["Style 4"])
	if got := gotPosted(k.picks); !slices.Equal(got, []int{104}) {
		t.Fatalf("a click after OnSelect was set from another goroutine: %v, want [104]", got)
	}
}

// 8. A ribbon's frame at rest with a gallery allocates no more than the same
// ribbon with a group of as many tool buttons in its place.
func TestRibbonGalleryAllocNothingAtRest(t *testing.T) {
	withMac(t, false)
	g := Ribbon().AddTab("Home").AddGroup("Styles")
	g.AddGallery(galleryItems(10), 4)
	with := ui.NewTester(content(VBox(0).Add(g, Label("body"))), 800, 400)
	plain := Ribbon().AddTab("Home").AddGroup("Styles").
		AddLarge(ToolButton("A"), ToolButton("B"), ToolButton("C"), ToolButton("D")).
		AddSmall(ToolButton("Previous"), ToolButton("Next"), ToolButton("More"))
	without := ui.NewTester(content(VBox(0).Add(plain, Label("body"))), 800, 400)
	for range 10 {
		with.Frame()
		without.Frame()
	}
	got, want := allocrun.Least(30, with.Frame), allocrun.Least(30, without.Frame)
	t.Logf("allocations in a frame: the gallery's ribbon %v, the tool buttons' %v", got, want)
	if got > want {
		t.Fatalf("a gallery's ribbon allocates %v in a frame at rest, the tool buttons' %v", got, want)
	}
}

// Off means unchanged: a ribbon with no gallery installs no gallery hook,
// and its tips act as before.
func TestRibbonGalleryOffIsUnchanged(t *testing.T) {
	k, tt := tipKit(t)
	typeTip(tt, ui.KeyF10)
	typeTip(tt, ui.KeyH)
	wantTips(t, tt, homeTips)
	typeTip(tt, ui.KeyP)
	received(t, k.pasted)
	mu.Lock()
	hook := k.r.galleryTip
	mu.Unlock()
	if hook != nil {
		t.Fatal("a ribbon with no gallery has the gallery's tip hook")
	}
}

// Own case: a disabled gallery's KeyTip opens nothing and gives the focus
// back, as a disabled command's; its items and buttons do nothing.
func TestRibbonGalleryDisabled(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.g.SetDisabled(true)
	k.tt.Frame()
	body, _ := k.tt.Access("Body")
	k.click(t, body)
	if !k.tt.Focused("Body") {
		t.Fatal("the text box did not take the focus")
	}
	typeTip(k.tt, ui.KeyF10)
	typeTip(k.tt, ui.KeyH)
	holdEvents(t)
	typeTip(k.tt, ui.KeyG)
	settleTips(t, k.tt)
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("the KeyTip opens a disabled gallery's drop-down (%d items)", n)
	}
	if !k.tt.Focused("Body") {
		t.Fatal("the KeyTip of a disabled gallery did not give the focus back")
	}
	k.click(t, k.items(t, false)["Style 1"])
	k.click(t, k.button(t, "More"))
	k.click(t, k.button(t, "Next"))
	k.wantStrip(t, 0, 4)
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("a click on a disabled gallery's More opens its drop-down (%d items)", n)
	}
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("a click on a disabled gallery's item ran OnSelect %v", got)
	}
}

// Own case: the drop-down closes when the gallery is disabled, and when its
// tab is no longer shown, with no pick.
func TestRibbonGalleryDropDownCloses(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.r.AddTab("Insert")
	k.tt.Frame()
	k.openDropDown(t)
	k.g.SetDisabled(true)
	k.tt.Frame()
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("disabling the gallery leaves its drop-down open (%d items)", n)
	}
	k.g.SetDisabled(false)
	k.tt.Frame()
	k.openDropDown(t)
	k.r.SetTab(1)
	k.tt.Frame()
	k.r.SetTab(0)
	k.tt.Frame()
	if n := len(k.items(t, true)); n != 0 {
		t.Fatalf("the drop-down is open again after its tab was hidden (%d items)", n)
	}
	if got := gotPosted(k.picks); len(got) != 0 {
		t.Fatalf("closing ran OnSelect %v", got)
	}
}

// Own case: columns under 1 are 1.
func TestRibbonGalleryZeroColumns(t *testing.T) {
	withMac(t, false)
	r := Ribbon().AddTab("Home").AddGroup("Styles")
	g := r.AddGallery(galleryItems(3), 0).SetAccessibleName("Gallery")
	k := &galleryKit{r: r, g: g, tt: tester(t, VBox(0).Add(r, Label("body")))}
	k.wantStrip(t, 0, 1)
	k.click(t, k.button(t, "Next"))
	k.wantStrip(t, 1, 2)
}

// Own case: SetItems while the drop-down is open moves its arrows off an
// item that is gone: Down then goes to the first item.
func TestRibbonGallerySetItemsWhileOpen(t *testing.T) {
	k := newGalleryKit(t, 10)
	k.g.SetSelected(9)
	k.tt.Frame()
	k.openDropDown(t)
	k.g.SetItems(galleryItems(5))
	k.tt.Frame()
	typeTip(k.tt, ui.KeyDown)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{0}) {
		t.Fatalf("Down, Enter after SetItems of 5 items with 9 selected: OnSelect %v, want [0]", got)
	}
	// The first item gone: 5 of 10, then 5 items.
	k.g.SetItems(galleryItems(10)).SetSelected(5)
	k.tt.Frame()
	k.openDropDown(t)
	k.g.SetItems(galleryItems(5))
	k.tt.Frame()
	typeTip(k.tt, ui.KeyEnter)
	typeTip(k.tt, ui.KeyDown)
	typeTip(k.tt, ui.KeyEnter)
	if got := gotPosted(k.picks); !slices.Equal(got, []int{0}) {
		t.Fatalf("Enter, Down, Enter after SetItems of 5 items with 5 selected: OnSelect %v, want [0]", got)
	}
}

// Own case: an item's image shows above its text, 32 DIPs high.
func TestRibbonGalleryImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := range img.Pix {
		img.Pix[i] = []uint8{ribbonRed.R, ribbonRed.G, ribbonRed.B, ribbonRed.A}[i%4]
	}
	items := galleryItems(4)
	items[1].Image = ui.NewBitmap(img)
	k := newGalleryKit(t, 0)
	k.g.SetItems(items)
	k.tt.Frame()
	strip := k.items(t, false)
	ink, ok := ribbonRedInk(k.tt.Image(), strip["Style 1"].Bounds)
	if !ok || !near1(ink.H, 32) {
		t.Fatalf("Style 1's image is %v (found %v), want 32 high", ink, ok)
	}
	if _, ok := ribbonRedInk(k.tt.Image(), strip["Style 0"].Bounds); ok {
		t.Fatal("Style 0, with no image, shows one")
	}
}
