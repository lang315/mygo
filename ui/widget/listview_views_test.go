package widget

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// "Check all" in the Details header checks every row, so it reports every key
// it flipped, as a click on one row's box does.
func TestListViewCheckAllInTheHeaderFiresOnCheckForEachRow(t *testing.T) {
	lv := newView().CheckBoxes()
	ev := watched(lv)
	tt := tester(t, lv)
	click(t, tt, "Check all")
	got := ev.drain()
	slices.Sort(got)
	if want := []string{"check a true", "check b true", "check c true", "check d true"}; !slices.Equal(got, want) {
		t.Fatalf("Check all: events %v, want %v", got, want)
	}
	if len(lv.Checked()) != 4 {
		t.Fatalf("Checked %v", lv.Checked())
	}
	click(t, tt, "Check all")
	got = ev.drain()
	slices.Sort(got)
	if want := []string{"check a false", "check b false", "check c false", "check d false"}; !slices.Equal(got, want) {
		t.Fatalf("Check all again: events %v, want %v", got, want)
	}
	if len(lv.Checked()) != 0 {
		t.Fatalf("Checked %v", lv.Checked())
	}
}

// A setter's check and a whole-set replacement by the user in the same frame
// (ui gives up its log of changes): the user's change is reported, the
// setter's key is not reported as unchecked, and it stays checked.
func TestListViewCommitKeepsASettersCheckOutOfTheUsersEvents(t *testing.T) {
	rows := make([]vitem, 20)
	keys := make([]string, 20)
	for i := range rows {
		keys[i] = fmt.Sprint("k", i)
		rows[i] = vitem{keys[i], fmt.Sprint("Item ", i), "kind"}
	}
	lv := newView().SetRows(rows).CheckBoxes()
	ev := watched(lv)
	tt := tester(t, lv)
	lv.SetChecked(keys[1:], true)
	tt.Frame() // the table takes them
	ev.drain()
	lv.SetChecked(keys[:1], true) // a setter checks k0
	var kept ui.Selection[string]
	kept.Add("k1")
	kept.Add("k2")
	mu.Lock()
	lv.t.checked.shown.CopyFrom(&kept) // the user's whole-set change: only k1 and k2 stay checked
	mu.Unlock()
	lv.t.commit(&lv.t.checked)
	got := ev.drain()
	if slices.Contains(got, "check k0 false") {
		t.Fatalf("the setter's check was reported as the user unchecking k0: %v", got)
	}
	if len(got) != 17 {
		t.Fatalf("%d events %v, want the 17 the user unchecked (k3 to k19)", len(got), got)
	}
	if !lv.IsChecked("k0") || !lv.IsChecked("k1") || lv.IsChecked("k3") {
		t.Fatalf("Checked %v: want k0 (the setter's), k1 and k2", lv.Checked())
	}
}

// The focus, the cursor and the item Shift extends from are the list view's,
// by key: they cross every switch of the view.
func TestListViewSwitchingTheViewCarriesFocusCursorAndAnchor(t *testing.T) {
	for _, from := range allModes {
		for _, to := range allModes {
			if from.mode == to.mode {
				continue
			}
			t.Run(from.name+" to "+to.name, func(t *testing.T) {
				// The keys move on from the item that was chosen.
				lv := newView().SetView(from.mode)
				tt := tester(t, lv)
				click(t, tt, "Banana")
				lv.SetView(to.mode)
				tt.Frame()
				tt.Frame()
				next := ui.KeyDown
				if to.mode == ViewLargeIcon || to.mode == ViewSmallIcon {
					next = ui.KeyRight
				}
				tt.Key(0, next)
				tt.Frame()
				if got := lv.Selected(); !slices.Equal(got, []string{"c"}) {
					t.Fatalf("after switching, %v from Banana chose %v, want Cherry: focus and cursor must cross", next, got)
				}
				// Shift extends from the item that was chosen without Shift.
				lv = newView().SetView(from.mode)
				tt = tester(t, lv)
				click(t, tt, "Banana")
				lv.SetView(to.mode)
				tt.Frame()
				tt.Frame()
				if err := tt.ClickWith(ui.Shift, "Date"); err != nil {
					t.Fatal(err)
				}
				tt.Frame()
				if got := lv.Selected(); !slices.Equal(got, []string{"b", "c", "d"}) {
					t.Fatalf("after switching, Shift-click on Date chose %v, want Banana to Date", got)
				}
			})
		}
	}
}

