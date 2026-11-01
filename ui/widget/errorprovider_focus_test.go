package widget

import (
	"strings"
	"sync"
	"testing"
)

// focusForm is a form of text boxes a, b, c in tab order, each with the error
// of ep when its name is in bad; typed returns the name of the box that takes
// the next typed character.
func focusForm(t *testing.T, content func(boxes ...*TextBoxWidget) Control) (f *Form, tt *Tester, boxes []*TextBoxWidget, ep *ErrorProviderWidget) {
	t.Helper()
	ep = ErrorProvider()
	boxes = []*TextBoxWidget{TextBox(), TextBox(), TextBox()}
	f = NewForm("focus", 400, 300)
	f.SetContent(content(boxes...))
	tt = TestForm(t, f)
	tt.Settle()
	return
}

// typed types a character and returns the index of the box that got it, -1
// for none; it clears the box again.
func typed(tt *Tester, boxes []*TextBoxWidget) int {
	tt.Tester.Type("#")
	tt.Settle()
	at := -1
	for i, b := range boxes {
		if strings.Contains(b.Text(), "#") {
			at = i
			b.SetText("")
		}
	}
	return at
}

func vbox(boxes ...*TextBoxWidget) Control {
	return VBox(4).Add(boxes[0], boxes[1], boxes[2])
}

func TestFocusFirstGoesToTheFirstErrorInTabOrder(t *testing.T) {
	f, tt, boxes, ep := focusForm(t, vbox)
	ep.SetError(boxes[2], "c")
	ep.SetError(boxes[1], "b") // set after c: tab order, not the order of SetError
	if !ep.FocusFirst(f) {
		t.Fatal("FocusFirst said no control was marked")
	}
	tt.Settle()
	if got := typed(tt, boxes); got != 1 {
		t.Fatalf("the focus went to box %d, want 1", got)
	}
}

func TestFocusFirstDoesNothingWithoutErrors(t *testing.T) {
	f, tt, boxes, ep := focusForm(t, vbox)
	boxes[0].Focus()
	tt.Settle()
	if ep.FocusFirst(f) {
		t.Fatal("FocusFirst said true with no errors")
	}
	tt.Settle()
	if got := typed(tt, boxes); got != 0 {
		t.Fatalf("the focus moved to box %d with no errors", got)
	}
}

func TestFocusFirstSkipsHiddenAndDisabledControls(t *testing.T) {
	f, tt, boxes, ep := focusForm(t, vbox)
	for _, b := range boxes {
		ep.SetError(b, "bad")
	}
	boxes[0].SetVisible(false)
	boxes[1].SetDisabled(true)
	tt.Settle()
	if !ep.FocusFirst(f) {
		t.Fatal("FocusFirst said no control could take the focus")
	}
	tt.Settle()
	if got := typed(tt, boxes); got != 2 {
		t.Fatalf("the focus went to box %d, want 2", got)
	}
	boxes[2].SetDisabled(true)
	if ep.FocusFirst(f) {
		t.Fatal("FocusFirst said true when every marked control is hidden or disabled")
	}
}

func TestFocusFirstSkipsAControlInADisabledContainer(t *testing.T) {
	f, tt, boxes, ep := focusForm(t, func(b ...*TextBoxWidget) Control {
		return VBox(4).Add(GroupBox("g").Add(b[0]).SetDisabled(true), b[1], b[2])
	})
	ep.SetError(boxes[0], "a")
	ep.SetError(boxes[2], "c")
	tt.Settle()
	if !ep.FocusFirst(f) {
		t.Fatal("FocusFirst said no")
	}
	tt.Settle()
	if got := typed(tt, boxes); got != 2 {
		t.Fatalf("the focus went to box %d, want 2", got)
	}
}

// A control on a page that does not show: the page shows, then the control has
// the focus (ValidateChildren, which walks every page, does not switch pages).
func TestFocusFirstShowsThePageOfTheControl(t *testing.T) {
	var tabs *TabControlWidget
	var side *SidebarWidget
	f, tt, boxes, ep := focusForm(t, func(b ...*TextBoxWidget) Control {
		side = Sidebar().Add("one", "One", b[0]).Add("two", "Two", b[1])
		tabs = TabControl().AddPage("A", Label("a")).AddPage("B", VBox(4).Add(side)).Expand()
		return VBox(4).Add(tabs, b[2])
	})
	ep.SetError(boxes[1], "second page of the sidebar, second tab")
	tt.Settle()
	if tabs.Page() != 0 {
		t.Fatalf("page %d to start", tabs.Page())
	}
	if !ep.FocusFirst(f) {
		t.Fatal("FocusFirst said no")
	}
	tt.Settle()
	tt.Settle()
	if tabs.Page() != 1 || side.Selected() != "two" {
		t.Fatalf("page %d, sidebar %q: the pages of the control did not show", tabs.Page(), side.Selected())
	}
	if got := typed(tt, boxes); got != 1 {
		t.Fatalf("the focus went to box %d, want 1", got)
	}
}

// After a failed ValidateChildren, the first invalid control takes the focus.
func TestFocusFirstAfterValidateChildren(t *testing.T) {
	f, tt, boxes, ep := focusForm(t, vbox)
	for i, b := range boxes {
		b.OnValidating(func() bool {
			if b.Text() == "" {
				ep.SetError(b, "required")
				return false
			}
			ep.SetError(b, "")
			return true
		})
		_ = i
	}
	boxes[0].SetText("x")
	if f.ValidateChildren() {
		t.Fatal("validated")
	}
	ep.FocusFirst(f)
	tt.Settle()
	if got := typed(tt, boxes); got != 1 {
		t.Fatalf("the focus went to box %d, want the first invalid one, 1", got)
	}
}

// From a worker goroutine while frames run: the request is made under the
// lock and applied at the next build (red only under -race).
func TestFocusFirstFromAWorkerDoesNotRace(t *testing.T) {
	f, tt, boxes, ep := focusForm(t, vbox)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 200 {
			ep.SetError(boxes[i%3], "bad")
			ep.FocusFirst(f)
		}
	}()
	for range 100 {
		tt.Frame()
	}
	wg.Wait()
}
