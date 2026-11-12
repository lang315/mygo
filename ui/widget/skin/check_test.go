package skin

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// Every built-in skin passes the contrast floors in both appearances.
func TestCalmPassesContrast(t *testing.T) {
	if err := check(&Calm.light); err != nil {
		t.Errorf("Calm light: %v", err)
	}
	if err := check(&Calm.dark); err != nil {
		t.Errorf("Calm dark: %v", err)
	}
}

// The error names the pair that fails, its ratio and the floor.
func TestCheckNamesTheFailingPair(t *testing.T) {
	tk := Calm.light
	tk.Text2 = ui.Hex("#9aa0a6")
	err := check(&tk)
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
	err := check(&tk)
	if err == nil || !strings.Contains(err.Error(), "Success Text on Subtle") {
		t.Errorf("an opaque #34c759 under #7fd99a gives %v, want an error naming Success Text on Subtle", err)
	}
	if err := check(&Calm.dark); err != nil {
		t.Errorf("Calm dark, with its translucent fills: %v", err)
	}
	// A translucent foreground blends over its layer too: the field outline
	// at half its alpha is too faint.
	tk = Calm.light
	tk.FieldLine = tk.FieldLine.Alpha(0.5)
	if err := check(&tk); err == nil || !strings.Contains(err.Error(), "FieldLine on Content") {
		t.Errorf("FieldLine at 50%% alpha gives %v, want an error naming FieldLine on Content", err)
	}
}

// The outlines of fields and the accent line have their own floor, 3:1, not
// text's: an outline under 3:1 fails, one over it passes.
func TestCheckOutlineFloor(t *testing.T) {
	tk := Calm.light
	tk.FieldLine = ui.Hex("#9aa0a6") // about 2.6:1 on white
	err := check(&tk)
	if err == nil || !strings.Contains(err.Error(), "FieldLine on Content") || !strings.Contains(err.Error(), "want 3:1") {
		t.Errorf("FieldLine #9aa0a6 gives %v, want an error naming FieldLine on Content and 3:1", err)
	}
	tk.FieldLine = ui.Hex("#80868f") // about 3.6:1 on white and on the chrome, under 4.5:1
	if err := check(&tk); err != nil {
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
		if err := check(&tk); err == nil || !strings.Contains(err.Error(), c.name) {
			t.Errorf("%s just under %g:1 gives %v, want an error naming it", c.name, c.floor, err)
		}
		tk = Calm.light
		c.set(&tk, c.over)
		if err := check(&tk); err != nil {
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
		accent ui.Color
	}{
		{"light", Calm.light, ui.Hex("#c9d7ee")},
		{"dark", Calm.dark, ui.Hex("#2a3550")},
	} {
		c.tk.Accent.Base = c.accent
		err := check(&c.tk)
		if err == nil || !strings.Contains(err.Error(), "Accent.Base on Chrome") || !strings.Contains(err.Error(), "want 3:1") {
			t.Errorf("%s: Accent.Base %v gives %v, want an error naming Accent.Base on Chrome and 3:1", c.name, c.accent, err)
		}
	}
}

// Every failing pair is named, not only the first.
func TestCheckNamesEveryFailingPair(t *testing.T) {
	tk := Calm.light
	tk.Text2 = ui.Hex("#9aa0a6")
	tk.Danger.On = tk.Danger.Base
	err := check(&tk)
	for _, want := range []string{"Text2 on Content", "Text2 on Chrome", "Danger On on Base"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("the error is %v, want it to name %s", err, want)
		}
	}
}
