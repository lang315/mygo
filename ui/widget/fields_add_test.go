package widget

import "testing"

// Fields.Add: a wide row with no label, as Row("", c).Wide() makes. The
// forms are compared to each other, not to fixed bounds, and built fresh
// for each side, so each has controls of its own.

// addLayouts are the three layouts Add is checked in: the default grid,
// Stacked, and SettingsRows with a section.
var addLayouts = []struct {
	name string
	make func() *FieldsWidget
}{
	{"grid", Fields},
	{"stacked", func() *FieldsWidget { return Fields().Stacked() }},
	{"settings", func() *FieldsWidget { return Fields().SettingsRows().Section("S") }},
}

// addControls are the controls Add is checked with.
var addControls = []struct {
	name string
	make func() Control
}{
	{"label", func() Control { return Label("Ghi chú dài") }},
	{"textbox", func() Control { return TextBox() }},
}

// sameForm fails unless a and b draw the same pixels and nodes at 560 x 400.
func sameForm(t *testing.T, a, b *FieldsWidget) {
	t.Helper()
	ta, tb := NewTester(t, a, 560, 400), NewTester(t, b, 560, 400)
	ta.Settle()
	tb.Settle()
	if !sameDrawing(ta, tb) {
		t.Errorf("the forms differ:\n%v\n%v", ta.Nodes(), tb.Nodes())
	}
}

// Case 1: Add(c) between two rows builds what Row("", c).Wide() does.
func TestFieldsAddIsAWideRowWithNoLabel(t *testing.T) {
	for _, l := range addLayouts {
		for _, c := range addControls {
			t.Run(l.name+"/"+c.name, func(t *testing.T) {
				add := l.make().Row("Tên", TextBox()).Add(c.make()).Row("Mã", TextBox())
				row := l.make().Row("Tên", TextBox()).Row("", c.make()).Wide().Row("Mã", TextBox())
				sameForm(t, add, row)
			})
		}
	}
}

// Case 2: Add(nil), and Add of a control already in the form, add no row
// and leave the rows before as they were: the last one is not made wide.
func TestFieldsAddNothing(t *testing.T) {
	for _, l := range addLayouts {
		t.Run(l.name, func(t *testing.T) {
			dup := TextBox()
			got := l.make().Row("Tên", TextBox()).Row("Mã", dup).Add(nil).Add(dup)
			want := l.make().Row("Tên", TextBox()).Row("Mã", TextBox())
			if got.Label(2) != nil {
				t.Errorf("Add(nil) or Add(dup) added a row: Label(2) is %v", got.Label(2))
			}
			sameForm(t, got, want)
		})
	}
}

// Case 3: Describe and Action after Add apply to its row, as after
// Row("", c).Wide().
func TestFieldsAddThenDescribeAndAction(t *testing.T) {
	for _, l := range addLayouts {
		t.Run(l.name+"/describe", func(t *testing.T) {
			add := l.make().Row("Tên", TextBox()).Add(TextBox()).Describe("d").Row("Mã", TextBox())
			row := l.make().Row("Tên", TextBox()).Row("", TextBox()).Wide().Describe("d").Row("Mã", TextBox())
			sameForm(t, add, row)
		})
		t.Run(l.name+"/action", func(t *testing.T) {
			add := l.make().Row("Tên", TextBox()).Add(TextBox()).Action(LinkLabel("x", "")).Row("Mã", TextBox())
			row := l.make().Row("Tên", TextBox()).Row("", TextBox()).Wide().Action(LinkLabel("x", "")).Row("Mã", TextBox())
			sameForm(t, add, row)
		})
	}
}

// Case 4: an Add row counts as a row in Label(i), as Row("", b) does.
func TestFieldsAddCountsInLabel(t *testing.T) {
	f := Fields().Row("A", TextBox()).Add(TextBox()).Row("C", TextBox())
	if l := f.Label(2); l == nil || l.Text() != "C" {
		t.Fatalf("Label(2) is %q, want the label C", labelText(l))
	}
	if l := f.Label(1); l == nil || l.Text() != "" {
		t.Fatalf("Label(1) is %q, want the Add row's empty label", labelText(l))
	}
}

// Case 8: Add and Row from two goroutines on one form: every Add row is
// wide, every Row row is not (run with -race).
func TestFieldsAddAndRowConcurrently(t *testing.T) {
	const n = 200
	f := Fields()
	added := map[Control]bool{}
	adds, rows := make([]Control, n), make([]Control, n)
	for i := range n {
		adds[i], rows[i] = TextBox(), TextBox()
		added[adds[i]] = true
	}
	done := make(chan bool)
	go func() {
		for _, c := range adds {
			f.Add(c)
		}
		done <- true
	}()
	go func() {
		for _, c := range rows {
			f.Row("R", c)
		}
		done <- true
	}()
	<-done
	<-done
	mu.Lock()
	defer mu.Unlock()
	if len(f.rows) != 2*n {
		t.Fatalf("%d rows, want %d", len(f.rows), 2*n)
	}
	for i, r := range f.rows {
		if r.wide != added[r.w] {
			t.Errorf("row %d: wide %v, want %v (an Add row: %v)", i, r.wide, added[r.w], added[r.w])
		}
	}
}

// labelText returns l's text, "<nil>" for none.
func labelText(l *LabelWidget) string {
	if l == nil {
		return "<nil>"
	}
	return l.Text()
}
