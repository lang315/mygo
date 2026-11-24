package ui

import (
	"fmt"
	"slices"
	"testing"
)

// Cases of a TokenField's chips, which cost little a frame:
//  1. an idle frame costs at most 3 allocations a chip.
//  2. [a b c]: "Remove b" gives [a c], Changed once; "Remove c" gives [a].
//  3. [a b c], the app sets [c a], "Remove a" gives [c].
//  4. the remove buttons are named "Remove a", are buttons, and take no focus.
//  5. a remove button draws its cross and shows its hover surface.

// tokenFieldFrameAllocs is the allocations of an idle frame of a field of n
// tokens.
func tokenFieldFrameAllocs(t *testing.T, n int) float64 {
	t.Helper()
	tokens := make([]string, n)
	for i := range tokens {
		tokens[i] = fmt.Sprintf("tag%d", i)
	}
	tt := coreNewTester(func(c *context) {
		coreTokenField(c, &tokens, nil).Label("Tags").Width(600)
	}, 800, 4000)
	tt.Frame()
	tt.Frame()
	return allocsPerRun(t, 20, 0, tt.Frame)
}

func TestTokenFieldChipFrameAllocs(t *testing.T) {
	few, many := tokenFieldFrameAllocs(t, 10), tokenFieldFrameAllocs(t, 100)
	per := (many - few) / 90
	t.Logf("10 tokens: %.0f, 100 tokens: %.0f allocations a frame: %.2f a chip", few, many, per)
	if per > 3 {
		t.Errorf("an idle frame allocates %.2f a chip, want at most 3", per)
	}
}

func TestTokenFieldRemoveButtons(t *testing.T) {
	tokens := []string{"a", "b", "c"}
	changes := 0
	tt := coreNewTester(func(c *context) {
		if coreTokenField(c, &tokens, nil).Label("Tags").Width(400).Changed() {
			changes++
		}
	}, 500, 200)
	tt.Frame()
	if err := tt.Click("Remove b"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tokens, []string{"a", "c"}) || changes != 1 {
		t.Fatalf("Remove b left %q, Changed %d times", tokens, changes)
	}
	if err := tt.Click("Remove c"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tokens, []string{"a"}) || changes != 2 {
		t.Fatalf("Remove c left %q, Changed %d times", tokens, changes)
	}
}

func TestTokenFieldRemoveAfterReorder(t *testing.T) {
	tokens := []string{"a", "b", "c"}
	tt := coreNewTester(func(c *context) {
		coreTokenField(c, &tokens, nil).Label("Tags").Width(400)
	}, 500, 200)
	tt.Frame()
	tokens = []string{"c", "a"}
	tt.Frame()
	if err := tt.Click("Remove a"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tokens, []string{"c"}) {
		t.Fatalf("Remove a left %q, want [c]", tokens)
	}
}

func TestTokenFieldRemoveButtonsRoleAndFocus(t *testing.T) {
	tokens := []string{"a", "b"}
	var field *node
	tt := coreNewTester(func(c *context) {
		field = coreTokenField(c, &tokens, nil).Label("Tags").Width(400)
	}, 500, 200)
	tt.Frame()
	for _, tok := range tokens {
		var got *TestNode
		for _, n := range tt.Nodes() {
			if n.Label == "Remove "+tok {
				got = &n
			}
		}
		if got == nil || got.Role != RoleButton {
			t.Fatalf("Remove %s: %+v", tok, got)
		}
	}
	tt.Click("Tags")
	tt.Click("Remove a")
	if !tt.Focused("Tags") {
		t.Error("a remove button took the focus from the input")
	}
	var walk func(n *node)
	buttons := 0
	walk = func(n *node) {
		if n.label == "Remove a" || n.label == "Remove b" {
			buttons++
			if n.flags&flagFocusable != 0 || n.flags&flagKeepFocus == 0 || n.role != RoleButton {
				t.Errorf("%s: focusable %v, keeps focus %v, role %v", n.label, n.flags&flagFocusable != 0, n.flags&flagKeepFocus != 0, n.role)
			}
		}
		for ch := n.first; ch != nil; ch = ch.next {
			walk(ch)
		}
	}
	walk(field)
	if buttons != 1 { // "Remove a" was clicked and is gone
		t.Errorf("%d remove buttons left, want 1", buttons)
	}
}

func TestTokenFieldRemoveButtonDraws(t *testing.T) {
	tokens := []string{"abc"}
	tt := coreNewTester(func(c *context) {
		coreTokenField(c, &tokens, nil).Label("Tags").Width(400)
	}, 500, 200)
	tt.Frame()
	r, ok := tt.Find("Remove abc")
	if !ok {
		t.Fatal("no remove button")
	}
	at := func() (cross, edge [4]uint8) {
		img := tt.Image()
		cx, cy := center(r)
		x, y := int(cx), int(cy)
		ex, ey := int(r.X)+1, y // inside the button, off the cross
		o := img.PixOffset(x, y)
		copy(cross[:], img.Pix[o:o+4])
		o = img.PixOffset(ex, ey)
		copy(edge[:], img.Pix[o:o+4])
		return
	}
	cross, edge := at()
	// The cross is drawn over the center: it differs from the button's edge.
	if cross == edge {
		t.Errorf("no cross: center %v, edge %v", cross, edge)
	}
	tt.Move(center(r))
	_, hovered := at()
	if hovered == edge {
		t.Errorf("no hover surface: edge %v at rest and hovered", edge)
	}
}

// A language change renames the remove buttons: the cached label is the
// language's it was made in.
func TestTokenFieldRemoveButtonsFollowLanguage(t *testing.T) {
	tokens := []string{"a"}
	tt := coreNewTester(func(c *context) {
		coreTokenField(c, &tokens, nil).Label("Tags").Width(400)
	}, 500, 200)
	tt.Frame()
	if _, ok := tt.Find("Remove a"); !ok {
		t.Fatal("no English label")
	}
	tt.SetLocale("ja")
	if _, ok := tt.Find("aを削除"); !ok {
		t.Errorf("after ja: %q", tt.Texts())
	}
}
