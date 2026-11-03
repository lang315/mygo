package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func tourTableTab(t *testing.T) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(widget.View(tour(nil)), 1000, 700)
	tt.Frame()
	if err := tt.Click("DataGridView"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	return tt
}

// The Name column of the tour's table edits in place: F2 opens the editor,
// typing replaces the text and Enter writes it.
func TestTourTableEditsAName(t *testing.T) {
	tt := tourTableTab(t)
	if err := tt.Click("acct000000"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Type("Nguyễn")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Nguyễn") {
		t.Fatalf("the edited name is not shown; texts %q", tt.Texts())
	}
}

// An empty name is refused: the editor stays open with the message.
func TestTourTableRefusesAnEmptyName(t *testing.T) {
	tt := tourTableTab(t)
	if err := tt.Click("acct000000"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyF2)
	tt.Frame()
	tt.Key(0, ui.KeyBackspace) // the text starts selected
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("A name cannot be empty") {
		t.Fatalf("no refusal shown; texts %q", tt.Texts())
	}
}
