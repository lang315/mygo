//go:build !mygo_noinspector

package ui

import (
	"slices"
	"strings"
	"testing"
)

// widgetsView is a window of controls with notes, as ui/widget leaves them:
// a Panel holding a GroupBox, which holds a TextBox and, inside a plain
// Column, a Button, with another Button after the GroupBox.
func widgetsView(made *int) func(c *context) {
	note := func(c *context, e *node, title string) {
		if c.Inspecting() {
			*made++
			e.InspectNote(title, func() []InspectorField { return []InspectorField{{Name: title}} })
		}
	}
	return func(c *context) {
		var v string
		panel := coreColumn(c).Padding(10).Gap(4)
		note(c, panel, "*widget.PanelWidget")
		panel.Children(func() {
			group := coreColumn(c).Label("Account").Gap(4)
			note(c, group, "*widget.GroupBoxWidget")
			group.Children(func() {
				box := coreTextInput(c, &v).Label("Name")
				note(c, box, "*widget.TextBoxWidget")
				coreRow(c).Children(func() { // a Column of no control
					b := coreButton(c, "Save")
					note(c, b, "*widget.ButtonWidget")
				})
			})
			b := coreButton(c, "Close")
			note(c, b, "*widget.ButtonWidget")
		})
	}
}

// widgetRows returns the rows of the Widgets tab that name one of the types
// of widgetsView, in order. A row is two texts, its label for assistive
// technology and the markup, so it takes one of each pair.
func widgetRows(tt *Tester) []string {
	var all []string
	for _, s := range tt.Texts() {
		for _, t := range []string{"Panel", "GroupBox", "TextBox", "Button"} {
			if s == t || strings.HasPrefix(s, t+" ") {
				all = append(all, s)
			}
		}
	}
	var out []string
	for i := 0; i < len(all); i += 2 {
		out = append(out, all[i])
	}
	return out
}

func TestInspectorWidgetsShowTheControlsAsTheAppWroteThem(t *testing.T) {
	made := 0
	tt, _ := inspected(widgetsView(&made))
	openInspector(tt)
	if made == 0 {
		t.Fatal("no notes made while the inspector is open")
	}
	if tt.HasText("Elements") && !tt.HasText("Widgets") {
		t.Fatalf("no Widgets switch beside Elements: %q", tt.Texts())
	}
	if err := tt.Click("Widgets"); err != nil { // no frame asked for after: the click asks for it
		t.Fatal(err)
	}
	got := widgetRows(tt)
	want := []string{"Panel", "GroupBox", "TextBox", "Button", "Button"}
	if len(got) < len(want) {
		t.Fatalf("rows %q, want the types %q in order", got, want)
	}
	for i, w := range want {
		if !(got[i] == w || strings.HasPrefix(got[i], w+" ")) {
			t.Errorf("row %d is %q, want %q (rows %q)", i, got[i], w, got)
		}
	}
	// They nest by the controls around them: the Panel holds the GroupBox,
	// which holds the TextBox and the Save button (inside a Row of no control);
	// the Close button is in the Panel.
	in := &tt.rt.insp
	var depths []int32
	for _, r := range in.rows {
		depths = append(depths, in.viewDepth(&in.nodes[r.node]))
	}
	if want := []int32{0, 1, 2, 2, 1}; !slices.Equal(depths, want) {
		t.Errorf("row depths %v, want %v", depths, want)
	}
	// The elements of no control are not rows.
	for _, s := range tt.Texts() {
		if strings.HasPrefix(s, "Column") || strings.HasPrefix(s, "Row") || strings.HasPrefix(s, "Text") && !strings.HasPrefix(s, "TextBox") {
			t.Errorf("row %q of an element that is no control", s)
		}
	}
	// Back to the elements: the tags of elements.
	if err := tt.Click("Elements"); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(tt.Texts(), func(s string) bool { return strings.HasPrefix(s, "Column") }) {
		t.Errorf("the Elements tree lacks its Column: %q", tt.Texts())
	}
}

// Choosing a row chooses the control's root element, so the tabs follow.
func TestInspectorWidgetsRowChoosesTheRootElement(t *testing.T) {
	made := 0
	tt, _ := inspected(widgetsView(&made))
	openInspector(tt)
	if err := tt.Click("Widgets"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("GroupBox"); err != nil {
		t.Fatal(err)
	}
	sel := tt.rt.insp.selected
	var chosen *inspNode
	for i := range tt.rt.insp.nodes {
		if tt.rt.insp.nodes[i].id == sel {
			chosen = &tt.rt.insp.nodes[i]
		}
	}
	if chosen == nil || chosen.label != "Account" {
		t.Fatalf("chose %+v, want the GroupBox's element (label Account)", chosen)
	}
	if !tt.HasText("Widget") {
		t.Errorf("no Widget tab for the control chosen: %q", tt.Texts())
	}
}

// An element of a control, chosen elsewhere, shows its control's row chosen.
func TestInspectorWidgetsShowTheControlOfTheElementChosen(t *testing.T) {
	made := 0
	tt, _ := inspected(widgetsView(&made))
	openInspector(tt)
	tt.rt.insp.choose(tt.rt.insp.nodes[0].id)
	if err := tt.Click("Widgets"); err != nil {
		t.Fatal(err)
	}
	in := &tt.rt.insp
	for i := range in.nodes {
		if in.nodes[i].name == "Text" && in.nodes[i].text == "Save" {
			in.choose(in.nodes[i].id) // the label inside the Save button
		}
	}
	tt.Frame()
	if r := in.row; r < 0 || r >= len(in.rows) {
		t.Fatalf("no row chosen for an element inside a control (row %d)", in.row)
	}
}
