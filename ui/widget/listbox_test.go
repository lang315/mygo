package widget

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// litem is an item of the test list boxes.
type litem struct{ id, name string }

func newListBox() *ListBoxWidget[litem, string] {
	return ListBox(func(i *litem) string { return i.id }, func(i *litem) string { return i.name })
}

func newChecked() *ListBoxWidget[litem, string] {
	return CheckedListBox(func(i *litem) string { return i.id }, func(i *litem) string { return i.name })
}

var fruit = []litem{{"a", "Apple"}, {"b", "Banana"}, {"c", "Cherry"}, {"d", "Date"}}

func fruitBox() *ListBoxWidget[litem, string] { return newListBox().SetItems(fruit) }

// checkBoxAt returns where the check box of the row showing text is: find
// gives the whole row, which has 8 DIPs of padding before its box.
func checkBoxAt(t *testing.T, tt *ui.Tester, text string) (x, y float32) {
	t.Helper()
	r := find(t, tt, text)
	return r.X + 16, r.Y + r.H/2
}

func TestListBoxShowsItemsInOrderAndClickChoosesByKey(t *testing.T) {
	selected := make(chan []string, 8)
	lb := fruitBox().OnSelect(func(ks []string) { selected <- ks })
	tt := tester(t, lb)
	a, b := find(t, tt, "Apple"), find(t, tt, "Banana")
	if !(a.Y < b.Y) || !shows(tt, "Cherry") || !shows(tt, "Date") {
		t.Fatalf("shows %q", tt.Texts())
	}
	click(t, tt, "Banana")
	if got := soon(t, selected); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("OnSelect got %v", got)
	}
	if got := lb.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Selected %v", got)
	}
}

func TestListBoxSettersFireNoUserEvents(t *testing.T) {
	events := make(chan string, 8)
	lb := newChecked().SetItems(fruit).
		OnSelect(func(ks []string) { events <- fmt.Sprint("select ", ks) }).
		OnCheck(func(k string, on bool) { events <- "check " + k })
	tt := tester(t, lb)
	lb.SetSelected("b")
	lb.SetChecked([]string{"a", "c"}, true)
	lb.SetItems(append([]litem{{"z", "Zucchini"}}, fruit...))
	lb.UpdateItem("a", func(i *litem) { i.name = "Apricot" })
	lb.RemoveItems("d")
	lb.ClearSelection()
	tt.Frame()
	tt.Frame()
	<-posted()
	select {
	case e := <-events:
		t.Fatalf("a program's change fired %q", e)
	default:
	}
}

func TestListBoxChoosesOneItemUnlessMulti(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	click(t, tt, "Apple")
	if err := tt.ClickWith(ui.Cmd, "Cherry"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"c"}) {
		t.Fatalf("Cmd-click added to the choice of a single list: %v", got)
	}
	if err := tt.ClickWith(ui.Shift, "Apple"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("Shift-click chose a range in a single list: %v", got)
	}
}

// ui treats a Selection as several: in a single list Cmd+A, Cmd-click on the
// chosen item and Shift-extend would run and be trimmed after. A single list
// binds ui's one choice instead, so they do what they mean in a single list.
func TestListBoxSingleCmdClickOnTheChosenItemKeepsIt(t *testing.T) {
	selected := make(chan []string, 8)
	lb := fruitBox().OnSelect(func(ks []string) { selected <- ks })
	tt := tester(t, lb)
	click(t, tt, "Banana")
	soon(t, selected)
	if err := tt.ClickWith(ui.Cmd, "Banana"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("Cmd-click on the chosen item left Selected %v: want it kept", got)
	}
	<-posted()
	select {
	case ks := <-selected:
		t.Fatalf("Cmd-click on the chosen item fired OnSelect(%v)", ks)
	default:
	}
}

func TestListBoxSingleShiftArrowsMoveTheChoice(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	click(t, tt, "Banana")
	tt.Key(ui.Shift, ui.KeyDown)
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"c"}) {
		t.Fatalf("Shift+Down from Banana: Selected %v, want Cherry", got)
	}
	tt.Key(ui.Shift, ui.KeyUp)
	tt.Key(ui.Shift, ui.KeyUp)
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("Shift+Up twice: Selected %v, want Apple", got)
	}
	tt.Key(ui.Shift, ui.KeyUp) // at the top: stays
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("Shift+Up at the top: Selected %v", got)
	}
}

