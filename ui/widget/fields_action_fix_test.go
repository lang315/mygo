package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Two gaps of Fields.Action: a control already in the form is not taken (A),
// and a wide row's label keeps its Align beside an action (B).

// occurs returns how often c is among f's controls.
func occurs(f *FieldsWidget, c Control) (n int) {
	mu.Lock()
	defer mu.Unlock()
	for _, x := range f.children() {
		if x == c {
			n++
		}
	}
	return n
}

// inTree returns how many nodes of role the form's tree has.
func inTree(tt *Tester, role ui.Role) int { return len(tt.FindAll(ByRole(role))) }

// A1: a row's own control is not its action.
func TestActionRefusesTheRowsOwnControl(t *testing.T) {
	box := TextBox()
	f := Fields().Row("A", box).Action(box)
	tt := NewTester(t, f, 400, 300)
	if n := inTree(tt, ui.RoleTextField); n != 1 {
		t.Errorf("the tree has %d text fields, want 1", n)
	}
	if n := occurs(f, box); n != 1 {
		t.Errorf("children lists the box %d times, want 1", n)
	}
}

// A2: the action of one row is not the action of another.
func TestActionRefusesAnotherRowsAction(t *testing.T) {
	a, b, link := TextBox(), TextBox(), help()
	f := Fields().Row("A", a).Action(link).Row("B", b).Action(link)
	tt := NewTester(t, f, 400, 300)
	if n := inTree(tt, ui.RoleLink); n != 1 {
		t.Errorf("the tree has %d links, want 1", n)
	}
	if n := occurs(f, link); n != 1 {
		t.Errorf("children lists the link %d times, want 1", n)
	}
	mu.Lock()
	first, second := f.rows[0].action, f.rows[1].action
	mu.Unlock()
	if first != Control(link) || second != nil {
		t.Errorf("row A has the link: %v, row B has an action: %v, want true and false", first == Control(link), second != nil)
	}
}

// A3: a row whose control is another row's action is not added.
func TestRowRefusesAnotherRowsAction(t *testing.T) {
	a, link := TextBox(), help()
	f := Fields().Row("A", a).Action(link).Row("B", link)
	tt := NewTester(t, f, 400, 300)
	if tt.Has(ByLabel("B")) {
		t.Errorf("row B is in the tree, with a control already in the form")
	}
	if n := inTree(tt, ui.RoleLink); n != 1 {
		t.Errorf("the tree has %d links, want 1", n)
	}
	if n := occurs(f, link); n != 1 {
		t.Errorf("children lists the link %d times, want 1", n)
	}
}

// A4: the row's own action again is kept.
func TestActionTwiceTheSameKeepsIt(t *testing.T) {
	link := help()
	f := Fields().Row("A", TextBox()).Action(link).Action(link)
	tt := NewTester(t, f, 400, 300)
	if n := inTree(tt, ui.RoleLink); n != 1 {
		t.Errorf("the tree has %d links, want 1", n)
	}
	mu.Lock()
	got := f.rows[0].action
	mu.Unlock()
	if got != Control(link) {
		t.Errorf("the action is %v, want the link", got)
	}
}

// A5: after Action(nil) the old action is free to be taken again.
func TestActionNilThenTheOldOne(t *testing.T) {
	link := help()
	f := Fields().Row("A", TextBox()).Action(link).Action(nil).Action(link)
	tt := NewTester(t, f, 400, 300)
	if n := inTree(tt, ui.RoleLink); n != 1 {
		t.Errorf("the tree has %d links, want 1", n)
	}
	mu.Lock()
	got := f.rows[0].action
	mu.Unlock()
	if got != Control(link) {
		t.Errorf("the action is %v, want the link", got)
	}
}
