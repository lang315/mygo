package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// threeSections is an accordion of three sections, each holding one label.
func threeSections() *AccordionWidget {
	return Accordion().
		AddSection("General", Label("Name field")).
		AddSection("Privacy", Label("Cookies field"), Label("History field")).
		AddSection("Advanced", Label("Proxy field"))
}

func TestAccordionOneOpen(t *testing.T) {
	changes := make(chan int, 4)
	var a *AccordionWidget
	a = threeSections().OnChange(func() { changes <- a.Open() })
	tt := NewTester(t, a, 400, 600)
	if a.Open() != -1 {
		t.Fatalf("a new accordion has %d open, want -1", a.Open())
	}
	a.SetOpen(1)
	closeRuns(tt)
	if a.Open() != 1 || !tt.HasText("Cookies field") || !tt.HasText("History field") {
		t.Fatalf("SetOpen(1): Open %d, texts %q", a.Open(), tt.Texts())
	}
	header(tt, "General").Click()
	closeRuns(tt)
	if got := received(t, changes); got != 0 || a.Open() != 0 {
		t.Fatalf("a click on General's header: OnChange saw %d, Open %d, want 0", got, a.Open())
	}
	if !tt.HasText("Name field") {
		t.Fatalf("section 0's controls are not built: %q", tt.Texts())
	}
	for _, s := range []string{"Cookies field", "History field", "Proxy field"} {
		if tt.HasText(s) {
			t.Fatalf("%s is built while its section is closed: %q", s, tt.Texts())
		}
	}
}

func TestAccordionClickOpenCloses(t *testing.T) {
	changes := make(chan int, 4)
	var a *AccordionWidget
	a = threeSections().SetOpen(2).OnChange(func() { changes <- a.Open() })
	tt := NewTester(t, a, 400, 600)
	header(tt, "Advanced").Click()
	closeRuns(tt)
	if got := received(t, changes); got != -1 || a.Open() != -1 {
		t.Fatalf("a click on the open header: OnChange saw %d, Open %d, want -1", got, a.Open())
	}
	if tt.HasText("Proxy field") {
		t.Fatalf("the closed section is still built: %q", tt.Texts())
	}
}

func TestAccordionSetOpenOutOfRangeIgnored(t *testing.T) {
	a := threeSections().SetOpen(1)
	tt := NewTester(t, a, 400, 600)
	for _, i := range []int{3, 99, -2} {
		a.SetOpen(i)
		tt.Settle()
		if a.Open() != 1 {
			t.Fatalf("SetOpen(%d) changed Open to %d, want 1", i, a.Open())
		}
	}
	a.SetOpen(-1)
	closeRuns(tt)
	if a.Open() != -1 || tt.HasText("Cookies field") {
		t.Fatalf("SetOpen(-1): Open %d, texts %q", a.Open(), tt.Texts())
	}
}

// Added: Review focus 3.
func TestAccordionSetterDoesNotFireOnChange(t *testing.T) {
	changes := make(chan struct{}, 4)
	a := threeSections().OnChange(func() { changes <- struct{}{} })
	tt := NewTester(t, a, 400, 600)
	a.SetOpen(0)
	closeRuns(tt)
	a.SetOpen(-1)
	closeRuns(tt)
	if len(changes) != 0 {
		t.Fatal("SetOpen ran OnChange")
	}
}

// A section's radio buttons are one group, as in a GroupBox.
func TestAccordionSectionRadiosAreOneGroup(t *testing.T) {
	r1, r2 := RadioButton("Small").SetChecked(true), RadioButton("Large")
	a := Accordion().AddSection("Size", r1, r2).SetOpen(0)
	tt := NewTester(t, a, 400, 600)
	tt.Find(ByLabel("Large").Role(ui.RoleRadio)).Click()
	if r1.Checked() || !r2.Checked() {
		t.Fatalf("Small %v, Large %v after a click on Large", r1.Checked(), r2.Checked())
	}
}