// Cmd+A in a single list changes nothing, fires nothing, and costs nothing by
// the size of the list.
func TestListBoxSingleCmdAChangesNothingAndCostsNothingBySize(t *testing.T) {
	cost := func(n int) (allocs float64, events int) {
		items := make([]litem, n)
		for i := range items {
			items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
		}
		selected := make(chan []string, 64)
		lb := newListBox().SetItems(items).OnSelect(func(ks []string) { selected <- ks })
		tt := tester(t, lb)
		click(t, tt, "Item 2")
		soon(t, selected)
		allocs = allocrun.Least(10, func() {
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Frame()
		})
		<-posted()
		if got := lb.Selected(); !slices.Equal(got, []string{"k2"}) {
			t.Fatalf("Cmd+A in a single list of %d items left Selected %v", n, got)
		}
		return allocs, len(selected)
	}
	small, smallEvents := cost(100)
	big, bigEvents := cost(100_000)
	t.Logf("Cmd+A: %.0f allocs on 100 items, %.0f on 100000", small, big)
	if smallEvents+bigEvents != 0 {
		t.Errorf("Cmd+A fired OnSelect %d times", smallEvents+bigEvents)
	}
	if big > small*1.1+10 {
		t.Errorf("Cmd+A allocated %.0f objects on 100000 items, %.0f on 100: want the same", big, small)
	}
}

func TestListBoxMultiChoosesSeveralAndReportsThemInListOrder(t *testing.T) {
	selected := make(chan []string, 8)
	lb := fruitBox().Multi().OnSelect(func(ks []string) { selected <- ks })
	tt := tester(t, lb)
	click(t, tt, "Cherry")
	soon(t, selected)
	if err := tt.ClickWith(ui.Cmd, "Apple"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := soon(t, selected); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("OnSelect got %v, want the items in the order of the list", got)
	}
	if err := tt.ClickWith(ui.Shift, "Date"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	// Shift-click chooses from the item last chosen, Apple, to the one clicked.
	if got := lb.Selected(); !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("Shift-click from Apple to Date: Selected %v", got)
	}
	if lb.CountSelected() != 4 {
		t.Fatalf("CountSelected %d", lb.CountSelected())
	}
}

// Selected is in the order of the list, whether few items are chosen (looked
// up by place) or many (found walking the list).
func TestListBoxSelectedAndCheckedAreInTheOrderOfTheList(t *testing.T) {
	items := make([]litem, 40)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	lb := newChecked().SetItems(items).Multi()
	lb.SetSelected("k30", "k2", "k17")
	lb.SetChecked([]string{"k9", "k3"}, true)
	if got := lb.Selected(); !slices.Equal(got, []string{"k2", "k17", "k30"}) {
		t.Errorf("Selected %v, want the order of the list", got)
	}
	if got := lb.Checked(); !slices.Equal(got, []string{"k3", "k9"}) {
		t.Errorf("Checked %v, want the order of the list", got)
	}
	var many []string
	for i := 39; i >= 10; i-- {
		many = append(many, fmt.Sprint("k", i))
	}
	lb.SetSelected(many...)
	if got := lb.Selected(); !slices.IsSortedFunc(got, func(a, b string) int { return slices.Index(many, b) - slices.Index(many, a) }) || len(got) != 30 {
		t.Errorf("Selected %v: want 30 items in the order of the list", got)
	}
}

