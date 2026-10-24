package twclass

import (
	"reflect"
	"testing"
)

func one(k Key, d Decl) []Rule { return []Rule{{k, d}} }

func TestFamilies(t *testing.T) {
	base := Key{}
	for _, c := range []struct {
		in   string
		want []Rule
	}{
		{"p-2", one(base, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"px-3", one(base, Decl{Kind: Padding, Sides: 0b1010, Value: 3})},
		{"pt-0.5", one(base, Decl{Kind: Padding, Sides: 0b0001, Value: 0.5})},
		{"p-px", one(base, Decl{Kind: Padding, Sides: 0b1111, Value: 0.25})},
		{"-mt-2", one(base, Decl{Kind: Margin, Sides: 0b0001, Value: -2})},
		{"my-96", one(base, Decl{Kind: Margin, Sides: 0b0101, Value: 96})},
		{"w-4", one(base, Decl{Kind: Width, Value: 16})},
		{"max-h-px", one(base, Decl{Kind: MaxHeight, Value: 1})},
		{"rounded", one(base, Decl{Kind: Radius, Sides: 0b1111, Value: 1})},
		{"rounded-lg", one(base, Decl{Kind: Radius, Sides: 0b1111, Value: 2})},
		{"rounded-l-none", one(base, Decl{Kind: Radius, Sides: 0b1001, Value: 0})},
		{"rounded-tr-2xl", one(base, Decl{Kind: Radius, Sides: 0b0010, Value: 4})},
		{"rounded-t", one(base, Decl{Kind: Radius, Sides: 0b0011, Value: 1})},
		{"rounded-full", one(base, Decl{Kind: Radius, Sides: 0b1111, Value: 9999})},
		{"border", one(base, Decl{Kind: BorderWidth, Sides: 0b1111, Value: 1})},
		{"border-2", one(base, Decl{Kind: BorderWidth, Sides: 0b1111, Value: 2})},
		{"border-l-0", one(base, Decl{Kind: BorderWidth, Sides: 0b1000, Value: 0})},
		{"border-x", one(base, Decl{Kind: BorderWidth, Sides: 0b1010, Value: 1})},
		{"border-t", one(base, Decl{Kind: BorderWidth, Sides: 0b0001, Value: 1})},
		{"border-dashed", one(base, Decl{Kind: BorderStyle, Value: 1})},
		{"border-solid", one(base, Decl{Kind: BorderStyle, Value: 0})},
		{"text-xs", one(base, Decl{Kind: TextSize, Value: 1})},
		{"text-base", one(base, Decl{Kind: TextSize, Value: 0})},
		{"text-3xl", one(base, Decl{Kind: TextSize, Value: 6})},
		{"text-4xl", one(base, Decl{Kind: FontSize, Value: 36})},
		{"text-9xl", one(base, Decl{Kind: FontSize, Value: 128})},
		{"font-semibold", one(base, Decl{Kind: FontWeight, Value: 600})},
		{"font-bold", one(base, Decl{Kind: FontWeight, Value: 700})},
		{"font-mono", one(base, Decl{Kind: Mono})},
		{"italic", one(base, Decl{Kind: Italic, Value: 1})},
		{"not-italic", one(base, Decl{Kind: Italic, Value: 0})},
		{"opacity-50", one(base, Decl{Kind: Opacity, Value: 0.5})},
		{"cursor-not-allowed", one(base, Decl{Kind: Cursor, Name: "not-allowed"})},
		{"halign-start", one(base, Decl{Kind: Align, Value: 1})},
		{"halign-fill", one(base, Decl{Kind: Align, Value: 4})},
		{"valign-center", one(base, Decl{Kind: VAlign, Value: 2})},
		{"hexpand", one(base, Decl{Kind: HExpand})},
		{"vexpand", one(base, Decl{Kind: VExpand})},
		{"transition", one(base, Decl{Kind: Transition, Value: 150})},
		{"duration-300", one(base, Decl{Kind: Transition, Value: 300})},
		{"hover:p-2", one(Key{Variant: Hover}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"active:p-2", one(Key{Variant: Active}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"focus:p-2", one(Key{Variant: Focus}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"disabled:p-2", one(Key{Variant: Disabled}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"dark:p-2", one(Key{Dark: true}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"dark:hover:p-2", one(Key{Variant: Hover, Dark: true}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"hover:dark:p-2", one(Key{Variant: Hover, Dark: true}, Decl{Kind: Padding, Sides: 0b1111, Value: 2})},
		{"", nil},
	} {
		got, bad := Parse(c.in)
		if len(bad) != 0 || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Parse(%q) = %+v, bad %v; want %+v", c.in, got, bad, c.want)
		}
	}
}

func TestBadClasses(t *testing.T) {
	for _, c := range []struct{ in, reason string }{
		{"p-13", Unknown},
		{"m-auto", Unknown},
		{"-p-2", Unknown},
		{"border-dotted", Unknown},
		{"rounded-q", Unknown},
		{"text-italic", Unknown},
		{"sm:p-2", Unknown},
		{"hover:", Unknown},
		{"p-[13px]", Unknown},
		{"rounded-", Unknown},
		{"rounded-t-", Unknown},
		{"border-", Unknown},
		{"border-t-", Unknown},
		{"border-x-", Unknown},
		{"hover:focus:p-2", Stack},
		{"dark:dark:p-2", Stack},
		{"hover:halign-start", LayoutInVariant},
		{"dark:hexpand", LayoutInVariant},
		{"hover:transition", TransitionInVariant},
		{"focus:duration-200", TransitionInVariant},
		{"hover:not-italic", NotItalicInVariant},
		{"dark:not-italic", NotItalicInVariant},
		{"hover:suggested-action", Unknown},
		{"\xff\xfe", Unknown},
	} {
		_, bad := Parse(c.in)
		if len(bad) != 1 || bad[0] != (Bad{c.in, c.reason}) {
			t.Errorf("Parse(%q) bad = %v, want [{%q %q}]", c.in, bad, c.in, c.reason)
		}
	}
}

func TestBadClassesDoNotStopTheOthers(t *testing.T) {
	got, bad := Parse("p-2 bogus mt-3")
	want := []Rule{{Key{}, Decl{Kind: Padding, Sides: 0b1111, Value: 2}}, {Key{}, Decl{Kind: Margin, Sides: 0b0001, Value: 3}}}
	if !reflect.DeepEqual(got, want) || len(bad) != 1 || bad[0].Class != "bogus" {
		t.Fatalf("got %+v bad %v", got, bad)
	}
}

func TestWhitespace(t *testing.T) {
	a, _ := Parse("p-2 mt-3")
	b, _ := Parse("\tp-2\n\n  mt-3  ")
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("%+v != %+v", a, b)
	}
	if r, bad := Parse(" \t\n"); r != nil || bad != nil {
		t.Fatalf("blank: %v %v", r, bad)
	}
}

func TestLaterWins(t *testing.T) {
	got, _ := Parse("p-2 p-4")
	if len(got) != 2 || got[1].Value != 4 {
		t.Fatalf("%+v: both rules, in order, so the builder's later one wins", got)
	}
}

func TestTransitionAndDuration(t *testing.T) {
	for in, ms := range map[string]float32{"transition": 150, "duration-300 transition": 300, "transition duration-300": 300, "duration-75": 75} {
		got, _ := Parse(in)
		if len(got) != 1 || got[0].Kind != Transition || got[0].Value != ms {
			t.Errorf("Parse(%q) = %+v, want one Transition of %v ms", in, got, ms)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"p-2 hover:bg-blue-600", "dark:hover:rounded-l-none", "::", "-", "rounded--", "border-x-", "\xff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { Parse(s) })
}

func TestColors(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Decl
	}{
		{"bg-blue-500", Decl{Kind: Background, Color: Color{RGBA: 0x3b82f6ff}}},
		{"text-gray-500", Decl{Kind: TextColor, Color: Color{RGBA: 0x6b7280ff}}},
		{"border-red-900", Decl{Kind: BorderColor, Color: Color{RGBA: 0x7f1d1dff}}},
		{"bg-slate-950", Decl{Kind: Background, Color: Color{RGBA: 0x020617ff}}},
		{"bg-yellow-100", Decl{Kind: Background, Color: Color{RGBA: 0xfef9c3ff}}},
		{"text-black", Decl{Kind: TextColor, Color: Color{RGBA: 0x000000ff}}},
		{"text-white", Decl{Kind: TextColor, Color: Color{RGBA: 0xffffffff}}},
		{"bg-transparent", Decl{Kind: Background, Color: Color{RGBA: 0}}},
		{"bg-default", Decl{Kind: Background, Color: Color{Role: "background"}}},
		{"text-default", Decl{Kind: TextColor, Color: Color{Role: "text"}}},
		{"border-default", Decl{Kind: BorderColor, Color: Color{Role: "border"}}},
		{"text-muted", Decl{Kind: TextColor, Color: Color{Role: "text-muted"}}},
		{"bg-surface-hover", Decl{Kind: Background, Color: Color{Role: "surface-hover"}}},
		{"text-on-accent", Decl{Kind: TextColor, Color: Color{Role: "accent-text"}}},
		{"bg-on-inverse", Decl{Kind: Background, Color: Color{Role: "inverse-text"}}},
		{"bg-danger", Decl{Kind: Background, Color: Color{Role: "danger"}}},
	} {
		got, bad := Parse(c.in)
		if len(bad) != 0 || !reflect.DeepEqual(got, one(Key{}, c.want)) {
			t.Errorf("Parse(%q) = %+v, bad %v", c.in, got, bad)
		}
	}
	for _, in := range []string{"bg-blue-550", "bg-blue", "text-green", "border-current", "bg-blue-600/50", "border-l-red-500"} {
		if _, bad := Parse(in); len(bad) != 1 {
			t.Errorf("Parse(%q) accepted", in)
		}
	}
}

func TestPaletteIsComplete(t *testing.T) {
	if len(palette) != 22 {
		t.Fatalf("%d hues, want Tailwind v3's 22", len(palette))
	}
	for hue := range palette {
		for _, shade := range []string{"50", "100", "200", "300", "400", "500", "600", "700", "800", "900", "950"} {
			if _, bad := Parse("bg-" + hue + "-" + shade); len(bad) != 0 {
				t.Errorf("bg-%s-%s: %v", hue, shade, bad)
			}
		}
	}
}

func TestEveryRoleIsListed(t *testing.T) {
	listed := map[string]bool{}
	for _, r := range Roles {
		listed[r] = true
	}
	for tok, role := range roles {
		if !listed[role] {
			t.Errorf("token %q gives role %q, not in Roles", tok, role)
		}
	}
	for _, role := range defaults {
		if !listed[role] {
			t.Errorf("default role %q not in Roles", role)
		}
	}
}

func TestShadows(t *testing.T) {
	got, _ := Parse("shadow-md")
	want := one(Key{}, Decl{Kind: Shadows, Shadows: []Shadow{{0, 4, 6, -1, 0.1, false}, {0, 2, 4, -2, 0.1, false}}})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shadow-md = %+v", got)
	}
	if got, _ := Parse("shadow-none"); len(got) != 1 || got[0].Kind != Shadows || got[0].Shadows != nil {
		t.Fatalf("shadow-none = %+v", got)
	}
	if got, _ := Parse("shadow-inner"); len(got) != 1 || !got[0].Shadows[0].Inset {
		t.Fatalf("shadow-inner = %+v", got)
	}
}

func TestButtons(t *testing.T) {
	got, bad := Parse("suggested-action")
	want, _ := Parse("bg-accent text-on-accent hover:bg-accent-hover active:bg-accent-pressed")
	if len(bad) != 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("suggested-action = %+v %v", got, bad)
	}
	got, bad = Parse("destructive-action")
	want, _ = Parse("bg-danger text-on-accent hover:opacity-90 active:opacity-80")
	if len(bad) != 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("destructive-action = %+v %v", got, bad)
	}
}

// fhtool lists the classes fhtool's AddClass calls use (recounted
// 2026-10-24): those tw takes, and those its port must change.
var fhtoolGood = []string{
	"halign-start", "p-2", "mt-3", "mb-3", "p-3", "mt-5", "font-bold", "hexpand", "halign-end",
	"text-gray-500", "rounded-l-none", "border-l-0", "rounded-r-none", "border-solid", "text-xl",
	"border-r-0", "border", "px-3", "halign-center", "suggested-action", "text-xs", "rounded-md",
	"py-2", "p-6", "p-4", "mt-2", "border-2", "text-muted", "text-lg", "text-black", "rounded-none",
	"pt-3", "pb-1", "ml-2", "mb-2", "italic", "font-mono", "vexpand", "text-white", "text-sm",
	"text-gray-900", "p-0", "ml-5", "ml-4", "ml-3", "mb-5", "m-2", "m-10", "m-0", "destructive-action",
	"border-yellow-500", "border-t", "border-red-900", "border-gray-300", "border-b", "bg-yellow-100",
	"bg-blue-50",
}

var fhtoolBad = []string{
	"text-italic", "text-green", "combobox-rounded-r-none", "combobox-rounded-none", "gota-card",
	"gota-table", "gota-table-bottom-panel", "gota-tag-select", "gota-tag-select-item",
	"gota-tag-select-item-text",
}

func TestFhtoolClasses(t *testing.T) {
	if len(fhtoolGood) != 57 || len(fhtoolBad) != 10 {
		t.Fatalf("lists hold %d and %d, want 57 and 10", len(fhtoolGood), len(fhtoolBad))
	}
	for _, c := range fhtoolGood {
		if _, bad := Parse(c); len(bad) != 0 {
			t.Errorf("%s: %v", c, bad)
		}
	}
	for _, c := range fhtoolBad {
		if _, bad := Parse(c); len(bad) != 1 || bad[0].Reason != Unknown {
			t.Errorf("%s: %v, want unknown", c, bad)
		}
	}
}

func TestParseResultsDoNotAliasTheTables(t *testing.T) {
	r, _ := Parse("shadow-md")
	r[0].Shadows[0].Alpha = 0.9
	again, _ := Parse("shadow-md")
	if again[0].Shadows[0].Alpha != 0.1 {
		t.Fatalf("a caller's write reached the table: alpha %v", again[0].Shadows[0].Alpha)
	}
	if n, _ := Parse("shadow-none"); n[0].Shadows != nil {
		t.Fatalf("shadow-none = %+v, want nil shadows", n[0].Shadows)
	}
}
