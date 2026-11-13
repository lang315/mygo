package skin

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Every built-in skin passes the contrast floors in both appearances.
func TestCalmPassesContrast(t *testing.T) {
	if err := check(&Calm.light, false); err != nil {
		t.Errorf("Calm light: %v", err)
	}
	if err := check(&Calm.dark, true); err != nil {
		t.Errorf("Calm dark: %v", err)
	}
	// The row pairs the theme draws (RowSelectedText is Text), measured
	// here too, over Content as check measures them.
	for _, tk := range []*skinhook.Tokens{&Calm.light, &Calm.dark} {
		sel, hover := tk.RowSelected.Over(tk.Content), tk.RowHover.Over(tk.Content)
		if r := contrastOf(tk.Text, sel); r < 4.5 {
			t.Errorf("Text on RowSelected is %.2f:1", r)
		}
		if r := contrastOf(tk.Text2, hover); r < 4.5 {
			t.Errorf("Text2 on RowHover is %.2f:1", r)
		}
	}
}

// check measures the rows' text: Text on RowSelected and Text2 on RowHover,
// each named when it fails.
func TestCheckRowPairs(t *testing.T) {
	tk := Calm.light
	tk.RowSelected = ui.Hex("#5a5f66")
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Text on RowSelected") {
		t.Errorf("a dark RowSelected under Text gives %v, want an error naming Text on RowSelected", err)
	}
	tk = Calm.light
	tk.RowHover = ui.Hex("#a0a4aa")
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Text2 on RowHover") {
		t.Errorf("a gray RowHover under Text2 gives %v, want an error naming Text2 on RowHover", err)
	}
}

// The error names the pair that fails, its ratio and the floor.
func TestCheckNamesTheFailingPair(t *testing.T) {
	tk := Calm.light
	tk.Text2 = ui.Hex("#9aa0a6")
	err := check(&tk, false)
	if err == nil {
		t.Fatal("Text2 #9aa0a6 passes the check, want an error")
	}
	if !strings.Contains(err.Error(), "Text2 on Content") || !strings.Contains(err.Error(), "want 4.5:1") {
		t.Errorf("the error is %q, want it to name Text2 on Content and the 4.5:1 floor", err)
	}
}

// A translucent fill is measured blended over the layer it sits on: an
// opaque bright green under light green text fails, the 14% one passes.
func TestCheckBlendsTranslucentFills(t *testing.T) {
	tk := Calm.dark
	tk.Success.Subtle = ui.Hex("#34c759")
	err := check(&tk, true)
	if err == nil || !strings.Contains(err.Error(), "Success Text on Subtle") {
		t.Errorf("an opaque #34c759 under #7fd99a gives %v, want an error naming Success Text on Subtle", err)
	}
	if err := check(&Calm.dark, true); err != nil {
		t.Errorf("Calm dark, with its translucent fills: %v", err)
	}
	// A translucent foreground blends over its layer too: the field outline
	// at half its alpha is too faint.
	tk = Calm.light
	tk.FieldLine = tk.FieldLine.Alpha(0.5)
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "FieldLine on Content") {
		t.Errorf("FieldLine at 50%% alpha gives %v, want an error naming FieldLine on Content", err)
	}
}

// The outlines of fields and the accent line have their own floor, 3:1, not
// text's: an outline under 3:1 fails, one over it passes.
func TestCheckOutlineFloor(t *testing.T) {
	tk := Calm.light
	tk.FieldLine = ui.Hex("#9aa0a6") // about 2.6:1 on white
	err := check(&tk, false)
	if err == nil || !strings.Contains(err.Error(), "FieldLine on Content") || !strings.Contains(err.Error(), "want 3:1") {
		t.Errorf("FieldLine #9aa0a6 gives %v, want an error naming FieldLine on Content and 3:1", err)
	}
	tk.FieldLine = ui.Hex("#80868f") // about 3.6:1 on white and on the chrome, under 4.5:1
	if err := check(&tk, false); err != nil {
		t.Errorf("FieldLine #80868f, over 3:1: %v", err)
	}
}

// The floors are 4.5:1 and 3:1 exactly: a pair just under one fails, and a
// pair just over it passes.
func TestCheckFloorsAreExact(t *testing.T) {
	white := ui.Hex("#ffffff")
	for _, c := range []struct {
		name        string
		under, over ui.Color
		floor       float64
		set         func(*skinhook.Tokens, ui.Color)
	}{
		{"Text3 on Control", ui.Hex("#777777"), ui.Hex("#767676"), 4.5, func(t *skinhook.Tokens, c ui.Color) { t.Text3 = c }},
		{"FieldLine on Content", ui.Hex("#959595"), ui.Hex("#939393"), 3, func(t *skinhook.Tokens, c ui.Color) { t.FieldLine, t.Chrome = c, white }},
	} {
		if r := contrastOf(c.under, white); r >= c.floor || r < c.floor-0.1 {
			t.Fatalf("%s: the color under is %.3f:1, not just under %g:1", c.name, r, c.floor)
		}
		if r := contrastOf(c.over, white); r < c.floor || r > c.floor+0.1 {
			t.Fatalf("%s: the color over is %.3f:1, not just over %g:1", c.name, r, c.floor)
		}
		tk := Calm.light
		c.set(&tk, c.under)
		if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), c.name) {
			t.Errorf("%s just under %g:1 gives %v, want an error naming it", c.name, c.floor, err)
		}
		tk = Calm.light
		c.set(&tk, c.over)
		if err := check(&tk, false); err != nil {
			t.Errorf("%s just over %g:1: %v", c.name, c.floor, err)
		}
	}
}

