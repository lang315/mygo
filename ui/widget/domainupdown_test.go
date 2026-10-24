package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func domainTester(t *testing.T, d *DomainUpDownWidget) (*Tester, *Node) {
	t.Helper()
	tt := NewTester(t, d, 400, 200)
	return tt, tt.Find(ByRole(ui.RoleStepper))
}

var days = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

func TestDomainUpDownShowsTheSelectedItem(t *testing.T) {
	d := DomainUpDown(days...).SetSelectedItem("Wed")
	tt, _ := domainTester(t, d)
	if !tt.Has(ByLabel("Wed")) || tt.Has(ByLabel("Tue")) || d.SelectedIndex() != 2 || d.SelectedItem() != "Wed" {
		t.Fatalf("index %d item %q texts %q", d.SelectedIndex(), d.SelectedItem(), tt.Texts())
	}
}

// As in WinForms, Up goes to the item before, Down to the next one.
func TestDomainUpDownUpIsThePreviousItemAndDownTheNext(t *testing.T) {
	changed := make(chan string, 8)
	var d *DomainUpDownWidget
	d = DomainUpDown(days...).SetSelectedIndex(2).OnChange(func() { changed <- d.SelectedItem() })
	tt, s := domainTester(t, d)
	s.Key(0, ui.KeyUp)
	if got := received(t, changed); got != "Tue" || d.SelectedIndex() != 1 {
		t.Fatalf("Up: OnChange saw %q, index %d", got, d.SelectedIndex())
	}
	s.Key(0, ui.KeyDown)
	s.Key(0, ui.KeyDown)
	received(t, changed)
	if got := received(t, changed); got != "Thu" {
		t.Fatalf("Down twice from Tue: %q", got)
	}
	if !tt.Has(ByLabel("Thu")) {
		t.Fatalf("the field still shows %q", tt.Texts())
	}
}

func TestDomainUpDownStopsAtTheEndsUnlessItWraps(t *testing.T) {
	d := DomainUpDown(days...)
	_, s := domainTester(t, d)
	s.Key(0, ui.KeyUp)
	if d.SelectedIndex() != 0 {
		t.Fatalf("Up from the first item went to %d", d.SelectedIndex())
	}
	d.SetWrap(true)
	s.Key(0, ui.KeyUp)
	if d.SelectedItem() != "Fri" {
		t.Fatalf("Up from the first item with Wrap gave %q", d.SelectedItem())
	}
	s.Key(0, ui.KeyDown)
	if d.SelectedItem() != "Mon" {
		t.Fatalf("Down from the last item with Wrap gave %q", d.SelectedItem())
	}
}

func TestDomainUpDownSettersDoNotReportAndIgnoreWhatIsNotThere(t *testing.T) {
	changed := make(chan struct{}, 2)
	d := DomainUpDown(days...).OnChange(func() { changed <- struct{}{} })
	tt, _ := domainTester(t, d)
	d.SetSelectedItem("Fri")
	d.SetSelectedItem("Sun") // not an item
	tt.Settle()
	if d.SelectedItem() != "Fri" || !tt.Has(ByLabel("Fri")) {
		t.Fatalf("item %q", d.SelectedItem())
	}
	d.SetSelectedIndex(99) // clamps
	if d.SelectedIndex() != 4 {
		t.Fatalf("index %d", d.SelectedIndex())
	}
	select {
	case <-changed:
		t.Fatal("a setter ran OnChange")
	default:
	}
}

func TestDomainUpDownSetItemsKeepsTheSelectedItemIfItStays(t *testing.T) {
	d := DomainUpDown(days...).SetSelectedItem("Wed")
	d.SetItems("Tue", "Wed", "Sat")
	if d.SelectedItem() != "Wed" || d.SelectedIndex() != 1 {
		t.Fatalf("item %q index %d", d.SelectedItem(), d.SelectedIndex())
	}
	d.SetItems("A", "B")
	if d.SelectedIndex() != 0 {
		t.Fatalf("index %d after the item left", d.SelectedIndex())
	}
	d.SetItems()
	if d.SelectedIndex() != -1 || d.SelectedItem() != "" {
		t.Fatalf("empty list: index %d item %q", d.SelectedIndex(), d.SelectedItem())
	}
}

func TestDomainUpDownBindsTheItem(t *testing.T) {
	v := "Thu"
	d := Bind(DomainUpDown(days...), func() string { return v }, func(s string) { v = s })
	tt, s := domainTester(t, d)
	if !tt.Has(ByLabel("Thu")) {
		t.Fatalf("shows %q, not the source's Thu", tt.Texts())
	}
	s.Key(0, ui.KeyDown)
	if v != "Fri" {
		t.Fatalf("the source got %q", v)
	}
}
