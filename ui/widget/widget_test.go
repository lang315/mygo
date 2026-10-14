package widget

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestBackendInstalled(t *testing.T) {
	n := len(fb.Windows())
	w := mygo.NewWindow(mygo.WindowOptions{Title: "probe"})
	defer w.Destroy()
	if got := len(fb.Windows()); got != n+1 {
		t.Fatalf("fake windows %d, want %d: platform.TestBackend not used", got, n+1)
	}
}

func TestLabelShowsText(t *testing.T) {
	tt := tester(t, Label("Hello"))
	if !tt.HasText("Hello") {
		t.Fatalf("no Hello in %q", tt.Texts())
	}
}

func TestSetTextFromAnotherGoroutine(t *testing.T) {
	l := Label("a")
	tt := tester(t, l)
	done := make(chan struct{})
	go func() { l.SetText("b"); close(done) }()
	<-done
	tt.Frame()
	if !tt.HasText("b") || l.Text() != "b" {
		t.Fatalf("texts %q, Text() %q, want b", tt.Texts(), l.Text())
	}
}

func TestHiddenWidgetIsNotShown(t *testing.T) {
	l := Label("gone")
	tt := tester(t, l)
	l.SetVisible(false)
	tt.Frame()
	if tt.HasText("gone") || l.Visible() {
		t.Fatalf("hidden label shown: %q", tt.Texts())
	}
	l.SetVisible(true)
	tt.Frame()
	if !tt.HasText("gone") {
		t.Fatalf("label not shown again: %q", tt.Texts())
	}
}

func TestAssignTo(t *testing.T) {
	var l *LabelWidget
	got := Label("x").AssignTo(&l)
	if l == nil || l != got || l.Text() != "x" {
		t.Fatalf("AssignTo stored %v, returned %v", l, got)
	}
}

// Review Focus 3: widgets live before any window or app.
func TestWidgetsWorkWithoutAWindow(t *testing.T) {
	l := Label("a").SetText("b").SetTooltip("tip").SetDisabled(true)
	if l.Text() != "b" {
		t.Fatalf("Text() %q, want b", l.Text())
	}
}

func TestOnClickOnALabel(t *testing.T) {
	clicked := make(chan bool, 1)
	tt := tester(t, Label("tap").OnClick(func() { clicked <- onEvents() }))
	if err := tt.Click("tap"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !wait(t, clicked) {
		t.Fatal("OnClick ran off the event goroutine")
	}
}

func TestRichLabel(t *testing.T) {
	l := Label("plain").Rich(ui.Span{Text: "Hello "}, ui.Span{Text: "world", Weight: 700})
	tt := tester(t, l)
	if l.Text() != "Hello world" || !tt.HasText("world") {
		t.Fatalf("Text() %q, texts %q", l.Text(), tt.Texts())
	}
	l.SetText("plain again")
	tt.Frame()
	if !tt.HasText("plain again") {
		t.Fatalf("SetText after Rich: %q", tt.Texts())
	}
}
