//go:build !mygo_noinspector

package tw

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

// The Widget tab shows the classes of each style Apply was given, in the order
// of Apply, and none for the control's own methods.
func TestInspectorShowsTheClassesOfEachAppliedStyle(t *testing.T) {
	b := widget.Button("Save").
		Apply(S("px-3 py-2 hover:bg-accent-hover"), S("rounded-lg"), widget.NewStyle().Opacity(0.9)).
		Bold()
	tt := widget.NewTester(t, b, 1000, 600)
	if err := tt.Inspect("Save"); err != nil {
		t.Fatal(err)
	}
	tt.Click("Widget")
	texts := tt.Texts()
	at := func(s string) int {
		for i, x := range texts {
			if x == s {
				return i
			}
		}
		return -1
	}
	first, second := at("classes: px-3 py-2 hover:bg-accent-hover"), at("classes: rounded-lg")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("classes at %d and %d, want both, the first first: %q", first, second, texts)
	}
	n := 0
	for _, x := range texts {
		if strings.HasPrefix(x, "classes: ") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d lines of classes, want 2: a Style of the typed methods and the control's own have none", n)
	}
}

// tw.S gives the Style the classes it was made from once, and the cache keeps
// them: a second call allocates nothing.
func TestSKeepsItsClassesWithoutAllocating(t *testing.T) {
	S("p-3 text-sm") // into the cache
	if n := testing.AllocsPerRun(100, func() { S("p-3 text-sm") }); n != 0 {
		t.Errorf("S of a cached string makes %v allocations", n)
	}
}