// A click ui took and a setter that removed its item, on another goroutine,
// before the list built again: the item is not chosen, and no event runs for
// it. The test hands ui what the click would have: its set of a list of
// several, or its cursor in a single list.
func TestListBoxItemRemovedBeforeTheUsersChoiceIsTakenIsNotChosen(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprint("multi ", multi), func(t *testing.T) {
			events := make(chan []string, 4)
			lb := fruitBox().OnSelect(func(ks []string) { events <- ks })
			if multi {
				lb.Multi()
			}
			tester(t, lb)
			mu.Lock()
			if multi {
				lb.chosen.shown.Add("b")
			} else {
				lb.cursor = 1 // Banana
			}
			mu.Unlock()
			lb.RemoveItems("b")
			mu.Lock()
			lb.takeUser()
			mu.Unlock()
			if got := lb.Selected(); len(got) != 0 {
				t.Errorf("Selected %v: the removed item came back", got)
			}
			if lb.chosen.shown.Has("b") {
				t.Error("ui's set still holds the removed item")
			}
			<-posted()
			select {
			case e := <-events:
				t.Errorf("OnSelect fired with %v for a removed item", e)
			default:
			}
		})
	}
}

func TestListBoxChoiceFollowsItsItemWhenItemsComeAbove(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	click(t, tt, "Cherry")
	lb.SetItems(append([]litem{{"x", "Fig"}, {"y", "Grape"}}, fruit...))
	tt.Frame()
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"c"}) {
		t.Fatalf("Selected %v after items came above", got)
	}
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"d"}) {
		t.Fatalf("Down moved to %v, want Date, the row after Cherry", got)
	}
}

func TestListBoxRemovedItemsLeaveTheSetsSilentlyAndComeBackClean(t *testing.T) {
	events := make(chan string, 8)
	lb := newChecked().SetItems(fruit).Multi().
		OnSelect(func(ks []string) { events <- "select" }).OnCheck(func(k string, on bool) { events <- "check" })
	lb.SetSelected("a", "b")
	lb.SetChecked([]string{"a", "c"}, true)
	tt := tester(t, lb)
	lb.RemoveItems("a")
	tt.Frame()
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"b"}) {
		t.Errorf("Selected %v after removing Apple", got)
	}
	if got := lb.Checked(); !slices.Equal(got, []string{"c"}) {
		t.Errorf("Checked %v after removing Apple", got)
	}
	lb.SetItems(fruit[:3]) // Apple is added again
	tt.Frame()
	if lb.IsChecked("a") || slices.Contains(lb.Selected(), "a") {
		t.Errorf("a re-added item is checked %v, selected %v: want it clean", lb.IsChecked("a"), lb.Selected())
	}
	lb.SetItems(fruit[1:2]) // only Banana stays; Cherry goes with its check
	if lb.IsChecked("c") {
		t.Error("an item SetItems dropped is still checked")
	}
	<-posted()
	select {
	case e := <-events:
		t.Fatalf("removing items fired %q", e)
	default:
	}
}

// As DataGridView's SetRows: of items with the same key, the last stays,
// where the first was.
func TestListBoxItemsOfTheSameKeyKeepTheLastWhereTheFirstWas(t *testing.T) {
	lb := newListBox().SetItems([]litem{{"a", "first a"}, {"b", "b"}, {"a", "last a"}})
	if lb.Len() != 2 {
		t.Fatalf("Len %d, want 2", lb.Len())
	}
	if it, _ := lb.Item("a"); it.name != "last a" {
		t.Fatalf("Item a %v, want the last", it)
	}
	tt := tester(t, lb)
	first, second := find(t, tt, "last a"), find(t, tt, "b")
	if first.Y >= second.Y {
		t.Fatalf("a at %v is not above b at %v: the last a stays where the first was", first, second)
	}
}

func TestListBoxUnknownKeysDoNothing(t *testing.T) {
	events := make(chan string, 4)
	lb := newChecked().SetItems(fruit).OnCheck(func(k string, on bool) { events <- k })
	lb.UpdateItem("zz", func(i *litem) { i.name = "changed" })
	lb.SetSelected("zz")
	lb.SetChecked([]string{"zz"}, true)
	lb.RemoveItems("zz")
	if lb.Len() != 4 || len(lb.Selected()) != 0 || len(lb.Checked()) != 0 {
		t.Errorf("Len %d, Selected %v, Checked %v", lb.Len(), lb.Selected(), lb.Checked())
	}
	if _, ok := lb.Item("zz"); ok {
		t.Error("Item found an unknown key")
	}
	tester(t, lb)
	<-posted()
	select {
	case k := <-events:
		t.Errorf("OnCheck fired for %q", k)
	default:
	}
}