// Details() options apply in every view: its keys, and its bottom panel.
func TestListViewDetailsKeysAndPanelWorkInEveryView(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			pressed := make(chan string, 4)
			lv := newView().SetView(m.mode)
			lv.Details().BottomPanel().OnKey(ui.KeyDelete, 0, func() { pressed <- "delete" })
			tt := tester(t, lv)
			if !shows(tt, "Rows: 4") {
				t.Fatalf("%s shows %q: want the bottom panel's counts", m.name, tt.Texts())
			}
			click(t, tt, "Banana")
			if !shows(tt, "Selected: 1") {
				t.Fatalf("%s shows %q: the panel's count of the rows chosen", m.name, tt.Texts())
			}
			tt.Key(0, ui.KeyDelete)
			tt.Frame()
			if got := soon(t, pressed); got != "delete" {
				t.Fatalf("OnKey(Delete) got %q", got)
			}
		})
	}
}

// The items chosen show in the accent in every view, focused or not.
func TestListViewChosenItemsShowInTheAccentInEveryView(t *testing.T) {
	th := ui.LightTheme()
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			lv.SetSelected("b") // chosen, and nothing focused
			tt := lightTester(t, lv)
			r := find(t, tt, "Banana")
			if first, _ := accentExtent(tt, int(r.Y+r.H/2), th.Accent); first < 0 {
				t.Fatalf("%s: no accent in the row of the chosen Banana at y %v", m.name, r.Y+r.H/2)
			}
			r = find(t, tt, "Cherry")
			if first, _ := accentExtent(tt, int(r.Y+r.H/2), th.Accent); first >= 0 && m.mode == ViewDetails {
				t.Fatalf("%s: accent in the row of Cherry, which is not chosen", m.name)
			}
		})
	}
}

// In SmallIcon and List the check box is centred on the text's line.
func TestListViewCheckBoxIsCentredOnTheTextInTheSmallViews(t *testing.T) {
	for _, m := range []ViewMode{ViewSmallIcon, ViewList} {
		t.Run(m.String(), func(t *testing.T) {
			lv := newView().CheckBoxes().SetView(m)
			tt := tester(t, lv)
			box, text := find(t, tt, "Check Apple"), find(t, tt, "Apple")
			bc, tc := box.Y+box.H/2, text.Y+text.H/2
			if d := bc - tc; d < -1.5 || d > 1.5 {
				t.Fatalf("%v: the box's centre is at y %v, the text's at %v: %v apart", m, bc, tc, d)
			}
		})
	}
}

// accentExtent returns the first and the last column of the accent in the
// pixel row y of the window.
func accentExtent(tt *ui.Tester, y int, accent ui.Color) (first, last int) {
	img := tt.Image()
	first, last = -1, -1
	for x := range img.Bounds().Dx() {
		c := img.RGBAAt(x, y)
		if near(c, accent) {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	return first, last
}

// The last tile of a row of SmallIcon has the room the first has on the
// other side.
func TestListViewSmallIconTilesHaveTheSameInsetOnBothSides(t *testing.T) {
	th := ui.LightTheme()
	lv := newView().SetView(ViewSmallIcon)
	lv.SetSelected("a", "b", "c", "d")
	tt := lightTester(t, lv)
	r := find(t, tt, "Apple")
	y := int(r.Y + r.H/2)
	first, last := accentExtent(tt, y, th.Accent)
	if first < 0 {
		t.Fatalf("no accent in the row of Apple at y %d", y)
	}
	w := tt.Image().Bounds().Dx()
	left, right := first, w-1-last
	t.Logf("tiles span x %d to %d of a %d wide window", first, last, w)
	if left < 4 || right < 4 || left-right < -2 || left-right > 2 {
		t.Fatalf("the tiles start %d px from the left edge and end %d px from the right: want the same room, 4 px or more", left, right)
	}
}
