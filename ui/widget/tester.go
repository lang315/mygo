package widget

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// TB is what the testers need of a test: *testing.T and *testing.B.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
	Cleanup(func())
}

// Tester drives widgets without a window, for tests: a ui.Tester that
// finds them as assistive technology does, by label or role, and acts on
// them as the user would.
//
//	tt := widget.NewTester(t, form, 0, 0)
//	tt.Find(widget.ByLabel("Name")).Type("Ada")
//	tt.Find(widget.ByLabel("OK")).Click()
//
// A Tester embeds the *ui.Tester under it, so its Frame, Key, Texts and
// the rest work on the same window. Its methods fail the test (Fatalf)
// where a user could not do what they ask, and say why.
type Tester struct {
	*ui.Tester
	tb   TB
	w, h int
}

// NewTester returns a Tester of c in a window of w by h DIPs (0 for 600
// by 800).
func NewTester(t TB, c Control, w, h int) *Tester { return newTester(t, content(c), w, h) }

// TestForm returns a Tester of f's view: its header, tool strip, content
// and status strip, in a window of the form's size (600 by 800 where it
// has none, as a form of natural size).
func TestForm(t TB, f *Form) *Tester {
	mu.Lock()
	w, h := f.width, f.height
	mu.Unlock()
	return newTester(t, f.view, w, h)
}

func newTester(t TB, view func(c *ui.Context), w, h int) *Tester {
	if w <= 0 {
		w = 600
	}
	if h <= 0 {
		h = 800
	}
	tt := &Tester{Tester: ui.NewTester(view, w, h), tb: t, w: w, h: h}
	tt.Frame()
	return tt
}

func (t *Tester) fatalf(format string, args ...any) {
	t.tb.Helper()
	t.tb.Fatalf(format, args...)
}

// Settle runs what was posted to the event goroutine, then draws a frame.
// Every action of a Node does. Do not call it from a handler, which runs
// on that goroutine.
func (t *Tester) Settle() {
	done := make(chan struct{})
	Post(func() { close(done) })
	<-done
	t.Frame()
}

// Query says which element to find: ByLabel and ByRole make one, and
// Label, Role and Nth narrow it.
type Query struct {
	role     ui.Role
	hasRole  bool
	label    string
	hasLabel bool
	nth      int
	hasNth   bool
}

// ByLabel matches the elements named s: a button's text, a field's label,
// a text shown. The match is exact and case-sensitive.
func ByLabel(s string) Query { return Query{label: s, hasLabel: true} }

// ByRole matches the elements of a role (ui.RoleButton, ui.RoleTextField).
func ByRole(r ui.Role) Query { return Query{role: r, hasRole: true} }

// Label narrows q to the elements named s.
func (q Query) Label(s string) Query { q.label, q.hasLabel = s, true; return q }

// Role narrows q to the elements of a role.
func (q Query) Role(r ui.Role) Query { q.role, q.hasRole = r, true; return q }

// Nth picks the i-th element (from 0) of those q matches, in tree order.
func (q Query) Nth(i int) Query { q.nth, q.hasNth = i, true; return q }

func (q Query) matches(n ui.TestNode) bool {
	return (!q.hasRole || n.Role == q.role) && (!q.hasLabel || n.Label == q.label)
}

func (q Query) String() string {
	s := "element"
	if q.hasRole {
		s = roleName(q.role)
	}
	if q.hasLabel {
		s += fmt.Sprintf(" labeled %q", q.label)
	}
	return s
}

// matching returns the nodes q matches before Nth picks one.
func (t *Tester) matching(q Query) []ui.TestNode {
	t.Frame()
	var out []ui.TestNode
	for _, n := range t.Nodes() {
		if q.matches(n) {
			out = append(out, n)
		}
	}
	return out
}

// FindAll returns every element q matches, in tree order (all of them
// where q has Nth).
func (t *Tester) FindAll(q Query) []*Node {
	var out []*Node
	for _, n := range t.matching(q) {
		out = append(out, &Node{t: t, id: n.ID, role: n.Role, label: n.Label})
	}
	return out
}

// Has reports whether an element matches q.
func (t *Tester) Has(q Query) bool { q.hasNth = false; return len(t.matching(q)) > 0 }

// Find returns the one element q matches, and fails the test when none
// does or several do, listing what it found.
func (t *Tester) Find(q Query) *Node {
	t.tb.Helper()
	all := t.matching(Query{role: q.role, hasRole: q.hasRole, label: q.label, hasLabel: q.hasLabel})
	switch {
	case q.hasNth && q.nth >= 0 && q.nth < len(all):
		all = all[q.nth : q.nth+1]
	case q.hasNth:
		t.fatalf("Nth(%d) of the %d %s (found: %s)", q.nth, len(all), q, t.found(q))
	case len(all) == 0:
		t.fatalf("no %s (found: %s)", q, t.found(q))
	case len(all) > 1:
		t.fatalf("%d elements match %s (found: %s)", len(all), q, t.found(q))
	}
	n := all[0]
	return &Node{t: t, id: n.ID, role: n.Role, label: n.Label}
}

// found lists the elements near a query that failed: those of its role,
// or the named ones when it has none.
func (t *Tester) found(q Query) string {
	var have []string
	for _, n := range t.Nodes() {
		if q.hasRole && n.Role != q.role || !q.hasRole && n.Label == "" {
			continue
		}
		have = append(have, fmt.Sprintf("%s %q", roleName(n.Role), n.Label))
		if len(have) == 12 {
			have = append(have, "...")
			break
		}
	}
	if len(have) == 0 {
		return "none"
	}
	return strings.Join(have, ", ")
}

// Node is an element a Tester found. Its reads and actions look it up
// again by its identity in the last frame, so they follow what the app
// did to it; the test fails if it is gone.
type Node struct {
	t     *Tester
	id    uint64
	role  ui.Role
	label string
}

