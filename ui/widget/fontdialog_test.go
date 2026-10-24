package widget

import (
	"fmt"
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
	dt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	dt.Tester.Tester.Command("selectAll")
	dt.Find(ByRole(ui.RoleTextField).Nth(1)).Type("20").Key(0, ui.KeyEnter)
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
	dt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	dt.Tester.Tester.Command("selectAll")
	dt.Find(ByRole(ui.RoleTextField).Nth(1)).Type("500").Key(0, ui.KeyEnter)
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Size; got != 12 {
		t.Errorf("Size = %v, want 12: 500 is above MaxSize", got)
	}
}

func TestFontDialogListsAFamilyOnceWhateverTheCaseOfSetFont(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "alpha", Size: 12})
	dt, done := runFont(t, fd)
	if dt.Has(ByRole(ui.RoleListItem).Label("alpha")) || !dt.Has(ByRole(ui.RoleListItem).Label("Alpha")) {
		t.Error("the family is listed twice, or not in the system's spelling")
	}
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Family; got != "Alpha" {
		t.Errorf("Family = %q, want the system's spelling Alpha", got)
	}
}

func TestFontDialogBringsTheStartSizeInsideMinAndMax(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "Alpha", Size: 500}).SetMaxSize(40)
	dt, done := runFont(t, fd)
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Size; got != 40 {
		t.Errorf("Size = %v, want 40", got)
	}
}

func TestFontDialogEscapeIsCancel(t *testing.T) {
	start := ui.Font{Family: "Alpha", Size: 12}
	fd := newFontDialog().SetFont(start)
	dt, done := runFont(t, fd)
	dt.Find(ByRole(ui.RoleListItem).Label("Gamma")).Click()
	dt.Escape()
	if <-done || fd.Font() != start {
		t.Errorf("Escape: Font = %+v", fd.Font())
	}
}

func TestFontDialogMinAndMaxMoveEachOther(t *testing.T) {
	for _, c := range []struct {
		name     string
		set      func(*FontDialogWidget)
		wantSize float32
	}{
		{"min above max", func(d *FontDialogWidget) { d.SetMaxSize(8).SetMinSize(40) }, 40},
		{"max below min", func(d *FontDialogWidget) { d.SetMinSize(40).SetMaxSize(8) }, 8},
	} {
		fd := newFontDialog().SetFont(ui.Font{Family: "Alpha", Size: 20})
		c.set(fd)
		dt, done := runFont(t, fd)
		dt.Find(ByLabel("OK")).Click()
		<-done
		if got := fd.Font().Size; got != c.wantSize {
			t.Errorf("%s: Size = %v, want %v", c.name, got, c.wantSize)
		}
	}
}

func TestFontDialogFixedPitchOnlyStartsOnAFixedPitchFamilyWhenTheFontIsProportional(t *testing.T) {
	fd := newFontDialog().SetFixedPitchOnly(true).SetFont(ui.Font{Family: "Alpha", Size: 12})
	dt, done := runFont(t, fd)
	if dt.Has(ByRole(ui.RoleListItem).Label("Alpha")) {
		t.Error("a proportional family was added to a fixed-pitch list")
	}
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Family; got != "Beta Mono" {
		t.Errorf("Family = %q, want the first fixed-pitch family", got)
	}
}

func TestFontDialogChoosesTheWeightItShowsAndKeepsWhatItDoesNotChange(t *testing.T) {
	for _, c := range []struct {
		start, now   bool
		weight, want int
	}{
		{false, false, 300, 300}, {true, true, 600, 600}, {false, true, 300, 700}, {true, false, 600, 400},
	} {
		if got := chosenWeight(c.weight, c.start, c.now); got != c.want {
			t.Errorf("start bold %v, now %v, weight %d: %d, want %d", c.start, c.now, c.weight, got, c.want)
		}
	}
}

func TestFontDialogShowsTheSampleTextItIsGiven(t *testing.T) {
	fd := newFontDialog().SetSampleText("Nguyễn Văn Đức")
	dt, done := runFont(t, fd)
	if !dt.Has(ByLabel("Nguyễn Văn Đức")) || dt.Has(ByLabel("AaBbYyZz")) {
		t.Errorf("texts %q", dt.Texts())
	}
	dt.Find(ByLabel("Cancel")).Click()
	<-done
}

func manyFamilies() []ui.FontFamily {
	var many []ui.FontFamily
	for i := range 300 {
		many = append(many, ui.FontFamily{Name: fmt.Sprintf("Fam%03d", i)})
	}
	return many
}

func TestFontDialogScrollsToTheCurrentFamilyWhenItOpens(t *testing.T) {
	fd := FontDialog().SetFont(ui.Font{Family: "Fam290", Size: 12})
	fd.families = manyFamilies
	dt, done := runFont(t, fd)
	dt.Settle()
	shown := func(label string) bool {
		if !dt.Has(ByRole(ui.RoleListItem).Label(label)) {
			return false
		}
		b := dt.Find(ByRole(ui.RoleListItem).Label(label)).Bounds()
		return b.Y >= 0 && b.Y+b.H <= 460
	}
	if !shown("Fam290") || shown("Fam000") {
		t.Errorf("on screen at open: Fam290 %v, Fam000 %v", shown("Fam290"), shown("Fam000"))
	}
	dt.Find(ByLabel("Cancel")).Click()
	<-done
}

func TestFontDialogTypeAheadJumpsToTheFirstFamilyThatStartsWithWhatIsTyped(t *testing.T) {
	fd := FontDialog().SetFont(ui.Font{Family: "Fam001", Size: 12})
	fd.families = manyFamilies
	dt, done := runFont(t, fd)
	box := dt.Find(ByRole(ui.RoleTextField).Nth(0))
	box.Focus()
	dt.Tester.Tester.Command("selectAll")
	box.Type("fam25")
	if b := dt.Find(ByRole(ui.RoleListItem).Label("Fam250")).Bounds(); b.Y < 0 || b.Y+b.H > 460 {
		t.Errorf("the list did not scroll to the match: %v", b)
	}
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Family; got != "Fam250" {
		t.Errorf("Family = %q, want Fam250", got)
	}
}

func TestFontDialogTakesATypedSizeWithoutEnter(t *testing.T) {
	fd := newFontDialog().SetFont(ui.Font{Family: "Alpha", Size: 12})
	dt, done := runFont(t, fd)
	dt.Find(ByRole(ui.RoleTextField).Nth(1)).Focus()
	dt.Tester.Tester.Command("selectAll")
	dt.Find(ByRole(ui.RoleTextField).Nth(1)).Type("20")
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := fd.Font().Size; got != 20 {
		t.Errorf("Size = %v, want 20", got)
	}
}
