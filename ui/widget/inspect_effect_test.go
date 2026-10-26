package widget

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// setters are the Style methods the oracle draws from: single properties, the
// sides of the four-sided ones apart, and the layout ones.
var setters = []func(s Style, r *rand.Rand) Style{
	func(s Style, r *rand.Rand) Style { return s.Padding(float32(r.Intn(6))) },
	func(s Style, r *rand.Rand) Style { return s.PaddingTop(float32(r.Intn(6))) },
	func(s Style, r *rand.Rand) Style { return s.PaddingLeft(float32(r.Intn(6))) },
	func(s Style, r *rand.Rand) Style { return s.PaddingX(float32(r.Intn(6))) },
	func(s Style, r *rand.Rand) Style { return s.Margin(float32(r.Intn(6))) },
	func(s Style, r *rand.Rand) Style { return s.MarginBottom(float32(r.Intn(6))) },
	func(s Style, r *rand.Rand) Style { return s.Radius(float32(r.Intn(4))) },
	func(s Style, r *rand.Rand) Style { return s.RadiusTopLeft(float32(r.Intn(4))) },
	func(s Style, r *rand.Rand) Style { return s.BorderWidth(float32(r.Intn(3))) },
	func(s Style, r *rand.Rand) Style { return s.BorderRight(float32(r.Intn(3))) },
	func(s Style, r *rand.Rand) Style { return s.Background(Fixed(ui.RGB(uint8(r.Intn(256)), 0, 0))) },
	func(s Style, r *rand.Rand) Style { return s.Background(RoleSurface) },
	func(s Style, r *rand.Rand) Style { return s.TextColor(Fixed(ui.RGB(0, uint8(r.Intn(256)), 0))) },
	func(s Style, r *rand.Rand) Style { return s.BorderColor(RoleBorder) },
	func(s Style, r *rand.Rand) Style { return s.Opacity(float32(r.Intn(10)) / 10) },
	func(s Style, r *rand.Rand) Style { return s.Width(float32(r.Intn(200))) },
	func(s Style, r *rand.Rand) Style { return s.FontSize(float32(10 + r.Intn(8))) },
	func(s Style, r *rand.Rand) Style { return s.TextSize(Size(r.Intn(int(XXXL) + 1))) },
	func(s Style, r *rand.Rand) Style { return s.FontWeight(100 * (1 + r.Intn(8))) },
	func(s Style, r *rand.Rand) Style { return s.Italic() },
	func(s Style, r *rand.Rand) Style { return s.Shadow(0, float32(r.Intn(4)), 2, 0, RoleBorder) },
	func(s Style, r *rand.Rand) Style { return s.NoShadow() },
	func(s Style, r *rand.Rand) Style { return s.Cursor(ui.CursorPointer) },
	func(s Style, r *rand.Rand) Style { return s.Transition(time.Duration(r.Intn(300)) * time.Millisecond) },
	func(s Style, r *rand.Rand) Style { return s.HExpand() },
	func(s Style, r *rand.Rand) Style { return s.Align(Align(r.Intn(4))) },
}

func randomStyle(r *rand.Rand, depth int) Style {
	var s Style
	for range r.Intn(5) {
		s = setters[r.Intn(len(setters))](s, r)
	}
	if depth > 0 {
		for _, v := range []func(Style) Style{
			func(v Style) Style { return s.Hover(v) }, func(v Style) Style { return s.Focused(v) },
			func(v Style) Style { return s.Pressed(v) }, func(v Style) Style { return s.Disabled(v) },
			func(v Style) Style { return s.Dark(v) },
		} {
			if r.Intn(3) == 0 {
				s = v(randomStyle(r, depth-1))
			}
		}
	}
	return s
}

// winnersOf reads, by property, the value each of the layers' candidates leaves
// last.
func winnersOf(th *ui.Theme, layers []styleLayer, st frameState) map[string]string {
	names, by := candidates(th, layers, st)
	out := map[string]string{}
	for _, n := range names {
		out[n] = by[n][len(by[n])-1].value
	}
	return out
}

// expectedOf is what the control's frames resolve: the layers merged as
// restyle merges them, then effective, whose properties describe lists side by
// side.
func expectedOf(th *ui.Theme, layers []styleLayer, st frameState) map[string]string {
	var s Style
	for _, l := range layers {
		s = s.Merge(l.s)
	}
	p := s.effective(st.dark, st.hover, st.focused, st.pressed, st.disabled)
	group := map[string][]string{}
	for _, f := range p.describeSides(th, "", false) {
		group[f.Name] = append(group[f.Name], f.Value)
	}
	out := map[string]string{}
	for n, v := range group {
		out[n] = strings.Join(v, "; ")
	}
	return out
}