// The accent line must stand out from the chrome it sits on, 3:1, in both
// appearances.
func TestCheckAccentOnChrome(t *testing.T) {
	for _, c := range []struct {
		name   string
		tk     skinhook.Tokens
		dark   bool
		accent ui.Color
	}{
		{"light", Calm.light, false, ui.Hex("#c9d7ee")},
		{"dark", Calm.dark, true, ui.Hex("#2a3550")},
	} {
		c.tk.Accent.Base = c.accent
		err := check(&c.tk, c.dark)
		if err == nil || !strings.Contains(err.Error(), "Accent.Base on Chrome") || !strings.Contains(err.Error(), "want 3:1") {
			t.Errorf("%s: Accent.Base %v gives %v, want an error naming Accent.Base on Chrome and 3:1", c.name, c.accent, err)
		}
	}
}

// Calm's Info, the running state, is its accent, in both appearances.
func TestCalmInfoIsItsAccent(t *testing.T) {
	if Calm.light.Info != Calm.light.Accent {
		t.Errorf("light Info is %v, want the accent %v", Calm.light.Info, Calm.light.Accent)
	}
	if Calm.dark.Info != Calm.dark.Accent {
		t.Errorf("dark Info is %v, want the accent %v", Calm.dark.Info, Calm.dark.Accent)
	}
}

// The Info family is measured as the others are: its text on its subtle
// fill.
func TestCheckInfoFamily(t *testing.T) {
	tk := Calm.light
	tk.Info.Subtle = tk.Info.Text
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Info Text on Subtle") {
		t.Errorf("Info text on itself gives %v, want an error naming Info Text on Subtle", err)
	}
}

// The state-distance error names the pair, the eye it fails for and the
// floor of the appearance.
func TestCheckNamesStateDistance(t *testing.T) {
	tk := Calm.light
	tk.Info.Text = tk.Success.Text
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Info Text near Success Text (normal)") {
		t.Errorf("Info Text = Success Text gives %v, want an error naming Info Text near Success Text (normal)", err)
	}
	tk = Calm.dark
	tk.Info.Text = ui.Hex("#7fdccf") // teal's own dark text
	err := check(&tk, true)
	if err == nil || !strings.Contains(err.Error(), "Info Text near Success Text") || !strings.Contains(err.Error(), "want 35") {
		t.Errorf("dark Info Text #7fdccf gives %v, want an error naming Info Text near Success Text and want 35", err)
	}
}

// The error names the eye that sees the pair nearest: so a vision left out
// of the check changes the name.
func TestStateDistanceNamesTheNearestEye(t *testing.T) {
	// Teal's light text is nearest green for a deuteranope, 28 against 31
	// for normal vision.
	tk := Calm.light
	tk.Info.Text = ui.Hex("#0b5d56")
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Info Text near Success Text (deuteranopia): ΔE 28, want 45") {
		t.Errorf("light Info Text #0b5d56 gives %v, want Info Text near Success Text (deuteranopia): ΔE 28, want 45", err)
	}
	// Two colors that differ only in red are nearest for a protanope, who
	// sees red least.
	tk = Calm.light
	tk.Info.Text = ui.Hex("#d8261b")
	if err := check(&tk, false); err == nil || !strings.Contains(err.Error(), "Info Text near Danger Text (protanopia)") {
		t.Errorf("light Info Text #d8261b gives %v, want an error naming Info Text near Danger Text (protanopia)", err)
	}
}

// Translucent text is measured blended over the content, as contrast is:
// green at 5% alpha is almost white, far from green, though its own RGB is
// green's.
func TestStateDistanceBlendsTranslucentText(t *testing.T) {
	tk := Calm.light
	tk.Info.Text = tk.Success.Text.Alpha(0.05)
	if d := deltaE(tk.Info.Text.Over(tk.Content), tk.Success.Text, nil); d < 60 {
		t.Fatalf("green at 5%% over white is ΔE %.1f from green, want a clear pass", d)
	}
	if err := stateDistance(&tk, false); err != nil {
		t.Errorf("green at 5%% alpha: %v", err)
	}
}

// Calm passes the state-distance floors; plum, 40 from green, fails the
// light one.
func TestStateDistanceFloors(t *testing.T) {
	if err := stateDistance(&Calm.light, false); err != nil {
		t.Errorf("Calm light: %v", err)
	}
	if err := stateDistance(&Calm.dark, true); err != nil {
		t.Errorf("Calm dark: %v", err)
	}
	tk := Calm.light
	tk.Info.Text = ui.Hex("#7a2569")
	if err := stateDistance(&tk, false); err == nil || !strings.Contains(err.Error(), "want 45") {
		t.Errorf("plum #7a2569 gives %v, want an error with want 45", err)
	}
}

