package widget

import (
	"fmt"
	"image"
	"math"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
	"github.com/egoist/mygo/ui/widget/skin"
)

// frameLooks are the looks every frame case runs under: no skin and Calm,
// light and dark.
var frameLooks = []struct {
	name string
	skin *skin.Skin
	dark bool
}{{"nil light", nil, false}, {"nil dark", nil, true}, {"Calm light", skin.Calm, false}, {"Calm dark", skin.Calm, true}}

// useLook turns s on until the test ends.
func useLook(t *testing.T, s *skin.Skin) {
	t.Helper()
	skin.Use(s)
	t.Cleanup(func() { skin.Use(nil) })
}

// lookColors are the frame of a list in the look on, Calm's LineStrong or the
// plain theme's Border, and the content background.
func lookColors(dark bool) (frame, bg ui.Color) {
	if tk := skinhook.Get(dark); tk != nil {
		return tk.LineStrong, tk.Content
	}
	th := ui.LightTheme()
	if dark {
		th = ui.DarkTheme()
	}
	return th.Border, th.Background
}

// listControls are the five list controls at 300 by 120, each with n rows
// and the style st, in a box that keeps 8 DIPs of page around them (10 device
// pixels at 1.25).
func listControls(n int, st Style) map[string]func() Control {
	strs := make([]string, n)
	accts := make([]acct, n)
	items := make([]vitem, n)
	nodes := make([]tnode, n)
	for i := range n {
		s := "Row " + strconv.Itoa(i)
		strs[i], accts[i], items[i], nodes[i] = s, acct{id: i, name: s}, vitem{id: s, name: s}, tnode{s, s}
	}
	id := func(s *string) string { return *s }
	view := func() *ListViewWidget[vitem, string] {
		return ListView(func(v *vitem) string { return v.id },
			Column[vitem]{Title: "Name", Text: func(v *vitem) string { return v.name }, Width: 200}).SetRows(items)
	}
	boxed := func(w Control) Control { return VBox(0).Add(w).Padding(2) }
	return map[string]func() Control{
		"ListBox":            func() Control { return boxed(ListBox(id, id).SetItems(strs).SetSize(300, 120).Apply(st)) },
		"CheckedListBox":     func() Control { return boxed(CheckedListBox(id, id).SetItems(strs).SetSize(300, 120).Apply(st)) },
		"ListView":           func() Control { return boxed(view().SetSize(300, 120).Apply(st)) },
		"ListView LargeIcon": func() Control { return boxed(view().SetView(ViewLargeIcon).SetSize(300, 120).Apply(st)) },
		"ListView List":      func() Control { return boxed(view().SetView(ViewList).SetSize(300, 120).Apply(st)) },
		// The style given to the table a ListView shows its details in.
		"ListView Details()": func() Control {
			lv := view().SetSize(300, 120)
			lv.Details().Apply(st)
			return boxed(lv)
		},
		"TreeView": func() Control { return boxed(newTree().SetRoots(nodes).SetSize(300, 120).Apply(st)) },
		"DataGridView": func() Control {
			return boxed(DataGridView(func(a *acct) int { return a.id },
				Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).SetRows(accts).SetSize(300, 120).Apply(st))
		},
	}
}

// listBox returns the box of the scrolling element of a list control: the
// node of its role.
func listBox(t *testing.T, tt *ui.Tester) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		switch n.Role {
		case ui.RoleList, ui.RoleTree, ui.RoleTable, ui.RoleGrid:
			return n.Bounds
		}
	}
	t.Fatal("no list, tree, table or grid node")
	return ui.Rect{}
}

// devicePx is the pixel at DIP x and device row y.
func devicePx(img *image.RGBA, x float32, y int, scale float32) ui.Color {
	c := img.RGBAAt(int(x*scale), y)
	return ui.RGB(c.R, c.G, c.B)
}

func sameColor(a, b ui.Color) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= 2 && int(y)-int(x) <= 2 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

// Each list control draws a frame of 1 device pixel on its top edge, in the
// look's strong line, at the scales 1 and 1.25.
func TestListControlsHaveAFrame(t *testing.T) {
	for _, look := range frameLooks {
		for name, mk := range listControls(3, Style{}) {
			for _, scale := range []float32{1, 1.25} {
				t.Run(fmt.Sprintf("%s/%s/%v", look.name, name, scale), func(t *testing.T) {
					useLook(t, look.skin)
					tt := tester(t, mk())
					tt.SetDark(look.dark)
					tt.SetScale(scale)
					tt.Frame()
					img, box := tt.Image(), listBox(t, tt)
					want, page := lookColors(look.dark)
					if sameColor(want, page) {
						t.Fatalf("the frame's color %v is the page's", want)
					}
					x := box.X + box.W/2
					top := int(math.Round(float64(box.Y * scale)))
					if got := devicePx(img, x, top, scale); !sameColor(got, want) {
						t.Fatalf("top edge at device row %d is %v, want the frame's %v (page %v)", top, got, want, page)
					}
					// One device pixel of one color: the page above it, and the
					// row under it the inside's, not a blend of the two.
					if got := devicePx(img, x, top-1, scale); !sameColor(got, page) {
						t.Fatalf("device row %d over the top edge is %v, want the page's %v", top-1, got, page)
					}
					if a, b := devicePx(img, x, top+1, scale), devicePx(img, x, top+2, scale); !sameColor(a, b) || sameColor(a, want) {
						t.Fatalf("device rows %d and %d under the top edge are %v and %v, want the inside's color twice, not the frame's %v", top+1, top+2, a, b, want)
					}
				})
			}
		}
	}
}

