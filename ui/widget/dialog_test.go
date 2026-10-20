package widget

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

// shownParent shows a window to be a dialog's parent.
func shownParent(t *testing.T, title string) *Window {
	t.Helper()
	p := NewWindow(title, 600, 400)
	p.Show()
	t.Cleanup(p.Close)
	return p
}

// Run on a worker goroutine waits; the buttons work.
func TestDialogRunOnAWorkerGoroutine(t *testing.T) {
	p := shownParent(t, "P1")
	var d *Dialog
	d = NewDialog(p, "Ask", "sub", 300, 200, Button("Done").OnClick(func() { d.Close() }))
	ran := runAsync(func() bool { d.Run(); return true })
	fw := waitFake(t, "Ask")
	if !fw.Opts.Modal || fw.Opts.Parent == nil {
		t.Fatalf("dialog not a modal child: Modal %v, Parent %v", fw.Opts.Modal, fw.Opts.Parent)
	}
	press(t, fw, "Done")
	received(t, ran)
}

func TestDialogRunInAHandlerPumpsHandlers(t *testing.T) {
	p := shownParent(t, "P2")
	var d *Dialog
	d = NewDialog(p, "Pump", "", 300, 200, Button("Done").OnClick(func() { d.Close() }))
	ran := runAsync(func() bool { Update(d.Run); return true })
	fw := waitFake(t, "Pump")
	press(t, fw, "Done") // its OnClick runs while Run blocks the event goroutine
	received(t, ran)
}

func TestDialogShowsItsTitleInside(t *testing.T) {
	d := NewDialog(nil, "Titled", "Subtitled", 300, 200, Label("body"))
	go Update(d.Run)
	fw := waitFake(t, "Titled")
	press(t, fw, "Subtitled") // drawn in the content; pressing a label is harmless
	d.Close()
	waitClosed(t, d.Window)
}

func TestUndeletableDialogStaysOpen(t *testing.T) {
	var d *Dialog
	d = NewDialog(nil, "Stay", "", 300, 200, Button("Done").OnClick(func() { d.Close() }))
	d.SetDeletable(false)
	ran := runAsync(func() bool { d.Run(); return true })
	fw := waitFake(t, "Stay")
	mygo.RunOnMain(func() { fw.H.ShouldClose() })
	Update(func() {})
	if fw.IsClosed() {
		t.Fatal("an undeletable dialog closed on the user's request")
	}
	press(t, fw, "Done")
	received(t, ran)
}

func TestOnBeforeRunRunsFirst(t *testing.T) {
	var order []string
	var d *Dialog
	d = NewDialog(nil, "Before", "", 300, 200, Label("x")).OnBeforeRun(func() {
		order = append(order, "before")
		go func() { waitFake(t, "Before"); d.Close() }()
	})
	d.Run()
	order = append(order, "after")
	if !slices.Equal(order, []string{"before", "after"}) {
		t.Fatalf("order %v", order)
	}
}

// Closing the parent ends every nested Run, inner first.
func TestClosingTheParentEndsNestedRuns(t *testing.T) {
	p := shownParent(t, "P3")
	var order []string
	outer := NewDialog(p, "Outer", "", 400, 300, Label("outer"))
	inner := NewDialog(outer.Window, "Inner", "", 300, 200, Label("inner"))
	outer.OnBeforeRun(func() {
		Post(func() { inner.Run(); order = append(order, "inner") })
	})
	ran := runAsync(func() bool {
		Update(func() { outer.Run(); order = append(order, "outer") })
		return true
	})
	waitFake(t, "Inner")
	p.Close()
	received(t, ran)
	Update(func() {})
	if !slices.Equal(order, []string{"inner", "outer"}) {
		t.Fatalf("order %v", order)
	}
}

func TestDialogRunOnTheMainThreadPanics(t *testing.T) {
	d := NewDialog(nil, "Main", "", 300, 200, Label("x"))
	var msg string
	mygo.RunOnMain(func() {
		defer func() { msg, _ = recover().(string) }()
		d.Run()
	})
	if !strings.Contains(msg, "main thread") {
		t.Fatalf("panic %q, want one naming the main thread", msg)
	}
	if d.Native() != nil {
		t.Fatal("the dialog showed before panicking")
	}
}

// An undeletable dialog does not stop the app's quit.
func TestUndeletableDialogLetsTheAppQuit(t *testing.T) {
	d := NewDialog(nil, "Busy", "", 300, 200, Label("working"))
	d.SetDeletable(false)
	d.Show()
	defer d.Close()
	fw := waitFake(t, "Busy")
	var closes bool
	mygo.RunOnMain(func() {
		quitAsked = true // as onBeforeQuit sets it
		closes = fw.H.ShouldClose()
		quitAsked = false
	})
	if !closes {
		t.Fatal("an undeletable dialog vetoed the app's quit")
	}
}
