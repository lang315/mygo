package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

var testFamilies = []ui.FontFamily{{Name: "Alpha"}, {Name: "Beta Mono", Monospaced: true}, {Name: "Gamma"}}

func newFontDialog() *FontDialogWidget {
	d := FontDialog()
	d.families = func() []ui.FontFamily { return testFamilies }
	return d
}

func runFont(t *testing.T, fd *FontDialogWidget) (*DialogTester, chan bool) {
	t.Helper()
	d, run := fd.build(nil)
	dt := TestDialog(t, d)
	done := make(chan bool, 1)
	go func() { done <- run() }()
	<-posted()
	return dt, done
}

func TestFontDialogOKReturnsTheChosenFont(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "Alpha", Size: 12, Features: "liga"})
	dt, done := runFont(t, fd)
	dt.Find(ByRole(ui.RoleListItem).Label("Gamma")).Click()
	dt.Find(ByRole(ui.RoleListItem).Label("Bold Italic")).Click()
	dt.Find(ByRole(ui.RoleTextField)).Focus()
	dt.Tester.Tester.Command("selectAll")
	dt.Find(ByRole(ui.RoleTextField)).Type("20").Key(0, ui.KeyEnter)
	dt.Find(ByLabel("OK")).Click()
	if !<-done {
		t.Fatal("OK answered false")
	}
	want := ui.Font{Family: "Gamma", Size: 20, Weight: 700, Italic: true, Features: "liga"}
	if got := fd.Font(); got != want {
		t.Errorf("Font = %+v, want %+v", got, want)
	}
}

func TestFontDialogCancelLeavesTheFontAlone(t *testing.T) {
	start := ui.Font{Family: "Alpha", Size: 12}
	fd := newFontDialog().SetFont(start)
	dt, done := runFont(t, fd)
	dt.Find(ByRole(ui.RoleListItem).Label("Gamma")).Click()
	dt.Find(ByLabel("Cancel")).Click()
	if <-done || fd.Font() != start {
		t.Errorf("Cancel: Font = %+v", fd.Font())
	}
}

func TestFontDialogKeepsTheWeightWhenTheStyleDoesNotChange(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "Alpha", Size: 12, Weight: 300})
	dt, done := runFont(t, fd)
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Weight; got != 300 {
		t.Errorf("Weight = %d, want 300 as set", got)
	}
}

func TestFontDialogShowsASetFamilyTheSystemLacks(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "Zeta"})
	dt, done := runFont(t, fd)
	if !dt.Has(ByRole(ui.RoleListItem).Label("Zeta")) {
		t.Error("the family SetFont gave is not listed")
	}
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font(); got.Family != "Zeta" || got.Size != 14 {
		t.Errorf("Font = %+v, want Zeta at 14 (the size when 0)", got)
	}
}

func TestFontDialogFixedPitchOnlyListsMonospacedFamilies(t *testing.T) {
	fd := newFontDialog().SetFixedPitchOnly(true)
	dt, done := runFont(t, fd)
	if !dt.Has(ByRole(ui.RoleListItem).Label("Beta Mono")) || dt.Has(ByRole(ui.RoleListItem).Label("Alpha")) || dt.Has(ByRole(ui.RoleListItem).Label("Gamma")) {
		t.Error("FixedPitchOnly lists a family that is not monospaced, or hides one that is")
	}
	dt.Find(ByLabel("Cancel")).Click()
	<-done
}

func TestFontDialogTakesNoTypedSizeOutsideMinAndMax(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "Alpha", Size: 12}).SetMinSize(8).SetMaxSize(40)
	dt, done := runFont(t, fd)
	dt.Find(ByRole(ui.RoleTextField)).Focus()
	dt.Tester.Tester.Command("selectAll")
	dt.Find(ByRole(ui.RoleTextField)).Type("500").Key(0, ui.KeyEnter)
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Size; got != 12 {
		t.Errorf("Size = %v, want 12: 500 is above MaxSize", got)
	}
}
