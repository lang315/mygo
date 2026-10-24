package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// runColor opens the dialog on another goroutine, as a handler would, and
// returns its tester and a channel of the answer.
func runColor(t *testing.T, cd *ColorDialogWidget) (*DialogTester, chan bool) {
	t.Helper()
	d, run := cd.build(nil)
	dt := TestDialog(t, d)
	done := make(chan bool, 1)
	go func() { done <- run() }()
	<-posted()
	return dt, done
}

// typeHex replaces the picker's hex text with s.
func typeHex(dt *DialogTester, s string) {
	hex := dt.Find(ByRole(ui.RoleTextField).Label("Hex")).Focus()
	dt.Tester.Tester.Command("selectAll")
	hex.Type(s).Key(0, ui.KeyEnter)
}

func TestColorDialogOKReturnsTheChosenColor(t *testing.T) {
	cd := ColorDialog().SetColor(ui.Hex("#2563eb"))
	dt, done := runColor(t, cd)
	hex := dt.Find(ByRole(ui.RoleTextField).Label("Hex"))
	if hex.Value() != "#2563EB" && hex.Value() != "#2563eb" {
		t.Fatalf("the picker shows %q, not the dialog's color", hex.Value())
	}
	typeHex(dt, "#112233")
	dt.Find(ByLabel("OK")).Click()
	if !<-done {
		t.Fatal("OK answered false")
	}
	if got := cd.Color(); got != ui.Hex("#112233") {
		t.Fatalf("Color %v, want the hex typed", got)
	}
}

func TestColorDialogCancelKeepsTheColorTheDialogHad(t *testing.T) {
	cd := ColorDialog().SetColor(ui.Hex("#2563eb"))
	dt, done := runColor(t, cd)
	typeHex(dt, "#112233")
	dt.Find(ByLabel("Cancel")).Click()
	if <-done {
		t.Fatal("Cancel answered true")
	}
	if cd.Color() != ui.Hex("#2563eb") {
		t.Fatalf("Color %v after Cancel", cd.Color())
	}
}

func TestColorDialogChoosingASwatchThenOKReturnsIt(t *testing.T) {
	cd := ColorDialog().SetColor(ui.Hex("#000000"))
	dt, done := runColor(t, cd)
	swatches := dt.FindAll(ByRole(ui.RoleButton))
	var pick *Node
	for _, s := range swatches {
		if l := s.Label(); l != "OK" && l != "Cancel" && l != "" {
			pick = s
			break
		}
	}
	if pick == nil {
		t.Fatalf("no swatch among %d buttons", len(swatches))
	}
	pick.Click()
	dt.Find(ByLabel("OK")).Click()
	if !<-done || cd.Color() == ui.Hex("#000000") {
		t.Fatalf("Color %v after choosing %q", cd.Color(), pick)
	}
}

func TestColorDialogEscapeIsCancel(t *testing.T) {
	cd := ColorDialog().SetColor(ui.Hex("#123456"))
	dt, done := runColor(t, cd)
	typeHex(dt, "#654321")
	dt.Escape()
	if <-done || cd.Color() != ui.Hex("#123456") {
		t.Fatalf("Escape: color %v", cd.Color())
	}
}

func TestColorDialogShowsItsTitleAndTakesANewOne(t *testing.T) {
	cd := ColorDialog().SetTitle("Pick a tint")
	dt, done := runColor(t, cd)
	if !dt.Has(ByLabel("Pick a tint")) {
		t.Fatalf("texts %q", dt.Texts())
	}
	dt.Find(ByLabel("Cancel")).Click()
	<-done
}

// WinForms' colors have no alpha: the opacity slider is ignored.
func TestColorDialogOKReturnsAnOpaqueColor(t *testing.T) {
	cd := ColorDialog()
	dt, done := runColor(t, cd)
	typeHex(dt, "#11223380")
	dt.Find(ByLabel("OK")).Click()
	<-done
	if got := cd.Color(); got != ui.Hex("#112233") {
		t.Fatalf("Color %v: a half-opaque pick must come back opaque", got)
	}
}