func TestListBoxAddUpdateRemoveClearShow(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	lb.AddItem(litem{"e", "Elderberry"})
	lb.AddItems([]litem{{"f", "Fig"}, {"a", "Apple!"}}) // a again: replaces Apple in place
	tt.Frame()
	if !shows(tt, "Elderberry") || !shows(tt, "Fig") || !shows(tt, "Apple!") || shows(tt, "Apple") || lb.Len() != 6 {
		t.Fatalf("shows %q, Len %d", tt.Texts(), lb.Len())
	}
	lb.UpdateItem("b", func(i *litem) { i.name = "Blueberry" })
	lb.RemoveItems("c", "d")
	tt.Frame()
	if !shows(tt, "Blueberry") || shows(tt, "Cherry") || shows(tt, "Date") || lb.Len() != 4 {
		t.Fatalf("shows %q, Len %d", tt.Texts(), lb.Len())
	}
	lb.Clear()
	tt.Frame()
	if lb.Len() != 0 || shows(tt, "Blueberry") {
		t.Fatalf("Clear left %d items, shows %q", lb.Len(), tt.Texts())
	}
}

func TestListBoxCallbacksRunUnderThePackageLock(t *testing.T) {
	var keyLocked, textLocked, updateLocked bool
	lb := ListBox(func(i *litem) string { keyLocked = keyLocked || lockHeld(); return i.id },
		func(i *litem) string { textLocked = textLocked || lockHeld(); return i.name })
	lb.SetItems([]litem{{"a", "Apple"}})
	lb.UpdateItem("a", func(i *litem) { updateLocked = lockHeld() })
	tester(t, lb)
	if !keyLocked || !textLocked || !updateLocked {
		t.Errorf("under the lock: key %v, text %v, UpdateItem %v: want all true, as the docs say", keyLocked, textLocked, updateLocked)
	}
}

func TestListBoxPlainHasNoCheckBoxes(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	x, y := checkBoxAt(t, tt, "Apple")
	tt.ClickAt(x, y) // where a box would be: it chooses the row
	tt.Frame()
	if len(lb.Checked()) != 0 || !slices.Equal(lb.Selected(), []string{"a"}) {
		t.Fatalf("Checked %v, Selected %v", lb.Checked(), lb.Selected())
	}
}

func TestCheckedListBoxChecksReportTheUsersAndSettersAreSilent(t *testing.T) {
	checked := make(chan string, 8)
	lb := newChecked().SetItems(fruit).OnCheck(func(k string, on bool) { checked <- fmt.Sprint(k, on) })
	tt := tester(t, lb)
	x, y := checkBoxAt(t, tt, "Cherry")
	tt.ClickAt(x, y)
	tt.Frame()
	if got := soon(t, checked); got != "ctrue" {
		t.Fatalf("OnCheck got %q", got)
	}
	x, y = checkBoxAt(t, tt, "Apple")
	tt.ClickAt(x, y)
	tt.Frame()
	soon(t, checked)
	if got := lb.Checked(); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("Checked %v, want the checked items in the order of the list", got)
	}
	if !lb.IsChecked("a") || lb.IsChecked("b") {
		t.Fatalf("IsChecked a %v, b %v", lb.IsChecked("a"), lb.IsChecked("b"))
	}
	if len(lb.Selected()) != 0 {
		t.Errorf("checking chose %v", lb.Selected())
	}
	tt.ClickAt(x, y) // uncheck Apple
	tt.Frame()
	if got := soon(t, checked); got != "afalse" {
		t.Fatalf("OnCheck got %q", got)
	}
}

func TestCheckedListBoxSpaceChecksTheItemsChosen(t *testing.T) {
	checked := make(chan string, 8)
	lb := newChecked().SetItems(fruit).Multi().OnCheck(func(k string, on bool) { checked <- fmt.Sprint(k, on) })
	tt := tester(t, lb)
	click(t, tt, "Banana")
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if got := soon(t, checked); got != "btrue" {
		t.Fatalf("Space: OnCheck got %q", got)
	}
	tt.Key(0, ui.KeySpace) // all chosen are checked: unchecks
	tt.Frame()
	if got := soon(t, checked); got != "bfalse" {
		t.Fatalf("Space again: OnCheck got %q", got)
	}
	if len(lb.Checked()) != 0 {
		t.Fatalf("Checked %v", lb.Checked())
	}
}