// A style that sets a border of 0 takes the frame off: the top edge is the
// content background.
func TestListFrameYieldsToOwnStyle(t *testing.T) {
	for _, look := range frameLooks {
		t.Run(look.name, func(t *testing.T) {
			useLook(t, look.skin)
			id := func(s *string) string { return *s }
			tt := tester(t, ListBox(id, id).SetItems([]string{"A", "B", "C"}).SetSize(300, 120).
				Apply(NewStyle().BorderWidth(0)))
			tt.SetDark(look.dark)
			tt.Frame()
			img, box := tt.Image(), listBox(t, tt)
			_, want := lookColors(look.dark)
			if got := devicePx(img, box.X+box.W/2, int(math.Round(float64(box.Y))), 1); !sameColor(got, want) {
				t.Fatalf("top edge is %v, want the content background %v", got, want)
			}
		})
	}
}

// A style that sets only a border color recolors every list control's frame,
// keeping its width, though a table's style lands on the column around it.
func TestListFrameTakesOwnBorderColor(t *testing.T) {
	green := ui.RGB(0, 160, 0)
	for _, look := range frameLooks {
		for name, mk := range listControls(3, NewStyle().BorderColor(Fixed(green))) {
			t.Run(look.name+"/"+name, func(t *testing.T) {
				useLook(t, look.skin)
				tt := tester(t, mk())
				tt.SetDark(look.dark)
				tt.Frame()
				img, box := tt.Image(), listBox(t, tt)
				if got := devicePx(img, box.X+box.W/2, int(box.Y), 1); !sameColor(got, green) {
					t.Fatalf("top edge is %v, want the style's border color %v", got, green)
				}
			})
		}
	}
}

// Every list control's own style overrides the frame: a border of 0 leaves
// no line at the top edge, and a background of its own shows under the rows,
// though a table's style lands on the column around the framed table.
func TestListFrameYieldsToOwnStyleEveryControl(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	for _, look := range frameLooks {
		for name, mk := range listControls(3, NewStyle().BorderWidth(0)) {
			t.Run(look.name+"/border/"+name, func(t *testing.T) {
				useLook(t, look.skin)
				tt := tester(t, mk())
				tt.SetDark(look.dark)
				tt.Frame()
				img, box := tt.Image(), listBox(t, tt)
				x, y := box.X+box.W/2, int(box.Y)
				if edge, in := devicePx(img, x, y, 1), devicePx(img, x, y+3, 1); !sameColor(edge, in) {
					t.Fatalf("top edge is %v, the inside %v: a line is left with a border of 0", edge, in)
				}
			})
		}
		for name, mk := range listControls(3, NewStyle().Background(Fixed(red))) {
			t.Run(look.name+"/background/"+name, func(t *testing.T) {
				useLook(t, look.skin)
				tt := tester(t, mk())
				tt.SetDark(look.dark)
				tt.Frame()
				img, box := tt.Image(), listBox(t, tt)
				if got := devicePx(img, box.X+box.W/2, int(box.Y+box.H)-4, 1); !sameColor(got, red) {
					t.Fatalf("under the rows is %v, want the style's background %v", got, red)
				}
			})
		}
	}
}

// Every list control draws the content background under its rows, not its
// parent's: here a blue box around it.
func TestListControlsHaveTheContentBackground(t *testing.T) {
	blue := ui.RGB(0, 0, 255)
	for _, look := range frameLooks {
		for name, mk := range listControls(1, Style{}) {
			t.Run(look.name+"/"+name, func(t *testing.T) {
				useLook(t, look.skin)
				tt := tester(t, VBox(0).Add(mk()).Background(blue))
				tt.SetDark(look.dark)
				tt.Frame()
				img, box := tt.Image(), listBox(t, tt)
				_, bg := lookColors(look.dark)
				if got := devicePx(img, box.X+box.W/2, int(box.Y+box.H)-4, 1); !sameColor(got, bg) {
					t.Fatalf("under the rows is %v, want the content background %v", got, bg)
				}
				if got := devicePx(img, box.X-4, int(box.Y+box.H/2), 1); !sameColor(got, blue) {
					t.Fatalf("beside the list is %v, want the parent's %v", got, blue)
				}
			})
		}
	}
}

