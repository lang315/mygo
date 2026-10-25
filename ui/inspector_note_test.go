//go:build !mygo_noinspector

package ui

import (
	"testing"
)

// noted is a view of a box with a note for the inspector, holding a text, and
// a count of the notes made.
func noted(made *int, state *notedState) func(c *context) {
	return func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			if state.hidden {
				return
			}
			box := coreBox(c).Key("card").Width(160).Height(40).Label("Card")
			state.hover, state.pressed, state.focus = box.Hovered(), box.Pressed(), box.FocusVisible()
			if c.Inspecting() {
				*made++
				box.InspectNote("Card", func() []InspectorField {
					return []InspectorField{{Name: "Style"}, {Name: "padding", Value: "2"}, {Name: "variants", Value: "hover"}}
				})
			}
			box.Children(func() { coreText(c, "Hi") })
		})
		coreText(c, "Plain")
	}
}

type notedState struct {
	hidden                bool
	hover, pressed, focus bool
}

func TestInspectorWidgetTab(t *testing.T) {
	made := 0
	var st notedState
	tt, _ := inspected(noted(&made, &st))
	tt.Frame()
	if made != 0 {
		t.Fatalf("%d notes made while the inspector is closed", made)
	}
	openInspector(tt)
	if made == 0 {
		t.Fatal("no note made while the inspector is open")
	}
	// The text inside the box is chosen: the note is the box's.
	var text *node
	var walk func(e *node)
	walk = func(e *node) {
		if e.kind == kindText && e.text == "Hi" {
			text = e
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(tt.rt.c.root)
	if text == nil {
		t.Fatal("no text in the tree")
	}
	tt.rt.insp.choose(text.id)
	tt.Frame()
	if !tt.HasText("Widget") {
		t.Fatalf("no Widget tab for an element of a noted control: %q", tt.Texts())
	}
	tt.Click("Widget")
	for _, s := range []string{"Card", "from Column, 1 up", "Force state", "padding", "variants"} {
		if !tt.HasText(s) {
			t.Errorf("the Widget tab lacks %q: %q", s, tt.Texts())
		}
	}
	// An element of no control has no Widget tab.
	for _, e := range tt.rt.insp.nodes {
		if e.name == "Text" && e.text == "Plain" {
			tt.rt.insp.choose(e.id)
		}
	}
	tt.Frame()
	if tt.HasText("Force state") {
		t.Errorf("an element of no control shows the Widget tab: %q", tt.Texts())
	}
}

func TestInspectorForcesStates(t *testing.T) {
	made := 0
	var st notedState
	tt, _ := inspected(noted(&made, &st))
	openInspector(tt)
	for _, n := range tt.rt.insp.nodes {
		if n.label == "Card" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	tt.Click("Widget")
	for _, c := range []struct {
		button string
		state  func() bool
	}{
		{"Force hover", func() bool { return st.hover }},
		{"Force pressed", func() bool { return st.pressed }},
		{"Force focus", func() bool { return st.focus }},
	} {
		if c.state() {
			t.Fatalf("%s: already in the state", c.button)
		}
		if err := tt.Click(c.button); err != nil {
			t.Fatal(err)
		}
		if !c.state() {
			t.Errorf("%s: the element is not in the state", c.button)
		}
		if err := tt.Click(c.button); err != nil {
			t.Fatal(err)
		}
		if c.state() {
			t.Errorf("%s: the state stays after the button is pressed again", c.button)
		}
	}
	// Forced, then another element chosen: the state goes.
	tt.Click("Force hover")
	if !st.hover {
		t.Fatal("hover not forced")
	}
	for _, n := range tt.rt.insp.nodes {
		if n.name == "Text" && n.text == "Plain" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	if st.hover || tt.rt.insp.forced != 0 {
		t.Error("the state stays when another element is chosen")
	}
	// Forced, then the inspector closed: the state goes.
	for _, n := range tt.rt.insp.nodes {
		if n.label == "Card" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	tt.Click("Widget")
	tt.Click("Force hover")
	if !st.hover {
		t.Fatal("hover not forced again")
	}
	tt.rt.toggleInspector()
	tt.Frame()
	if st.hover || tt.rt.insp.forced != 0 {
		t.Error("the state stays when the inspector closes")
	}
	// Forced, then the element gone: the state goes.
	openInspector(tt)
	for _, n := range tt.rt.insp.nodes {
		if n.label == "Card" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	tt.Click("Widget")
	tt.Click("Force focus")
	if !st.focus {
		t.Fatal("focus not forced")
	}
	st.hidden = true
	tt.Frame()
	if tt.rt.insp.forced != 0 {
		t.Error("the state stays when the element is gone")
	}
}

// The note's fields are made when the Widget tab shows them, not in the
// frames of the other tabs: they may change each frame and would redraw the
// panel each time.
func TestInspectorNoteFieldsAreMadeOnlyForTheWidgetTab(t *testing.T) {
	calls := 0
	tt, _ := inspected(func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			box := coreBox(c).Key("card").Width(160).Height(40).Label("Card")
			if c.Inspecting() {
				box.InspectNote("Card", func() []InspectorField {
					calls++
					return []InspectorField{{Name: "n", Value: "1"}}
				})
			}
		})
	})
	openInspector(tt)
	for _, n := range tt.rt.insp.nodes {
		if n.label == "Card" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	tt.Frame()
	if calls != 0 {
		t.Fatalf("the fields were made %d times while the Styles tab shows", calls)
	}
	tt.Click("Widget")
	if calls == 0 || !tt.HasText("n: 1") {
		t.Errorf("the Widget tab does not show the fields (%d calls): %q", calls, tt.Texts())
	}
}

// Closing the inspector forgets the notes: they keep the last frame's data.
func TestInspectorForgetsTheNotesWhenItCloses(t *testing.T) {
	made := 0
	var st notedState
	tt, _ := inspected(noted(&made, &st))
	openInspector(tt)
	if len(tt.rt.insp.notes) == 0 {
		t.Fatal("no note while open")
	}
	tt.rt.toggleInspector()
	if len(tt.rt.insp.notes) != 0 {
		t.Errorf("%d notes kept after the inspector closed", len(tt.rt.insp.notes))
	}
}

// The tab lit is the pane shown: with the Widget tab chosen and an element
// that has no note chosen, the pane is Styles and that tab is lit.
func TestInspectorLightsTheTabItShows(t *testing.T) {
	made := 0
	var st notedState
	tt, _ := inspected(noted(&made, &st))
	openInspector(tt)
	for _, n := range tt.rt.insp.nodes {
		if n.label == "Card" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	tt.Click("Widget")
	if tt.rt.insp.side != inspWidget {
		t.Fatal("the Widget tab is not chosen")
	}
	for _, n := range tt.rt.insp.nodes {
		if n.name == "Text" && n.text == "Plain" {
			tt.rt.insp.choose(n.id)
		}
	}
	tt.Frame()
	lit := map[string]bool{}
	var walk func(e *node)
	walk = func(e *node) {
		if e.role == RoleTab && e.checked == 2 {
			lit[e.label] = true
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(tt.rt.c.root)
	if !lit["Styles"] || lit["Widget"] || lit["Computed"] || lit["Properties"] {
		t.Errorf("tabs lit: %v, want only Styles (the pane shown)", lit)
	}
}
