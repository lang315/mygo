package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

var (
	red  = ui.RGB(220, 0, 0)
	blue = ui.RGB(0, 0, 220)
)

func TestStyleSidesMergePerSide(t *testing.T) {
	a := NewStyle().Padding(4)
	b := NewStyle().PaddingTop(1)
	m := a.Merge(b)
	if m.pad != [4]float32{1, 4, 4, 4} || m.padSet != 0xf {
		t.Fatalf("pad %v set %b, want [1 4 4 4] 1111", m.pad, m.padSet)
	}
}

func TestStyleCSSSideCounts(t *testing.T) {
	s := NewStyle().Padding(1, 2)
	if s.pad != [4]float32{1, 2, 1, 2} {
		t.Fatalf("Padding(1, 2) = %v", s.pad)
	}
	s = NewStyle().Radius(1, 2, 3) // a bad count changes nothing
	if s.radiusSet != 0 {
		t.Fatalf("Radius(1, 2, 3) set %b, want nothing", s.radiusSet)
	}
}

func TestStyleClamps(t *testing.T) {
	s := NewStyle().Opacity(2).Padding(-3).Blur(-1).Radius(-2).BorderWidth(-1).Width(-5)
	if s.opacity != 1 || s.pad[0] != 0 || s.blur != 0 || s.radius[0] != 0 || s.border[0] != 0 || s.width != 0 {
		t.Fatalf("not clamped: %+v", s.props)
	}
	if m := NewStyle().Margin(-2); m.margin[0] != -2 {
		t.Fatalf("margin may be negative, as CSS: %v", m.margin)
	}
}

func TestStyleZeroColorIsUnset(t *testing.T) {
	s := NewStyle().Background(Color{}).TextColor(Color{}).Shadow(0, 1, 2, 0, Color{})
	if !s.empty() {
		t.Fatalf("zero colors set something: %+v", s.props)
	}
}

func TestStyleIsImmutable(t *testing.T) {
	shared := NewStyle().Shadow(0, 1, 2, 0, Fixed(red))
	a := shared.Shadow(0, 4, 8, 0, Fixed(blue))
	b := shared.Shadow(0, 2, 4, 0, Fixed(blue))
	if len(shared.shadows) != 1 || len(a.shadows) != 2 || len(b.shadows) != 2 || a.shadows[1].y != 4 || b.shadows[1].y != 2 {
		t.Fatalf("shadows aliased: shared %v a %v b %v", shared.shadows, a.shadows, b.shadows)
	}
	h := NewStyle().Background(Fixed(red))
	s := NewStyle().Hover(h)
	h = h.Background(Fixed(blue)) // changes the local copy only
	if s.v[vHover].background.fixed != red {
		t.Fatal("a variant changed after it was given")
	}
}

func TestStyleEffectiveOrder(t *testing.T) {
	s := NewStyle().Background(Fixed(ui.RGB(1, 1, 1))).
		Hover(NewStyle().Background(Fixed(ui.RGB(2, 2, 2)))).
		Focused(NewStyle().Background(Fixed(ui.RGB(3, 3, 3)))).
		Pressed(NewStyle().Background(Fixed(ui.RGB(4, 4, 4)))).
		Disabled(NewStyle().Background(Fixed(ui.RGB(5, 5, 5)))).
		Dark(NewStyle().Background(Fixed(ui.RGB(6, 6, 6))))
	for _, c := range []struct {
		dark, hover, focused, pressed, disabled bool
		want                                    uint8
	}{
		{false, false, false, false, false, 1},
		{true, false, false, false, false, 6},
		{true, true, false, false, false, 2},
		{false, true, true, false, false, 3},
		{false, true, true, true, false, 4},
		{true, true, true, true, true, 5},
	} {
		p := s.effective(c.dark, c.hover, c.focused, c.pressed, c.disabled)
		if got := p.background.fixed.R; got != c.want {
			t.Errorf("%+v: background %d, want %d", c, got, c.want)
		}
	}
}

func TestStyleDarkUnderAState(t *testing.T) {
	s := NewStyle().Hover(NewStyle().Background(Fixed(red)).Dark(NewStyle().Background(Fixed(blue))))
	if p := s.effective(true, true, false, false, false); p.background.fixed != blue {
		t.Fatalf("dark hover %v, want blue", p.background.fixed)
	}
	if p := s.effective(false, true, false, false, false); p.background.fixed != red {
		t.Fatalf("light hover %v, want red", p.background.fixed)
	}
}

func TestStyleMergeVariants(t *testing.T) {
	a := NewStyle().Hover(NewStyle().Background(Fixed(red)).Bold())
	b := NewStyle().Hover(NewStyle().Background(Fixed(blue)))
	m := a.Merge(b) // effective takes a pointer: it does not copy a Style each frame
	p := m.effective(false, true, false, false, false)
	if p.background.fixed != blue || p.set&pBold == 0 {
		t.Fatalf("merged hover: background %v bold %v", p.background.fixed, p.set&pBold != 0)
	}
}

func TestRoleResolvesFromTheTheme(t *testing.T) {
	l, d := ui.LightTheme(), ui.DarkTheme()
	if RoleSurface.resolve(l) != l.Surface || RoleSurface.resolve(d) != d.Surface || l.Surface == d.Surface {
		t.Fatal("RoleSurface does not follow the theme")
	}
	if Fixed(red).resolve(d) != red {
		t.Fatal("a fixed color changed with the theme")
	}
}

func TestCornerAndSideSetters(t *testing.T) {
	s := NewStyle().RadiusTopLeft(2).RadiusBottomRight(-1).BorderLeft(3).BorderTop(-2)
	if s.radiusSet != 0b0101 || s.radius != [4]float32{2, 0, 0, 0} {
		t.Fatalf("radius %v mask %04b", s.radius, s.radiusSet)
	}
	if s.borderSet != 0b1001 || s.border != [4]float32{0, 0, 0, 3} {
		t.Fatalf("border %v mask %04b", s.border, s.borderSet)
	}
}