func (n *Node) String() string { return fmt.Sprintf("%s %q", roleName(n.role), n.label) }

// live returns the node as the last frame shows it.
func (n *Node) live() ui.TestNode {
	n.t.tb.Helper()
	n.t.Frame()
	for _, x := range n.t.Nodes() {
		if x.ID == n.id {
			return x
		}
	}
	n.t.fatalf("%s is no longer shown", n)
	return ui.TestNode{}
}

// Role returns the element's role; Label its name; Value the text of a
// field or the value of a control that has one.
func (n *Node) Role() ui.Role   { return n.role }
func (n *Node) Label() string   { n.t.tb.Helper(); return n.live().Label }
func (n *Node) Value() string   { n.t.tb.Helper(); return n.live().Value }
func (n *Node) Bounds() ui.Rect { n.t.tb.Helper(); return n.live().Bounds }
func (n *Node) Checked() bool   { n.t.tb.Helper(); return n.live().Checked }
func (n *Node) Selected() bool  { n.t.tb.Helper(); return n.live().Selected }
func (n *Node) Disabled() bool  { n.t.tb.Helper(); return n.live().Disabled }
func (n *Node) Focused() bool   { n.t.tb.Helper(); return n.live().Focused }
func (n *Node) Min() float64    { n.t.tb.Helper(); return n.live().Min }
func (n *Node) Max() float64    { n.t.tb.Helper(); return n.live().Max }
func (n *Node) Now() float64    { n.t.tb.Helper(); return n.live().Now }

// target returns the center of the element, failing the test where a user
// could not click it: it is disabled, or outside the window.
func (n *Node) target(what string) (x, y float32) {
	n.t.tb.Helper()
	l := n.live()
	if l.Disabled {
		n.t.fatalf("cannot %s %s: it is disabled", what, n)
	}
	x, y = l.Bounds.X+l.Bounds.W/2, l.Bounds.Y+l.Bounds.H/2
	if l.Bounds.W <= 0 || x < 0 || y < 0 || x >= float32(n.t.w) || y >= float32(n.t.h) {
		n.t.fatalf("cannot %s %s: it is at (%v, %v), outside the window", what, n, x, y)
	}
	return x, y
}

// Click clicks the middle of the element, then settles. The same goes for
// the other actions, which return the node to read from.
func (n *Node) Click() *Node {
	n.t.tb.Helper()
	x, y := n.target("click")
	n.t.ClickAt(x, y)
	n.t.Settle()
	return n
}

// DoubleClick clicks it twice, within the double-click interval.
func (n *Node) DoubleClick() *Node {
	n.t.tb.Helper()
	x, y := n.target("double-click")
	n.t.ClickAt(x, y)
	n.t.ClickAt(x, y)
	n.t.Settle()
	return n
}

// RightClick opens its context menu, as the user would.
func (n *Node) RightClick() *Node {
	n.t.tb.Helper()
	x, y := n.target("right-click")
	n.t.RightClickAt(x, y)
	n.t.Settle()
	return n
}

// Focus gives it the keyboard focus, by a click where it has none.
func (n *Node) Focus() *Node {
	n.t.tb.Helper()
	if !n.live().Focused {
		x, y := n.target("focus")
		n.t.ClickAt(x, y)
		n.t.Settle()
	}
	return n
}

// Type focuses the element and types s.
func (n *Node) Type(s string) *Node {
	n.t.tb.Helper()
	n.Focus()
	n.t.Tester.Type(s)
	n.t.Settle()
	return n
}

// Key focuses the element and presses a key.
func (n *Node) Key(mods ui.Modifiers, key ui.Key) *Node {
	n.t.tb.Helper()
	n.Focus()
	n.t.Tester.Key(mods, key)
	n.t.Settle()
	return n
}

var roleNames = map[ui.Role]string{
	ui.RoleGroup: "group", ui.RoleText: "text", ui.RoleButton: "button", ui.RoleLink: "link", ui.RoleCheckBox: "check box",
	ui.RoleRadio: "radio button", ui.RoleSwitch: "switch", ui.RoleSlider: "slider", ui.RoleProgress: "progress bar",
	ui.RoleTextField: "text field", ui.RoleImage: "image", ui.RoleList: "list", ui.RoleScroll: "scroll area", ui.RoleDialog: "dialog",
	ui.RolePopup: "popup", ui.RoleTooltip: "tooltip", ui.RolePopUpButton: "pop-up button", ui.RoleTabList: "tab list", ui.RoleTab: "tab",
	ui.RoleSplitter: "splitter", ui.RoleStatus: "status", ui.RoleTable: "table", ui.RoleRow: "row", ui.RoleCell: "cell",
	ui.RoleColumnHeader: "column header", ui.RoleTree: "tree", ui.RoleTreeItem: "tree item", ui.RoleListItem: "list item",
	ui.RoleMenuButton: "menu button", ui.RoleToolbar: "toolbar", ui.RoleRadioGroup: "radio group", ui.RoleToggleButton: "toggle button",
	ui.RoleComboBox: "combo box", ui.RoleDisclosure: "disclosure", ui.RoleMeter: "meter", ui.RoleStepper: "stepper",
	ui.RoleColorWell: "color well", ui.RoleAlertDialog: "alert dialog", ui.RoleMenu: "menu", ui.RoleMenuBar: "menu bar",
	ui.RoleMenuItem: "menu item", ui.RoleMenuItemCheckBox: "menu item", ui.RoleMenuItemRadio: "menu item", ui.RoleHeading: "heading",
}

func roleName(r ui.Role) string {
	if s, ok := roleNames[r]; ok {
		return s
	}
	return fmt.Sprintf("role %d", r)
}
