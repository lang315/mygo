package widget

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
)

// nodeOf returns the node of role r labelled label in the last frame.
func nodeOf(t *testing.T, tt *ui.Tester, r ui.Role, label string) ui.TestNode {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r && n.Label == label {
			return n
		}
	}
	t.Fatalf("no node of role %v labelled %q", r, label)
	return ui.TestNode{}
}

// inkRows returns the first and the last pixel row inside box that differ
// from the page's colour: where the glyphs or the outline really are.
func inkRows(tt *ui.Tester, box ui.Rect) (first, last int) {
	img, bg := tt.Image(), page(tt)
	first, last = -1, -1
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if img.RGBAAt(x, y) != bg {
				if first < 0 {
					first = y
				}
				last = y
				break
			}
		}
	}
	return first, last
}

// In every view the check box and the text of an item are on one line: as
// Details lays them out, in one row whose items centre. It is measured on the
// text element and on the pixels, not on the item that holds both.
func TestListViewCheckBoxAndTextShareALineInEveryView(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().CheckBoxes().SetView(m.mode)
			tt := tester(t, lv)
			box := nodeOf(t, tt, ui.RoleCheckBox, "Check Apple")
			text := nodeOf(t, tt, ui.RoleText, "Apple")
			if m.mode == ViewLargeIcon {
				return // its box sits at the corner of the picture, not by the text
			}
			if m.mode != ViewDetails {
				item := nodeOf(t, tt, ui.RoleListItem, "Apple")
				if d := item.Bounds.Y + item.Bounds.H/2 - (text.Bounds.Y + text.Bounds.H/2); d < -2 || d > 2 {
					t.Errorf("%s: the text's line is %v px from the middle of its item: want it centred", m.name, d)
				}
			}
			bc, tc := box.Bounds.Y+box.Bounds.H/2, text.Bounds.Y+text.Bounds.H/2
			if d := bc - tc; d < -1.5 || d > 1.5 {
				t.Errorf("%s: the box's centre is at y %v, the text's at %v: %v apart", m.name, bc, tc, d)
			}
			bf, bl := inkRows(tt, box.Bounds)
			tf, tl := inkRows(tt, text.Bounds)
			if bf < 0 || tf < 0 {
				t.Fatalf("%s: no ink in the box %v or the text %v", m.name, box.Bounds, text.Bounds)
			}
			if d := float32(bf+bl)/2 - float32(tf+tl)/2; d < -2.5 || d > 2.5 {
				t.Errorf("%s: the box's outline is centred at y %v, the glyphs at %v", m.name, float32(bf+bl)/2, float32(tf+tl)/2)
			}
		})
	}
}

// An idle check box has no panel behind it where there is no picture under
// it: on a red page a panel would show as a patch that is not red. Only a
// LargeIcon item with an image gets one, for readability.
func TestListViewIdleCheckBoxesHaveNoPanelWithoutAPicture(t *testing.T) {
	red := color.RGBA{200, 30, 30, 255}
	redIn := func(lv *ListViewWidget[vitem, string]) int {
		lv.Apply(NewStyle().Background(Fixed(ui.RGB(200, 30, 30))))
		lv.SetChecked([]string{"b"}, true) // any checked: every box shows
		tt := tester(t, lv)
		box := nodeOf(t, tt, ui.RoleCheckBox, "Check Apple")
		n := 0
		for y := int(box.Bounds.Y); y < int(box.Bounds.Y+box.Bounds.H); y++ {
			for x := int(box.Bounds.X); x < int(box.Bounds.X+box.Bounds.W); x++ {
				if tt.Image().RGBAAt(x, y) == red {
					n++
				}
			}
		}
		return n
	}
	// The small views hold their own plain boxes, in the row of the text; the
	// large icons' are the grid's, at the corner of the picture.
	if n := redIn(newView().CheckBoxes().SetView(ViewLargeIcon)); n < 150 {
		t.Errorf("large icons without pictures: %d red pixels in an idle box's bounds: a panel is behind it", n)
	}
	// With a picture under the large icon's box, the panel keeps it readable.
	il := ImageList(16)
	if err := il.Add("doc", pngOf(t, 16, color.RGBA{0, 0, 255, 255})); err != nil {
		t.Fatal(err)
	}
	withPicture := redIn(newView().CheckBoxes().SetImageList(il, func(*vitem) string { return "doc" }).SetView(ViewLargeIcon))
	if withPicture >= 150 {
		t.Errorf("large icons with pictures: %d red pixels in an idle box: want the readability panel", withPicture)
	}
}

