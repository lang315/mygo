package ui

import (
	"testing"
	"time"
)

// A theme's index stays the theme's in the frames after the one that noted
// it, also when the element that noted it is not built again: a retained
// row keeps its themeAt.
func TestBuiltThemeIndexOutlivesAFrameThatDoesNotNoteIt(t *testing.T) {
	t1, t2 := &Theme{}, &Theme{}
	both := true
	var atA uint16
	tt := coreNewTester(func(c *context) {
		if both {
			a := coreBox(c)
			a.builtWith(t1)
			atA = a.themeAt
		}
		coreBox(c).builtWith(t2)
	}, 100, 100)
	tt.Frame()
	gen := tt.rt.themeGen
	both = false
	tt.Frame()
	if got := tt.rt.themes[atA-1]; got != t1 {
		t.Fatalf("index %d of a theme noted the frame before holds another theme", atA)
	}
	if tt.rt.themeGen != gen || len(tt.rt.themes) != 2 {
		t.Fatalf("themes %d, generation %d → %d; want 2 and no start-over", len(tt.rt.themes), gen, tt.rt.themeGen)
	}
}

// An app that makes a theme in each frame does not grow the list without
// end: past themeKeep it starts over, with a new generation.
func TestThemesStartOverPastTheirKeepWithANewGeneration(t *testing.T) {
	tt := coreNewTester(func(c *context) { coreBox(c).builtWith(&Theme{}) }, 100, 100)
	tt.Frame()
	gen := tt.rt.themeGen
	for range themeKeep + 1 {
		tt.Frame()
	}
	if tt.rt.themeGen == gen {
		t.Fatalf("the generation stayed %d after %d new themes", gen, themeKeep+1)
	}
	if n := len(tt.rt.themes); n > themeKeep {
		t.Fatalf("%d themes kept, at most %d", n, themeKeep)
	}
}

// The theme list starting over in the frame an exit transition begins does
// not take from the copy going the theme its element was built with: the
// copy pins it from the list as the frame before had it.
func TestExitCopyKeepsItsThemeAcrossAStartOver(t *testing.T) {
	a := *LightTheme()
	a.AccentText = Hex("#ff0000")
	on, shown, crowd := true, true, true
	tt, now := clockTester(func(c *context) {
		c.SetTheme(&a)
		coreColumn(c).AlignItems(Start).Children(func() {
			if shown {
				coreRow(c).Key("box").Transition(ElementTransition{Duration: time.Second, Exit: &Motion{Opacity: 0}}).Children(func() {
					checkbox(c, &on, "", false)
				})
			}
		})
		other := *LightTheme()
		c.SetTheme(&other) // the frame theme is not the checkbox's, so that a copy without its theme shows
		if crowd {
			// So many themes noted that the next frame starts the list over.
			// After the column, so that the column keeps its place.
			for range themeKeep {
				coreBox(c).builtWith(&Theme{})
			}
		}
	}, 200, 100)
	gen := tt.rt.themeGen
	shown, crowd = false, false
	*now = now.Add(16 * time.Millisecond)
	tt.Frame()
	if tt.rt.themeGen == gen {
		t.Fatalf("the list did not start over: the test shows nothing")
	}
	if n := len(tt.rt.ghostThemes); n != 1 || tt.rt.ghostThemes[0] != &a {
		t.Fatalf("themes kept for the copy going: %v, want only the theme its checkbox was built with", tt.rt.ghostThemes)
	}
	var ghost *node
	for _, r := range tt.rt.trans {
		if r.ghost != nil {
			ghost = r.ghost
		}
	}
	if ghost == nil {
		t.Fatal("no copy going: the test shows nothing")
	}
	if got, ok := builtThemeOfFirst(ghost); !ok || got != &a {
		t.Errorf("the copy's checkbox paints with the theme %p, want the one it was built with, %p", got, &a)
	}
}

// builtThemeOfFirst returns the theme of the first node under e that
// was built with one, as the copy going paints with.
func builtThemeOfFirst(e *node) (*Theme, bool) {
	if e.themeAt != 0 {
		return e.builtTheme(), true
	}
	for c := e.first; c != nil; c = c.next {
		if t, ok := builtThemeOfFirst(c); ok {
			return t, true
		}
	}
	return nil, false
}