// The floors are 45 light and 35 dark exactly: grays look almost the same to
// every eye, so their ΔE is about their lightness apart, and a gray just under the floor
// from black fails while one just over passes.
func TestStateDistanceFloorsAreExact(t *testing.T) {
	black := ui.Hex("#000000")
	for _, c := range []struct {
		dark        bool
		floor       float64
		under, over ui.Color
	}{
		{false, 45, ui.Hex("#6a6a6a"), ui.Hex("#6b6b6b")},
		{true, 35, ui.Hex("#525252"), ui.Hex("#535353")},
	} {
		for _, v := range visions {
			if d := deltaE(c.under, black, v.m); d >= c.floor || d < c.floor-1 {
				t.Fatalf("dark=%v: the gray under is ΔE %.2f for %s, not just under %g", c.dark, d, v.name, c.floor)
			}
			if d := deltaE(c.over, black, v.m); d < c.floor || d > c.floor+1 {
				t.Fatalf("dark=%v: the gray over is ΔE %.2f for %s, not just over %g", c.dark, d, v.name, c.floor)
			}
		}
		tk := Calm.light
		tk.Success.Text, tk.Warning.Text, tk.Danger.Text = black, black, black
		tk.Info.Text = c.under
		if err := stateDistance(&tk, c.dark); err == nil || !strings.Contains(err.Error(), "Info Text near Success Text") {
			t.Errorf("dark=%v: a gray just under %g gives %v, want an error naming it", c.dark, c.floor, err)
		}
		tk.Info.Text = c.over
		if err := stateDistance(&tk, c.dark); err != nil {
			t.Errorf("dark=%v: a gray just over %g: %v", c.dark, c.floor, err)
		}
	}
}

// The ΔE of teal's text and green is the orchestrator's: 31 for normal
// vision, 28 for a deuteranope.
func TestDeltaEMatchesReference(t *testing.T) {
	want := map[string]float64{"normal": 31, "deuteranopia": 28}
	for _, v := range visions {
		w, ok := want[v.name]
		if !ok {
			continue
		}
		if d := deltaE(ui.Hex("#1d6b31"), ui.Hex("#0b5d56"), v.m); math.Abs(d-w) > 1 {
			t.Errorf("%s: ΔE %.2f, want %g ± 1", v.name, d, w)
		}
	}
	if len(visions) != 3 {
		t.Errorf("%d visions, want normal, deuteranopia and protanopia", len(visions))
	}
	// CIELAB (D65) of sRGB's white, red and blue, the published values: the
	// ΔE of one pair alone could match by chance.
	for _, c := range []struct {
		hex  string
		want [3]float64
	}{
		{"#ffffff", [3]float64{100, 0, 0}},
		{"#ff0000", [3]float64{53.24, 80.09, 67.20}},
		{"#0000ff", [3]float64{32.30, 79.19, -107.86}},
	} {
		got := labOf(ui.Hex(c.hex), nil)
		for i := range got {
			if math.Abs(got[i]-c.want[i]) > 0.05 {
				t.Errorf("%s is Lab %.2f, want %.2f", c.hex, got, c.want)
				break
			}
		}
	}
}

// Each chart color must stand out from the content, 3:1: the old amber
// #ca8a04 (2.94:1 on white) fails at any index and is named by it; Calm's
// light and dark pass, each color measured on its own content.
func TestCheckChartOnContent(t *testing.T) {
	for _, i := range []int{0, 6, 7} {
		tk := Calm.light
		tk.Chart[i] = ui.Hex("#ca8a04")
		err := check(&tk, false)
		name := fmt.Sprintf("Chart[%d] on Content", i)
		if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "want 3:1") {
			t.Errorf("Chart[%d] #ca8a04 gives %v, want an error naming %s and 3:1", i, err, name)
		}
	}
	// In dark, a color as dark as a line fails on the dark content.
	tk := Calm.dark
	tk.Chart[3] = ui.Hex("#2a2e34")
	if err := check(&tk, true); err == nil || !strings.Contains(err.Error(), "Chart[3] on Content") {
		t.Errorf("dark Chart[3] #2a2e34 gives %v, want an error naming Chart[3] on Content", err)
	}
	for _, p := range palettes {
		s := Calm.WithPalette(p)
		if err := check(&s.light, false); err != nil {
			t.Errorf("%s light: %v", p.Name(), err)
		}
		if err := check(&s.dark, true); err != nil {
			t.Errorf("%s dark: %v", p.Name(), err)
		}
	}
}

// Every failing pair is named, not only the first.
func TestCheckNamesEveryFailingPair(t *testing.T) {
	tk := Calm.light
	tk.Text2 = ui.Hex("#9aa0a6")
	tk.Danger.On = tk.Danger.Base
	err := check(&tk, false)
	for _, want := range []string{"Text2 on Content", "Text2 on Chrome", "Danger On on Base"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("the error is %v, want it to name %s", err, want)
		}
	}
}
