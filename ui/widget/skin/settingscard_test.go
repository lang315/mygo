package skin

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// cardForm is SettingsRows of two sections of one row each: no divider, so
// the column x 200 crosses only the cards' edges.
func cardForm() *widget.FieldsWidget {
	return widget.Fields().SettingsRows().
		Section("Alpha").Row("One", widget.Switch()).
		Section("Beta").Row("Two", widget.Switch())
}

// edges returns the y and thickness of each line of color c in the column
// x 200: the top and bottom of each card.
func edges(tt *widget.Tester, c ui.Color) (ys, ws []int) {
	img := tt.Image()
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		if dist(colorAt(img, 200, y), c) > 8 {
			continue
		}
		if n := len(ys); n > 0 && ys[n-1]+ws[n-1] == y {
			ws[n-1]++
			continue
		}
		ys, ws = append(ys, y), append(ws, 1)
	}
	return ys, ws
}

// pageOf is the color at the window's bottom left, below the form.
func pageOf(tt *widget.Tester) ui.Color {
	img := tt.Image()
	return colorAt(img, 2, img.Bounds().Max.Y-3)
}

// P2b case 12: HighContrast and Classic, light and dark, pass Check, and
// their settings cards are framed as their Card look says: HighContrast's
// border-2 border-strong, Classic's none (a line of 1 of Border, rounded by
// its RadiusCard of 3).
func TestSettingsCardsFramedByTheCardLook(t *testing.T) {
	for _, c := range []struct {
		s     *Skin
		width int
		round bool // 3.5 down the left edge is outside the corner
	}{
		{HighContrast, 2, true},
		{Classic, 1, false},
	} {
		if err := c.s.Check(); err != nil {
			t.Errorf("%s: Check: %v", c.s.Name(), err)
		}
		for _, dark := range []bool{false, true} {
			t.Run(c.s.Name()+"-"+appearance(dark), func(t *testing.T) {
				tt := showSkin(t, cardForm(), 400, 400, c.s, dark, 1)
				tk := c.s.Tokens(dark)
				line := tk.Border
				if c.width == 2 {
					line = tk.BorderStrong
				}
				ys, ws := edges(tt, line)
				if len(ys) != 4 {
					t.Fatalf("%d edges of %v at %v, want the top and bottom of 2 cards", len(ys), line, ys)
				}
				for i, w := range ws {
					if w != c.width {
						t.Errorf("edge %d at %d is %d high, want %d", i, ys[i], w, c.width)
					}
				}
				if got := colorAt(tt.Image(), 0, ys[0]+3) == pageOf(tt); got != c.round {
					t.Errorf("3 down the first card's left edge is the page: %v, want %v", got, c.round)
				}
			})
		}
	}
}

// P2b cases 2 and 6, under Calm: the cards are rounded by its RadiusCard,
// 10, and a card of one row is that row, at least SettingsRowHeight high,
// and the padding and lines outside it.
func TestSettingsCardsUnderCalm(t *testing.T) {
	tt := showSkin(t, cardForm(), 400, 400, Calm, false, 1)
	tk := Calm.Tokens(false)
	ys, _ := edges(tt, tk.Border)
	if len(ys) != 4 {
		t.Fatalf("%d edges at %v, want the top and bottom of 2 cards", len(ys), ys)
	}
	if colorAt(tt.Image(), 0, ys[0]+3) != pageOf(tt) {
		t.Error("3 down the first card's left edge is not the page: not rounded by RadiusCard 10")
	}
	space := ui.LightTheme().Space(1)
	if h, want := float32(ys[1]+1-ys[0]), 1+space+tk.SettingsRowHeight+space+1; h < want-0.5 || h > want+0.5 {
		t.Errorf("a card of one row is %v high, want %v: SettingsRowHeight %v, the padding outside it", h, want, tk.SettingsRowHeight)
	}
}

// P2b case 15: a skin switched at run time, Calm to Classic, draws Classic's
// cards on the next frame.
func TestSettingsCardsFollowUse(t *testing.T) {
	tt := showSkin(t, cardForm(), 400, 400, Calm, false, 1)
	ys, _ := edges(tt, Calm.Tokens(false).Border)
	if len(ys) != 4 || colorAt(tt.Image(), 0, ys[0]+3) != pageOf(tt) {
		t.Fatalf("under Calm: edges %v, want 4 and round corners", ys)
	}
	Use(Classic)
	tt.Frame()
	ys, _ = edges(tt, Classic.Tokens(false).Border)
	if len(ys) != 4 {
		t.Fatalf("under Classic: %d edges of its Border at %v, want 4", len(ys), ys)
	}
	if colorAt(tt.Image(), 0, ys[0]+3) == pageOf(tt) {
		t.Error("under Classic, 3 down the card's left edge is the page: Calm's radius stays")
	}
}