// Space checks the item the cursor is on in a single list.
func TestCheckedListBoxSingleSpaceChecksTheChosenItem(t *testing.T) {
	checked := make(chan string, 8)
	lb := newChecked().SetItems(fruit).OnCheck(func(k string, on bool) { checked <- fmt.Sprint(k, on) })
	tt := tester(t, lb)
	click(t, tt, "Cherry")
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if got := soon(t, checked); got != "ctrue" {
		t.Fatalf("Space: OnCheck got %q", got)
	}
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if got := soon(t, checked); got != "cfalse" {
		t.Fatalf("Space again: OnCheck got %q", got)
	}
}

// A check a setter set shows in the box: clicking the box unchecks it, which
// the user did.
func TestCheckedListBoxShowsTheChecksASetterSet(t *testing.T) {
	checked := make(chan string, 8)
	lb := newChecked().SetItems(fruit).OnCheck(func(k string, on bool) { checked <- fmt.Sprint(k, on) })
	lb.SetChecked([]string{"b"}, true)
	tt := tester(t, lb)
	x, y := checkBoxAt(t, tt, "Banana")
	tt.ClickAt(x, y)
	tt.Frame()
	if got := soon(t, checked); got != "bfalse" {
		t.Fatalf("clicking the box of an item a setter checked: OnCheck got %q, want bfalse", got)
	}
	if lb.IsChecked("b") {
		t.Fatal("the item is still checked")
	}
}

func TestListBoxKeyboard(t *testing.T) {
	lb := fruitBox()
	tt := tester(t, lb)
	click(t, tt, "Banana")
	for _, c := range []struct {
		key  ui.Key
		want string
	}{{ui.KeyDown, "c"}, {ui.KeyUp, "b"}, {ui.KeyEnd, "d"}, {ui.KeyHome, "a"}} {
		tt.Key(0, c.key)
		tt.Frame()
		if got := lb.Selected(); !slices.Equal(got, []string{c.want}) {
			t.Fatalf("key %v: Selected %v, want %q", c.key, got, c.want)
		}
	}
	tt.Key(0, ui.KeyD)
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"d"}) {
		t.Fatalf("typing d: Selected %v, want Date", got)
	}
}

func TestListBoxEnterActivates(t *testing.T) {
	activated := make(chan string, 4)
	lb := fruitBox().OnActivate(func(k string) { activated <- k })
	tt := tester(t, lb)
	click(t, tt, "Cherry")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if got := soon(t, activated); got != "c" {
		t.Fatalf("Enter activated %q", got)
	}
}

func TestListBoxImageIsAskedForTheRowsShown(t *testing.T) {
	calls := 0
	lb := fruitBox().SetImage(func(i *litem) *ui.Bitmap { calls++; return nil })
	tester(t, lb)
	if calls == 0 {
		t.Fatal("SetImage's function was not called")
	}
}

// listCost returns the objects and bytes a click on a row allocates in a list
// of n items.
func listCost(t *testing.T, n int) (allocs float64, bytes int64) {
	t.Helper()
	items := make([]litem, n)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	lb := newListBox().SetItems(items)
	tt := tester(t, lb)
	click(t, tt, "Item 1")
	pick := func(name string) func() {
		return func() {
			if err := tt.Click(name); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
		}
	}
	a, b := pick("Item 2"), pick("Item 3")
	a()
	b()
	var k int
	allocs = allocrun.Least(20, func() {
		if k++; k%2 == 0 {
			a()
		} else {
			b()
		}
	})
	res := testing.Benchmark(func(bb *testing.B) {
		for i := 0; i < bb.N; i++ {
			if i%2 == 0 {
				a()
			} else {
				b()
			}
		}
	})
	return allocs, res.AllocedBytesPerOp()
}