// The winner the tab shows for each property is what the merged Style gives a
// frame, in every state and for layers of Styles of random properties, sides
// and variants, among them the nested dark: inside a variant, and one side of
// a four-sided property set by a later layer only.
func TestInEffectIsWhatEffectiveResolves(t *testing.T) {
	th := ui.LightTheme()
	r := rand.New(rand.NewSource(1))
	for n := range 3000 {
		var layers []styleLayer
		for i := range 1 + r.Intn(3) {
			layers = append(layers, styleLayer{string(rune('A' + i)), randomStyle(r, 2)})
		}
		for bits := range 32 {
			st := frameState{bits&1 != 0, bits&2 != 0, bits&4 != 0, bits&8 != 0, bits&16 != 0}
			got, want := winnersOf(th, layers, st), expectedOf(th, layers, st)
			for _, name := range []string{"align", "valign", "hexpand", "vexpand"} { // the base's alone place the control: left out here
				delete(got, name)
				delete(want, name)
			}
			if len(got) != len(want) {
				t.Fatalf("case %d, state %v: %d properties, want %d\n got %v\nwant %v", n, st, len(got), len(want), got, want)
			}
			for name, v := range want {
				if got[name] != v {
					t.Fatalf("case %d, state %v: %s is %q, want %q", n, st, name, got[name], v)
				}
			}
		}
	}
}

// A property one layer sets on one side, over another that set all four, is a
// loser only on that side.
func TestInEffectKeepsTheSidesApart(t *testing.T) {
	th := ui.LightTheme()
	layers := []styleLayer{
		{"Apply, style 1", NewStyle().Padding(2).Hover(NewStyle().BorderTop(1))},
		{"Own methods", NewStyle().PaddingLeft(5).Hover(NewStyle().Dark(NewStyle()))},
	}
	fs := inEffect(th, layers, frameState{hover: true})
	var win, lose []string
	for _, f := range fs {
		switch {
		case f.Name == "padding-left" && !f.Struck:
			win = append(win, f.Value)
		case f.Name == "padding-left" && f.Struck:
			lose = append(lose, f.Value)
		case f.Name == "padding-top" && f.Struck:
			t.Errorf("padding-top has a loser: %v", f)
		}
	}
	if len(win) != 1 || !strings.Contains(win[0], "5 units") || !strings.Contains(win[0], "Own methods") {
		t.Errorf("padding-left winner %q, want 5 units from Own methods", win)
	}
	if len(lose) != 1 || !strings.Contains(lose[0], "2 units") || !strings.Contains(lose[0], "Apply, style 1") {
		t.Errorf("padding-left loser %q, want 2 units from Apply, style 1", lose)
	}
}

// The dark: of a variant wins over the variant, which wins over the base; a
// layout property set in a variant is no winner.
func TestInEffectOrdersVariantsAndNestedDark(t *testing.T) {
	th := ui.DarkTheme()
	l := NewStyle().Background(RoleSurface).HExpand().
		Hover(NewStyle().Background(RoleAccent).HExpand().Dark(NewStyle().Background(RoleDanger))).
		Dark(NewStyle().Background(RoleSuccess))
	layers := []styleLayer{{"Apply, style 1", l}}
	value := func(st frameState) (win string, losers int) {
		for _, f := range inEffect(th, layers, st) {
			if f.Name == "background" && !f.Struck {
				win = f.Value
			}
			if f.Name == "background" && f.Struck {
				losers++
			}
			if f.Name == "hexpand" && strings.HasPrefix(f.Value, "true") && strings.Contains(f.Value, "hover:") {
				t.Errorf("a variant's hexpand wins: %v", f)
			}
		}
		return win, losers
	}
	for _, c := range []struct {
		st     frameState
		want   string
		losers int
	}{
		{frameState{}, "role surface", 0},
		{frameState{dark: true}, "role success", 1},
		{frameState{hover: true}, "role accent", 1},
		{frameState{dark: true, hover: true}, "role danger", 3},
	} {
		got, losers := value(c.st)
		if !strings.HasPrefix(got, c.want) || losers != c.losers {
			t.Errorf("state %v: background %q with %d losers, want %q with %d", c.st, got, losers, c.want, c.losers)
		}
	}
}

func TestInEffectNamesTheStateAndAControlWithNoStyleHasNone(t *testing.T) {
	th := ui.LightTheme()
	if fs := inEffect(th, nil, frameState{}); fs != nil {
		t.Errorf("a control with no Style has %v in effect", fs)
	}
	fs := inEffect(th, []styleLayer{{"Own methods", NewStyle().Bold()}}, frameState{dark: true, hover: true})
	if len(fs) == 0 || fs[0].Name != "In effect (dark, hover)" || fs[0].Value != "" {
		t.Errorf("heading %v, want In effect (dark, hover)", fs)
	}
}

// Sides that every layer set alike show as one line, those that differ apart.
func TestInEffectJoinsTheSidesOnlyWhenTheyAreAlike(t *testing.T) {
	th := ui.LightTheme()
	has := func(fs []ui.InspectorField, name string) bool {
		for _, f := range fs {
			if f.Name == name {
				return true
			}
		}
		return false
	}
	all := inEffect(th, []styleLayer{{"Own methods", NewStyle().Padding(2).Margin(1, 2)}}, frameState{})
	if !has(all, "padding") || has(all, "padding-top") {
		t.Errorf("Padding(2): %v, want one padding line", all)
	}
	if has(all, "margin") || !has(all, "margin-top") || !has(all, "margin-left") {
		t.Errorf("Margin(1, 2) has two values: %v, want its sides apart", all)
	}
	one := inEffect(th, []styleLayer{{"Apply, style 1", NewStyle().Padding(2)}, {"Own methods", NewStyle().PaddingLeft(5)}}, frameState{})
	if has(one, "padding") || !has(one, "padding-left") || !has(one, "padding-top") {
		t.Errorf("a later layer set one side: %v, want the sides apart", one)
	}
}
