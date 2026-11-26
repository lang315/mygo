package widget

import (
	"slices"
	"strconv"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

func TestProgressPanelShowsCaptionAndDescription(t *testing.T) {
	p := ProgressPanel("Loading").SetDescription("Reading the files")
	tt := NewTester(t, p, 400, 300)
	if !tt.Has(ByLabel("Loading")) || !tt.Has(ByLabel("Reading the files")) {
		t.Fatalf("the panel shows %q", tt.Texts())
	}
	p.SetCaption("Saving").SetDescription("")
	tt.Settle()
	texts := tt.Texts()
	if !slices.Contains(texts, "Saving") || slices.Contains(texts, "Loading") || slices.Contains(texts, "Reading the files") {
		t.Fatalf("after SetCaption and an empty SetDescription the panel shows %q", texts)
	}
}

// waitForm returns a Tester of a form whose content is a button counting its
// clicks into clicks.
func waitForm(t *testing.T, clicks *int) (*Form, *Tester, *ButtonWidget) {
	t.Helper()
	b := Button("Go").OnClick(func() { *clicks++ })
	f := NewForm("Wait", 500, 400)
	f.SetContent(VBox(8).Add(b, TextBox()))
	return f, TestForm(t, f), b
}

func TestShowWaitBlocksInput(t *testing.T) {
	clicks := 0
	f, tt, _ := waitForm(t, &clicks)
	r := tt.Find(ByLabel("Go")).Bounds()
	f.ShowWait("Loading", "Please wait")
	tt.Settle()
	if !tt.Has(ByLabel("Loading")) || !tt.Has(ByLabel("Please wait")) {
		t.Fatalf("ShowWait shows %q", tt.Texts())
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Settle()
	if clicks != 0 {
		t.Fatal("a click reached the content's button while waiting")
	}
	f.CloseWait()
	tt.Settle()
	if tt.Has(ByLabel("Loading")) {
		t.Fatal("CloseWait left the panel")
	}
	tt.Find(ByLabel("Go")).Click()
	if clicks != 1 {
		t.Fatalf("after CloseWait a click ran OnClick %d times", clicks)
	}
}

// Keys do not reach the content either: the focused button, the form's
// shortcuts and its accept button wait.
func TestShowWaitBlocksKeys(t *testing.T) {
	clicks, shortcuts := 0, 0
	f, tt, b := waitForm(t, &clicks)
	f.Shortcut(0, ui.KeyF5, func() { shortcuts++ })
	f.SetAcceptButton(b)
	tt.Find(ByLabel("Go")).Focus()
	f.ShowWait("Loading", "")
	tt.Settle()
	for _, k := range []ui.Key{ui.KeySpace, ui.KeyEnter, ui.KeyF5} {
		tt.Key(0, k)
		tt.Settle()
	}
	if clicks != 0 || shortcuts != 0 {
		t.Fatalf("while waiting, keys clicked the button %d times and ran the shortcut %d times", clicks, shortcuts)
	}
	f.CloseWait()
	tt.Settle()
	tt.Key(0, ui.KeyF5)
	tt.Settle()
	if shortcuts != 1 {
		t.Fatalf("after CloseWait F5 ran the shortcut %d times", shortcuts)
	}
}

// What Modal adds to the box that covers the window: with the focus in a
// TextBox, a form shortcut, Enter on the accept button and typing all wait
// until CloseWait, then work again.
func TestShowWaitHoldsShortcutsAcceptAndTyping(t *testing.T) {
	saves, accepts := 0, 0
	box := TextBox()
	ok := Button("OK").OnClick(func() { accepts++ })
	f := NewForm("Wait", 500, 400)
	f.SetContent(VBox(8).Add(box, ok))
	f.Shortcut(ui.Ctrl, ui.KeyS, func() { saves++ })
	f.SetAcceptButton(ok)
	tt := TestForm(t, f)
	field := tt.Find(ByRole(ui.RoleTextField)).Focus()
	f.ShowWait("Saving", "")
	tt.Settle()
	tt.Key(ui.Ctrl, ui.KeyS)
	tt.Key(0, ui.KeyEnter)
	tt.Type("x")
	tt.Settle()
	if saves != 0 || accepts != 0 || box.Text() != "" {
		t.Fatalf("while waiting: Ctrl+S ran %d times, Enter clicked OK %d times, the box has %q", saves, accepts, box.Text())
	}
	f.CloseWait()
	tt.Settle()
	field.Focus()
	tt.Key(ui.Ctrl, ui.KeyS)
	tt.Type("x")
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if saves != 1 || accepts != 1 || box.Text() != "x" {
		t.Fatalf("after CloseWait: Ctrl+S ran %d times, Enter clicked OK %d times, the box has %q", saves, accepts, box.Text())
	}
}

func TestShowWaitReplacesCaption(t *testing.T) {
	clicks := 0
	f, tt, _ := waitForm(t, &clicks)
	f.ShowWait("Loading", "Step 1")
	tt.Settle()
	f.ShowWait("Saving", "Step 2")
	tt.Settle()
	texts := tt.Texts()
	if count(tt.Tester, "Saving") != 1 || count(tt.Tester, "Step 2") != 1 {
		t.Fatalf("a second ShowWait shows %q: want one Saving, one Step 2", texts)
	}
	if slices.Contains(texts, "Loading") || slices.Contains(texts, "Step 1") {
		t.Fatalf("the first ShowWait's text stayed: %q", texts)
	}
	if n := len(tt.FindAll(ByRole(ui.RoleStatus))); n != 1 {
		t.Fatalf("%d wait panels, want 1", n)
	}
}

func TestCloseWaitWithoutShowIsNoop(t *testing.T) {
	NewForm("Never shown", 100, 100).CloseWait()
	clicks := 0
	f, tt, _ := waitForm(t, &clicks)
	f.CloseWait()
	f.CloseWait()
	tt.Settle()
	tt.Find(ByLabel("Go")).Click()
	if clicks != 1 {
		t.Fatalf("after CloseWait alone a click ran OnClick %d times", clicks)
	}
	if n := len(tt.FindAll(ByRole(ui.RoleStatus))); n != 0 {
		t.Fatalf("%d wait panels after CloseWait alone", n)
	}
}

func TestShowWaitFromGoroutine(t *testing.T) {
	clicks := 0
	f, tt, _ := waitForm(t, &clicks)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 400 {
			f.ShowWait("Step "+strconv.Itoa(i), "of 400")
			if i%3 == 0 {
				f.CloseWait()
			}
		}
		f.ShowWait("Done", "")
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	if !tt.Has(ByLabel("Done")) {
		t.Fatalf("the last ShowWait does not show: %q", tt.Texts())
	}
}

// A form whose wait closed allocates no more in a frame than one that never
// waited, and a ProgressPanel's frame no more than its Spinner's.
func TestProgressPanelAllocNothingAtRest(t *testing.T) {
	calmOff()
	never := NewForm("Rest", 400, 300).SetContent(Label("Hello"))
	waited := NewForm("Rest", 400, 300).SetContent(Label("Hello"))
	plain, closed := ui.NewTester(never.view, 400, 300), ui.NewTester(waited.view, 400, 300)
	waited.ShowWait("Loading", "")
	closed.Frame()
	waited.CloseWait()
	spin := ui.NewTester(content(VBox(4).Add(Spinner().Start(), Label("Loading"), Label("Please wait"))), 400, 300)
	panel := ui.NewTester(content(ProgressPanel("Loading").SetDescription("Please wait")), 400, 300)
	for range 10 {
		plain.Frame()
		closed.Frame()
		spin.Frame()
		panel.Frame()
	}
	if p, c := allocrun.Least(30, plain.Frame), allocrun.Least(30, closed.Frame); c > p {
		t.Fatalf("a frame of a form whose wait closed allocates %v, one that never waited %v", c, p)
	}
	want, got := allocrun.Least(30, spin.Frame), allocrun.Least(30, panel.Frame)
	if got > want {
		t.Fatalf("a frame of a ProgressPanel allocates %v, more than a Spinner and two labels' %v", got, want)
	}
	t.Logf("a ProgressPanel's frame allocates %v, a Spinner and two labels' %v", got, want)
}