// A chosen row's fill stays inside the frame: the left edge at the middle of
// the first row, chosen, is the frame's color.
func TestListChosenRowKeepsTheFrame(t *testing.T) {
	id := func(s *string) string { return *s }
	controls := map[string]func() Control{
		"ListBox": func() Control {
			return VBox(0).Add(ListBox(id, id).SetItems([]string{"Row 0", "Row 1"}).SetSelected("Row 0").SetSize(300, 120)).Padding(2)
		},
		"DataGridView": func() Control {
			return VBox(0).Add(DataGridView(func(a *acct) int { return a.id },
				Column[acct]{Title: "Name", Text: func(a *acct) string { return a.name }, Width: 200}).
				SetRows([]acct{{id: 0, name: "Row 0"}, {id: 1, name: "Row 1"}}).SetSelected(0).SetSize(300, 120)).Padding(2)
		},
	}
	for _, look := range frameLooks {
		for name, mk := range controls {
			t.Run(look.name+"/"+name, func(t *testing.T) {
				useLook(t, look.skin)
				tt := tester(t, mk())
				tt.SetDark(look.dark)
				tt.Frame()
				img, box, row := tt.Image(), listBox(t, tt), find(t, tt, "Row 0")
				frame, _ := lookColors(look.dark)
				if got := devicePx(img, box.X, int(row.Y+row.H/2), 1); !sameColor(got, frame) {
					t.Fatalf("the left edge beside the chosen row is %v, want the frame's %v", got, frame)
				}
			})
		}
	}
}

// scrollColumn counts the DIPs of height of the column in DIPs inside the
// right edge of w's list (the pixel that ends there) that differ from the
// content background, with the pointer away from it. A ListBox's thumb ends
// 3 DIPs inside; a table's, whose rows scroll inside its frame, 4.
func scrollColumn(t *testing.T, w Control, dark bool, in float32) int {
	t.Helper()
	tt := tester(t, w)
	tt.SetDark(dark)
	tt.Move(790, 590)
	tt.Frame()
	img, box := tt.Image(), listBox(t, tt)
	_, bg := lookColors(dark)
	x, n := box.X+box.W-in-1, 0
	for y := int(box.Y) + 2; y < int(box.Y+box.H)-2; y++ {
		if !sameColor(devicePx(img, x, y, 1), bg) {
			n++
		}
	}
	return n
}

// listBoxOf is a ListBox of n rows at 300 by 120, in a box: alone in the
// window it would grow to the window's height.
func listBoxOf(n int) Control { return listControls(n, Style{})["ListBox"]() }

// A ListBox whose rows overflow shows its scroll bar with the pointer away.
func TestListScrollBarShowsWhenRowsOverflow(t *testing.T) {
	for _, look := range frameLooks {
		t.Run(look.name, func(t *testing.T) {
			useLook(t, look.skin)
			if n := scrollColumn(t, listBoxOf(20), look.dark, 3); n < 20 {
				t.Fatalf("%d DIPs of the scroll bar's column differ from the row background, want at least 20", n)
			}
		})
	}
}

// A ListBox whose rows fit shows no scroll bar.
func TestListScrollBarHiddenWhenRowsFit(t *testing.T) {
	for _, look := range frameLooks {
		t.Run(look.name, func(t *testing.T) {
			useLook(t, look.skin)
			if n := scrollColumn(t, listBoxOf(3), look.dark, 3); n != 0 {
				t.Fatalf("%d DIPs of the scroll bar's column differ from the row background, want none", n)
			}
		})
	}
}

// Every list control shows its scroll bar with the pointer away while its rows
// overflow: the column 5 DIPs inside its right edge, in the thumb of a list's
// and a table's alike, differs from the column 12 DIPs left of it (rows,
// stripes and a header are the same across) over at least 20 DIPs with 20
// rows, and nowhere with 1 (3 rows overflow a table's 120 under its header).
// The large icons reach those columns, and the List view stands for the grid
// of the icon views there. The thumb is shorter than 20 DIPs within 8 DIPs
// of the ends, so the count starts and stops 2 DIPs inside.
func TestListScrollBarShowsEveryControl(t *testing.T) {
	// thumb counts the DIPs of height where the two columns differ.
	thumb := func(t *testing.T, w Control, dark bool) int {
		t.Helper()
		tt := tester(t, w)
		tt.SetDark(dark)
		tt.Move(790, 590)
		tt.Frame()
		img, box := tt.Image(), listBox(t, tt)
		x, n := box.X+box.W-6, 0
		for y := int(box.Y) + 2; y < int(box.Y+box.H)-2; y++ {
			if !sameColor(devicePx(img, x, y, 1), devicePx(img, x-12, y, 1)) {
				n++
			}
		}
		return n
	}
	for _, look := range frameLooks {
		few, many := listControls(1, Style{}), listControls(20, Style{})
		for name := range few {
			if name == "ListView LargeIcon" {
				continue
			}
			t.Run(look.name+"/"+name, func(t *testing.T) {
				useLook(t, look.skin)
				if fit, over := thumb(t, few[name](), look.dark), thumb(t, many[name](), look.dark); fit != 0 || over < 20 {
					t.Fatalf("the thumb's column differs over %d DIPs with 20 rows and %d with 1, want at least 20 and 0", over, fit)
				}
			})
		}
	}
}
