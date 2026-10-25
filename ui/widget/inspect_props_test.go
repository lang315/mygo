package widget

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Every property a Style can set has a line in the note, as does each side
// of the four-sided ones: a property added without one fails here.
func TestEveryPropertyOfAStyleIsDescribed(t *testing.T) {
	th := ui.LightTheme()
	for bit := prop(1); bit < pEnd; bit <<= 1 {
		p := props{set: bit}
		if bit == pShadows {
			p.shadows = []shadow{{x: 1}}
		}
		if got := p.describe(th, ""); len(got) == 0 {
			t.Errorf("property %#x is not described", uint32(bit))
		}
	}
	for i := range 4 {
		bit := uint8(1) << i
		p := props{padSet: bit, marginSet: bit, radiusSet: bit, borderSet: bit}
		got := p.describe(th, "")
		if len(got) != 4 {
			t.Errorf("side %d: %d lines for padding, margin, radius and border width, want 4: %v", i, len(got), got)
		}
		for _, f := range got {
			if !strings.Contains(f.Name, "-") {
				t.Errorf("side %d: %q does not say which side", i, f.Name)
			}
		}
	}
	all := props{padSet: 0b1111, pad: [4]float32{2, 2, 2, 2}}
	if got := all.describe(th, ""); len(got) != 1 || got[0].Name != "padding" {
		t.Errorf("four equal sides: %v, want one padding line", got)
	}
}

func describedValue(t *testing.T, fs []ui.InspectorField, name string) string {
	t.Helper()
	for _, f := range fs {
		if f.Name == name {
			return f.Value
		}
	}
	t.Fatalf("no %q line in %v", name, fs)
	return ""
}

func TestNoteNamesTheCursor(t *testing.T) {
	th := ui.LightTheme()
	got := NewStyle().Cursor(ui.CursorPointer).describe(th)
	if v := describedValue(t, got, "cursor"); v != "pointer" {
		t.Errorf("cursor: %q, want pointer", v)
	}
}

// An out-of-range Size styles as Base, as apply does, not as the last name.
func TestNoteShowsTheSizeApplyUses(t *testing.T) {
	th := ui.LightTheme()
	got := NewStyle().TextSize(Size(99)).describe(th)
	if v := describedValue(t, got, "font-size"); v != "base" {
		t.Errorf("font-size: %q, want base", v)
	}
}

func TestNoteMarksLayoutInAVariantAsIgnored(t *testing.T) {
	th := ui.LightTheme()
	got := NewStyle().HExpand().Hover(NewStyle().HExpand().Align(Center)).describe(th)
	if v := describedValue(t, got, "hexpand"); v != "true" {
		t.Errorf("hexpand in the base: %q, want true", v)
	}
	for _, name := range []string{"hover:hexpand", "hover:align"} {
		if v := describedValue(t, got, name); !strings.Contains(v, "(no effect in a variant)") {
			t.Errorf("%s: %q lacks the note", name, v)
		}
	}
}

func TestNoteGivesShadowsTheirUnits(t *testing.T) {
	th := ui.LightTheme()
	got := NewStyle().Shadow(1, 2, 3, 4, RoleText).describe(th)
	if v := describedValue(t, got, "shadow"); !strings.Contains(v, "x 1 DIPs, y 2 DIPs, blur 3 DIPs, spread 4 DIPs") {
		t.Errorf("shadow: %q", v)
	}
}

func TestNoteSaysWhenADisabledAncestorDisablesTheControl(t *testing.T) {
	th := ui.LightTheme()
	b := Button("x")
	for _, c := range []struct {
		own, inert bool
		want       string
	}{{false, false, "false"}, {false, true, "false (inherited: true)"}, {true, true, "true"}} {
		b.disabled = c.own
		if v := describedValue(t, b.core.inspect(b, th, c.inert), "disabled"); v != c.want {
			t.Errorf("own %v, inert %v: %q, want %q", c.own, c.inert, v, c.want)
		}
	}
}

// A Style made of two has no source, and one that is given none, or an empty
// one, has none.
func TestAMergedStyleHasNoSource(t *testing.T) {
	a, b := NewStyle().Padding(1).Source("p-1"), NewStyle().Margin(1).Source("m-1")
	if a.src == nil || *a.src != "p-1" {
		t.Fatalf("Source did not note the classes: %v", a.src)
	}
	if m := a.Merge(b); m.src != nil {
		t.Errorf("a merged Style has the source %q", *m.src)
	}
	if a.Source("").src != nil {
		t.Error("an empty source did not take the source off")
	}
	if NewStyle().Padding(1).src != nil {
		t.Error("a typed Style has a source")
	}
}

// Only a layer given to Apply shows classes, and only if it has them.
func TestOnlyAnAppliedStyleWithASourceShowsClasses(t *testing.T) {
	b := Button("x").Apply(NewStyle().Padding(1).Source("p-1"), NewStyle().Padding(2)).Bold()
	fs := b.inspect(b, ui.LightTheme(), false)
	n, at := 0, -1
	for i, f := range fs {
		if f.Name == "classes" {
			n++
			at = i
			if f.Value != "p-1" {
				t.Errorf("classes %q, want p-1", f.Value)
			}
		}
	}
	if n != 1 || fs[at-1].Name != "Apply, style 1 of 2" {
		t.Errorf("%d lines of classes, the first after %q: %v", n, fs[at-1].Name, fs)
	}
}
