package skin

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
	"github.com/egoist/mygo/ui/widget/tw"
)

// allLooks sets every field, each to a class of its own.
var allLooks = Looks{
	Button: "bg-red-600", ToolButton: "bg-orange-600", Field: "bg-amber-600", ComboBox: "bg-yellow-600",
	CheckBox: "bg-lime-600", RadioButton: "bg-green-600", GroupBox: "bg-emerald-600", Card: "bg-teal-600",
	TabStrip: "bg-cyan-600", TabHeader: "bg-sky-600", GridHeader: "bg-blue-600", ToolStrip: "bg-indigo-600",
	StatusStrip: "bg-violet-600", MenuBar: "bg-purple-600", HeaderBar: "bg-fuchsia-600",
}

// WithLooks keeps the strings as given, Calm has none, and the receiver
// keeps its own.
func TestWithLooksKeepsStrings(t *testing.T) {
	s := Calm.WithLooks(allLooks)
	if got := s.Looks(); got != allLooks {
		t.Errorf("Looks() = %+v, want %+v", got, allLooks)
	}
	if got := Calm.Looks(); got != (Looks{}) {
		t.Errorf("Calm.Looks() = %+v, want none", got)
	}
	if s == Calm {
		t.Error("WithLooks returned its receiver, want a copy")
	}
	if s.Name() != "Calm" || s.Tokens(false) != Calm.Tokens(false) || s.Tokens(true) != Calm.Tokens(true) {
		t.Error("WithLooks changed the name or the tokens")
	}
}

// A second WithLooks replaces the looks whole: the kinds it leaves "" go.
func TestWithLooksReplacesKind(t *testing.T) {
	s := Calm.WithLooks(Looks{Button: "bg-red-600"}).WithLooks(Looks{Card: "rounded-none"})
	if got := s.Looks(); got.Button != "" || got.Card != "rounded-none" {
		t.Errorf("Looks() = %+v, want only Card", got)
	}
	t.Cleanup(func() { Use(nil) })
	Use(s)
	if skinhook.Look(skinhook.KindButton) != nil {
		t.Error("the Button look of the first WithLooks is still on")
	}
	if skinhook.Look(skinhook.KindCard) == nil {
		t.Error("no Card look after Use")
	}
}

// Use hands every look to widget, each a *widget.Style; Use(Calm), which
// has none, and Use(nil) take them off.
func TestUseSetsLooks(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	Use(Calm.WithLooks(allLooks))
	for k := range skinhook.NumKinds {
		if _, ok := skinhook.Look(k).(*widget.Style); !ok {
			t.Errorf("kind %d: look %T after Use, want a *widget.Style", k, skinhook.Look(k))
		}
	}
	if l := skinhook.Look(skinhook.NumKinds); l != nil {
		t.Errorf("a kind past the last has look %v while looks are on", l)
	}
	Use(Calm)
	for k := range skinhook.NumKinds {
		if l := skinhook.Look(k); l != nil {
			t.Errorf("kind %d: look %v after Use(Calm), want none", k, l)
		}
	}
	Use(Calm.WithLooks(allLooks).WithDensity(Compact))
	if skinhook.Look(skinhook.KindButton) == nil {
		t.Error("no Button look after Use of a compact skin with looks")
	}
	Use(nil)
	for k := range skinhook.NumKinds {
		if l := skinhook.Look(k); l != nil {
			t.Errorf("kind %d: look %v after Use(nil), want none", k, l)
		}
	}
	if l := skinhook.Look(skinhook.NumKinds); l != nil {
		t.Errorf("a kind past the last has look %v", l)
	}
}

// Looks of only blanks are no looks.
func TestBlankLooksAreNone(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	Use(Calm.WithLooks(Looks{Button: "  "}))
	if l := skinhook.Look(skinhook.KindButton); l != nil {
		t.Errorf("a blank Button look is %v, want none", l)
	}
}

// Palettes, accents and densities keep the looks.
func TestLooksSurviveTheOtherWiths(t *testing.T) {
	s := Calm.WithLooks(allLooks).WithPalette(Teal).WithDensity(Compact).WithAccent(ui.Hex("#7c3aed"))
	if s.Looks() != allLooks {
		t.Errorf("Looks() = %+v after the other Withs, want %+v", s.Looks(), allLooks)
	}
}

// The looks the docs show are classes tw knows.
func TestDocLooksParse(t *testing.T) {
	for _, cl := range []string{
		"rounded-none border border-control",
		"rounded-t border border-b-0 selected:bg-surface",
		"rounded-sm border border-button",
		"rounded-t border border-b-0 bg-chrome selected:bg-surface",
		"bg-chrome border-r border-strong",
	} {
		if _, err := tw.Parse(cl); err != nil {
			t.Errorf("%q: %v", cl, err)
		}
	}
}

// A class tw does not know is skipped, the rest of the look applies, and the
// skin still works.
func TestBadClassSkipped(t *testing.T) {
	red := ui.Hex("#dc2626") // red-600
	s := Calm.WithLooks(Looks{Button: "bg-nonsense bg-red-600"})
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			tt := showSkin(t, widget.Button("Go"), 300, 120, s, dark, 1)
			b := tt.Find(widget.ByLabel("Go")).Bounds()
			if got := colorAt(tt.Image(), int(b.X)+4, int(b.Y+b.H/2)); got != red {
				t.Errorf("the button paints %v, want red-600 %v", got, red)
			}
		})
	}
}
