package widget

import (
	"strings"
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

// labelAlignForms are the forms with a label line: wide, stacked, a card's,
// each with the left and right edge of its inside.
var labelAlignForms = map[string]func() (f *FieldsWidget, left, right float32){
	"wide": func() (*FieldsWidget, float32, float32) {
		return Fields().Row("Content", MultilineTextBox()).Wide().Action(help()), 0, 400
	},
	"stacked": func() (*FieldsWidget, float32, float32) {
		return Fields().Stacked().Row("Content", MultilineTextBox()).Action(help()), 0, 400
	},
	"card": func() (*FieldsWidget, float32, float32) {
		return Fields().SettingsRows().Row("Content", MultilineTextBox()).Wide().Action(help()), cardLeft, cardRight
	},
}

// B6 and B7: the label's Align places its text in the room left of the
// action, which ends one column gap before it.
func TestLabelAlignBesideAnAction(t *testing.T) {
	for name, form := range labelAlignForms {
		for _, a := range []Align{End, Center} {
			f, left, _ := form()
			f.Label(0).Align(a)
			tt := NewTester(t, f, 400, 400)
			l, k := find(t, tt.Tester, "Content"), linkBox(tt)
			room := k.X - space(3) // where the label's room ends
			switch a {
			case End:
				if !approx(l.X+l.W, room) {
					t.Errorf("%s, End: the label %v ends at %v, want %v, a column gap left of the link %v", name, l, l.X+l.W, room, k)
				}
			case Center:
				if !approx(l.X+l.W/2, (left+room)/2) {
					t.Errorf("%s, Center: the label %v is centred at %v, want %v, in [%v, %v]", name, l, l.X+l.W/2, (left+room)/2, left, room)
				}
			}
			if l.W > 100 {
				t.Errorf("%s: the label %v fills the line, want its text's width", name, l)
			}
		}
	}
}

// B: with no Align the label keeps its place, from the left, growing.
func TestLabelNoAlignBesideAnAction(t *testing.T) {
	for name, form := range labelAlignForms {
		f, left, _ := form()
		tt := NewTester(t, f, 400, 400)
		l, k := find(t, tt.Tester, "Content"), linkBox(tt)
		if !approx(l.X, left) || !approx(l.X+l.W, k.X-space(3)) {
			t.Errorf("%s: the label %v spans [%v, %v], want [%v, %v]", name, l, l.X, l.X+l.W, left, k.X-space(3))
		}
	}
}

// B: Align(Start) is from the left, as wide as its text.
func TestLabelStartBesideAnAction(t *testing.T) {
	for name, form := range labelAlignForms {
		f, left, _ := form()
		f.Label(0).Align(Start)
		tt := NewTester(t, f, 400, 400)
		if l := find(t, tt.Tester, "Content"); !approx(l.X, left) || l.W > 100 {
			t.Errorf("%s: the label %v starts at %v, want %v, as wide as its text", name, l, l.X, left)
		}
	}
}

// B: a label too long for the line, with an Align, clips; the action stays
// at the end.
func TestLabelAlignLongBesideAnAction(t *testing.T) {
	long := strings.Repeat("W", 120)
	f := Fields().Row(long, MultilineTextBox()).Wide().Action(help())
	f.Label(0).Align(End)
	tt := NewTester(t, f, 400, 400)
	if k := linkBox(tt); !approx(k.X+k.W, 400) {
		t.Errorf("the link %v, want it at the right edge", k)
	}
}

// B: a label with an Align but no text, or hidden, leaves the action at the
// end of the line.
func TestLabelAlignEmptyOrHiddenBesideAnAction(t *testing.T) {
	for name, build := range map[string]func() *FieldsWidget{
		"empty":  func() *FieldsWidget { return Fields().Row("", MultilineTextBox()).Wide().Action(help()) },
		"hidden": func() *FieldsWidget { return Fields().Row("Content", MultilineTextBox()).Wide().Action(help()) },
	} {
		f := build()
		f.Label(0).Align(End)
		if name == "hidden" {
			f.Label(0).SetVisible(false)
		}
		tt := NewTester(t, f, 400, 400)
		if k := linkBox(tt); !approx(k.X+k.W, 400) {
			t.Errorf("%s: the link %v, want it at the right edge", name, k)
		}
	}
}

// A6: Action(nil) removes the action of a row, whatever the other rows have.
func TestActionNilRemovesWhateverTheOthersHave(t *testing.T) {
	f := Fields().Row("A", TextBox()).Row("B", TextBox()).Action(help()).Action(nil)
	tt := NewTester(t, f, 400, 300)
	if n := inTree(tt, ui.RoleLink); n != 0 {
		t.Errorf("the tree has %d links after Action(nil), want 0", n)
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