// A click in a list of a million items costs what one in a list of a
// thousand does: nothing is cloned per click.
func TestListBoxClickAllocationsDoNotGrowWithTheList(t *testing.T) {
	smallAllocs, smallBytes := listCost(t, 1000)
	bigAllocs, bigBytes := listCost(t, 1_000_000)
	t.Logf("a click on 1000 items: %.0f allocs, %d B/op; on 1000000: %.0f allocs, %d B/op", smallAllocs, smallBytes, bigAllocs, bigBytes)
	if bigAllocs > smallAllocs*1.1+10 {
		t.Errorf("a click allocated %.0f objects on a million items, %.0f on a thousand: want the same", bigAllocs, smallAllocs)
	}
	if bigBytes > smallBytes*11/10+1<<10 {
		t.Errorf("a click allocated %d bytes on a million items, %d on a thousand: want the same", bigBytes, smallBytes)
	}
	if bigBytes > 512<<10 {
		t.Errorf("a click allocated %d bytes: want far less than a clone of the items (MB)", bigBytes)
	}
}

func TestListBoxEnsureVisibleScrollsToAnItem(t *testing.T) {
	items := make([]litem, 300)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	lb := newListBox().SetItems(items)
	tt := tester(t, lb)
	if r, ok := tt.Find("Item 250"); ok && r.Y < 600 {
		t.Fatal("Item 250 is on screen before it is asked for")
	}
	lb.EnsureVisible("k250")
	tt.Frame()
	tt.Frame()
	if r, ok := tt.Find("Item 250"); !ok || r.Y < 0 || r.Y+r.H > 600 {
		t.Errorf("EnsureVisible did not scroll to the item: %v %v", r, ok)
	}
	if got := lb.Selected(); len(got) != 0 {
		t.Errorf("EnsureVisible chose %v", got)
	}
}

// A request for an item in view moves nothing, and does not wait to scroll
// later when the items change.
func TestListBoxEnsureVisibleOfAnItemInViewMovesNothingAndEnds(t *testing.T) {
	items := make([]litem, 1000)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	lb := newListBox().SetItems(items)
	tt := tester(t, lb)
	before := find(t, tt, "Item 3")
	lb.EnsureVisible("k3")
	tt.Frame()
	tt.Frame()
	if after := find(t, tt, "Item 3"); after.Y != before.Y {
		t.Errorf("an item in view moved from %v to %v", before.Y, after.Y)
	}
	rev := slices.Clone(items)
	slices.Reverse(rev)
	lb.SetItems(rev) // k3 is now row 996
	tt.Frame()
	tt.Frame()
	mu.Lock()
	pending := lb.hasReveal
	mu.Unlock()
	if pending {
		t.Error("the request is still pending")
	}
}

// An item far away, for a list whose rows are not measured yet, shows (the
// FontDialog test checks it against the list's own bounds).
func TestListBoxEnsureVisibleShowsAFarItemOfAListNotMeasuredYet(t *testing.T) {
	items := make([]litem, 300)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	lb := newListBox().SetItems(items).SetSize(0, 200)
	lb.EnsureVisible("k290") // before the first frame: no row is measured
	tt := tester(t, lb)
	if r, ok := tt.Find("Item 290"); !ok || r.Y < 0 || r.Y+r.H > 600 {
		t.Errorf("Item 290: %v %v, want it in the window", r, ok)
	}
}

// A request for an item in view leaves it where it is, even after the list
// scrolled: it does not center it.
func TestListBoxEnsureVisibleMovesAnItemInViewByNothing(t *testing.T) {
	items := make([]litem, 300)
	for i := range items {
		items[i] = litem{fmt.Sprint("k", i), fmt.Sprint("Item ", i)}
	}
	lb := newListBox().SetItems(items)
	tt := tester(t, lb)
	lb.EnsureVisible("k100")
	tt.Frame()
	tt.Frame()
	at := find(t, tt, "Item 100")
	if at.Y < 500 {
		t.Errorf("Item 100 is at %v: as little as shows it is at the bottom, not the middle", at.Y)
	}
	lb.EnsureVisible("k100")
	tt.Frame()
	tt.Frame()
	if now := find(t, tt, "Item 100"); now.Y != at.Y {
		t.Errorf("Item 100 moved from %v to %v", at.Y, now.Y)
	}
}
