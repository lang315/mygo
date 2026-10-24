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
		{"hover:focus:p-2", Stack},
		{"dark:dark:p-2", Stack},
		{"hover:halign-start", LayoutInVariant},
		{"dark:hexpand", LayoutInVariant},
		{"hover:transition", TransitionInVariant},
		{"focus:duration-200", TransitionInVariant},
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
