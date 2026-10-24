package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// runColor opens the dialog on another goroutine, as a handler would, and
// returns its tester and a channel of the answer.
func runColor(t *testing.T, cd *ColorDialogWidget) (*DialogTester, chan bool) {
	t.Helper()
	d := cd.build(nil)
	dt := TestDialog(t, d)
	done := make(chan bool, 1)
	go func() { done <- cd.run(d) }()
	<-posted()
	return dt, done
}

func TestColorDialogOKReturnsTheChosenColor(t *testing.T) {
	cd := ColorDialog().SetColor(ui.Hex("#2563eb"))
	dt, done := runColor(t, cd)
	hex := dt.Find(ByRole(ui.RoleTextField).Label("Hex"))
	if hex.Value() != "#2563EB" && hex.Value() != "#2563eb" {
		t.Fatalf("the picker shows %q, not the dialog's color", hex.Value())
	}
	dt.Find(ByLabel("OK")).Click()
	if !<-done {
		t.Fatal("OK answered false")
	}
	if got := cd.Color(); got != ui.Hex("#2563eb") {
		t.Fatalf("Color %v", got)
	}
}

func TestColorDialogCancelKeepsTheColorTheDialogHad(t *testing.T) {
	cd := ColorDialog().SetColor(ui.Hex("#2563eb"))
	dt, done := runColor(t, cd)
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
