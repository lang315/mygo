package widget

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A wide row whose control is wider than the card's inside (a picture of 515)
// leaves the other rows of the card their sizes: it overflows, or is clipped,
// on its own, and does not push the label column, which shrinks the controls.

// woPic is a picture w by h DIPs.
func woPic(w, h int) *PictureBoxWidget {
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 255, 255}), image.Point{}, draw.Src)
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return PictureBox(b.Bytes(), float32(w), float32(h)).SetAccessibleName("pic")
}

// woForms are the rows' last row: none, a wide label, a wide picture of the
// size given, and that picture in a box under a label.
var woForms = map[string]func(f *FieldsWidget, w, h int) *FieldsWidget{
	"none":  func(f *FieldsWidget, w, h int) *FieldsWidget { return f },
	"label": func(f *FieldsWidget, w, h int) *FieldsWidget { return f.Row("", Label("A note")).Wide() },
	"pic":   func(f *FieldsWidget, w, h int) *FieldsWidget { return f.Row("", woPic(w, h)).Wide() },
	"vbox": func(f *FieldsWidget, w, h int) *FieldsWidget {
		return f.Row("", VBox(0).Add(Label("Guide:"), woPic(w, h))).Wide()
	},
}

// woForm shows the form of kind, its wide picture w by h, in a window width
// wide, settled, with or without Calm.
func woForm(t *testing.T, kind string, w, h, width int, calm bool) *Tester {
	t.Helper()
	useCalm(t, calm)
	t.Cleanup(calmOff)
	f := Fields().SettingsRows().Gap(10, 16).Section("Email server").
		Row("Use email server", Switch()).
		Row("Domain", TextBox().SetSize(280, 0))
	tt := NewTester(t, woForms[kind](f, w, h), width, 800)
	tt.Settle()
	return tt
}

func woEach(t *testing.T, test func(t *testing.T, width int, calm bool)) {
	for _, calm := range []bool{false, true} {
		for _, width := range []int{560, 760} {
			name := "noskin"
			if calm {
				name = "calm"
			}
			t.Run(name+"/"+map[int]string{560: "560", 760: "760"}[width], func(t *testing.T) { test(t, width, calm) })
		}
	}
}

// Cases 1 and 2: the text box is 280 wide, and the switch and the box keep
// the place they have with no wide row.
func TestWideOverflowKeepsTheOtherRows(t *testing.T) {
	woEach(t, func(t *testing.T, width int, calm bool) {
		base := woForm(t, "none", 515, 203, width, calm)
		bt, bs := base.Find(ByRole(ui.RoleTextField)).Bounds(), base.Find(ByRole(ui.RoleSwitch)).Bounds()
		for _, kind := range []string{"pic", "vbox"} {
			tt := woForm(t, kind, 515, 203, width, calm)
			tb, sw := tt.Find(ByRole(ui.RoleTextField)).Bounds(), tt.Find(ByRole(ui.RoleSwitch)).Bounds()
			if !approx(tb.W, 280) {
				t.Errorf("%s: the text box is %v wide, want 280", kind, tb.W)
			}
			if !approx(tb.X, bt.X) || !approx(tb.W, bt.W) || !approx(sw.X, bs.X) || !approx(sw.W, bs.W) {
				t.Errorf("%s: the text box is at %v wide %v and the switch at %v wide %v, with no wide row at %v, %v and %v, %v",
					kind, tb.X, tb.W, sw.X, sw.W, bt.X, bt.W, bs.X, bs.W)
			}
		}
	})
}

// Case 3: the picture starts at the card's inside left, as a wide label does,
// and the card is no wider than with the label (the switch ends its inside).
func TestWideOverflowPictureStartsAtTheCardsLeft(t *testing.T) {
	woEach(t, func(t *testing.T, width int, calm bool) {
		label := woForm(t, "label", 515, 203, width, calm)
		ln := find(t, label.Tester, "A note")
		ls := label.Find(ByRole(ui.RoleSwitch)).Bounds()
		for _, kind := range []string{"pic", "vbox"} {
			tt := woForm(t, kind, 515, 203, width, calm)
			sw := tt.Find(ByRole(ui.RoleSwitch)).Bounds()
			if !approx(sw.X+sw.W, ls.X+ls.W) {
				t.Errorf("%s: the card's inside ends at %v, with a wide label at %v", kind, sw.X+sw.W, ls.X+ls.W)
			}
			var left float32
			if kind == "pic" {
				left = tt.Find(ByLabel("pic")).Bounds().X
			} else {
				left = find(t, tt.Tester, "Guide:").X
			}
			if !approx(left, ln.X) {
				t.Errorf("%s starts at x %v, a wide label at %v", kind, left, ln.X)
			}
		}
	})
}

// Case 5: a plain grid and stacked rows keep the text box's width too.
func TestWideOverflowGridAndStacked(t *testing.T) {
	for name, lay := range map[string]func(f *FieldsWidget) *FieldsWidget{
		"grid":    func(f *FieldsWidget) *FieldsWidget { return f },
		"stacked": func(f *FieldsWidget) *FieldsWidget { return f.Stacked() },
	} {
		for _, width := range []int{560, 760} {
			t.Run(name, func(t *testing.T) {
				box := func(wide bool) float32 {
					f := lay(Fields().Gap(10, 16)).Row("Domain", TextBox().SetSize(280, 0))
					if wide {
						f.Row("", woPic(515, 203)).Wide()
					}
					tt := NewTester(t, f, width, 800)
					tt.Settle()
					return tt.Find(ByRole(ui.RoleTextField)).Bounds().W
				}
				if a, b := box(true), box(false); !approx(a, b) {
					t.Errorf("width %d: the text box is %v wide with a wide picture, %v without", width, a, b)
				}
			})
		}
	}
}

// Case 4: a picture narrower than the card keeps its size.
func TestWideOverflowNarrowPictureKeepsItsSize(t *testing.T) {
	woEach(t, func(t *testing.T, width int, calm bool) {
		tt := woForm(t, "pic", 200, 80, width, calm)
		if p := tt.Find(ByLabel("pic")).Bounds(); !approx(p.W, 200) || !approx(p.H, 80) {
			t.Errorf("the picture is %v by %v, want 200 by 80", p.W, p.H)
		}
		if tb := tt.Find(ByRole(ui.RoleTextField)).Bounds(); !approx(tb.W, 280) {
			t.Errorf("the text box is %v wide, want 280", tb.W)
		}
	})
}