// A ListView packed to expand and fill takes the room its boxes leave, however
// the boxes nest, and builds only the rows in it, in every view: it fills the
// cross axis of the box that holds it (the marker the box layout reads).
func TestListViewPackedInBoxesBuildsOnlyTheRowsInView(t *testing.T) {
	const rows = 50_000
	items := make([]vitem, rows)
	for i := range items {
		items[i] = vitem{fmt.Sprint("k", i), fmt.Sprint("Item ", i), "kind"}
	}
	nestings := []struct {
		name string
		wrap func(w Control) Control
	}{
		{"VBox", func(w Control) Control { return VBox(4).Pack(w, true, true) }},
		{"VBox>HBox", func(w Control) Control { return VBox(4).Pack(HBox(4).Pack(w, true, true), true, true) }},
		{"HBox>VBox", func(w Control) Control { return HBox(4).Pack(VBox(4).Pack(w, true, true), true, true) }},
		{"VBox>HBox>VBox with a label", func(w Control) Control {
			return VBox(4).Pack(HBox(4).Pack(VBox(4).Add(Label("title")).Pack(w, true, true), true, true), true, true)
		}},
	}
	for _, m := range allModes {
		role := ui.RoleListItem
		if m.mode == ViewDetails {
			role = ui.RoleRow
		}
		for _, n := range nestings {
			t.Run(m.name+" in "+n.name, func(t *testing.T) {
				lv := newView().SetRows(items).SetView(m.mode)
				f := NewForm("t", 400, 460)
				f.SetContent(n.wrap(lv))
				tt := TestForm(t, f)
				tt.Settle()
				got := len(tt.FindAll(ByRole(role)))
				if got < 3 || got > 120 {
					t.Errorf("%d items built of %d, want only those that fit", got, rows)
				}
			})
		}
	}
}

// SetImageList draws the list's images at its Size in the icon views and the
// list, as ListBox and TreeView do.
func TestListViewSetImageListDrawsAtTheListsSize(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	il := ImageList(24)
	if err := il.Add("doc", pngOf(t, 48, blue)); err != nil {
		t.Fatal(err)
	}
	key := func(v *vitem) string { return "doc" }
	for _, m := range []ViewMode{ViewLargeIcon, ViewSmallIcon, ViewList} {
		t.Run(m.String(), func(t *testing.T) {
			lv := newView().SetImageList(il, key).SetView(m)
			tt := tester(t, lv)
			r := nodeOf(t, tt, ui.RoleListItem, "Apple").Bounds
			if n := bluePixels2(tt, r); n < 24*24 || n > 24*24+4*24 {
				t.Errorf("%v: %d blue pixels in the item, want about %d", m, n, 24*24)
			}
		})
	}
}

// Details draws the first column's picture, 16 DIPs, from SetImage.
func TestListViewSetImageShowsBeforeTheFirstColumnsTextInDetails(t *testing.T) {
	blue := color.RGBA{0, 0, 255, 255}
	il := ImageList(16)
	if err := il.Add("doc", pngOf(t, 16, blue)); err != nil {
		t.Fatal(err)
	}
	lv := newView().SetImage(func(v *vitem) *ui.Bitmap { return il.Image("doc") })
	tt := tester(t, lv)
	if n := bluePixels2(tt, find(t, tt, "Apple")); n < 16*16 {
		t.Errorf("Details: %d blue pixels in the row, want the picture's %d", n, 16*16)
	}
}

// A check a setter set shows in the box in every view: clicking the box
// unchecks it, which the user did; and a click on the box leaves the choice.
func TestListViewShowsTheChecksASetterSetInEveryView(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().CheckBoxes().SetView(m.mode)
			ev := watched(lv)
			lv.SetChecked([]string{"b"}, true)
			tt := tester(t, lv)
			box := nodeOf(t, tt, ui.RoleCheckBox, "Check Banana")
			if !box.Checked {
				t.Fatalf("%s: the box of Banana, checked by a setter, shows unchecked", m.name)
			}
			tt.ClickAt(box.Bounds.X+box.Bounds.W/2, box.Bounds.Y+box.Bounds.H/2)
			tt.Frame()
			if got := ev.drain(); len(got) != 1 || got[0] != "check b false" {
				t.Fatalf("%s: clicking the box: events %v, want check b false", m.name, got)
			}
			if lv.IsChecked("b") || len(lv.Selected()) != 0 {
				t.Fatalf("%s: Checked %v, Selected %v: the click must check, not choose", m.name, lv.Checked(), lv.Selected())
			}
		})
	}
}
