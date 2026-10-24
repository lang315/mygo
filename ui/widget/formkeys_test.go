package widget

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

func keysForm(content Control) *Form { return NewForm("Keys", 600, 400).SetContent(content) }

func TestFormShortcutRunsOnTheEventGoroutineWhereverTheFocusIs(t *testing.T) {
	ran := make(chan bool, 4)
	f := keysForm(VBox(4).Add(TextBox(), Button("Other"))).
		Shortcut(ui.Ctrl, ui.KeyS, func() { ran <- onEvents() })
	tt := TestForm(t, f)
	tt.Find(ByRole(ui.RoleTextField)).Focus().Key(ui.Ctrl, ui.KeyS)
	if !received(t, ran) {
		t.Fatal("the handler ran off the event goroutine")
	}
	tt.Find(ByLabel("Other")).Focus().Key(ui.Ctrl, ui.KeyS)
	received(t, ran)
}

func TestFormShortcutReplacesAndRemoves(t *testing.T) {
	var a, b atomic.Int32
	f := keysForm(Button("x")).Shortcut(0, ui.KeyF5, func() { a.Add(1) })
	f.Shortcut(0, ui.KeyF5, func() { b.Add(1) })
	tt := TestForm(t, f)
	tt.Find(ByLabel("x")).Key(0, ui.KeyF5)
	if a.Load() != 0 || b.Load() != 1 {
		t.Fatalf("first handler %d, second %d: the same key replaces", a.Load(), b.Load())
	}
	f.Shortcut(0, ui.KeyF5, nil)
	tt.Settle()
	tt.Find(ByLabel("x")).Key(0, ui.KeyF5)
	if b.Load() != 1 {
		t.Fatalf("a removed shortcut ran: %d", b.Load())
	}
}

func TestFormShortcutYieldsToAFocusedControlThatTakesTheKey(t *testing.T) {
	var space atomic.Int32
	clicks := 0
	f := keysForm(Button("B").OnClick(func() { clicks++ })).Shortcut(0, ui.KeySpace, func() { space.Add(1) })
	tt := TestForm(t, f)
	tt.Find(ByLabel("B")).Key(0, ui.KeySpace)
	if clicks != 1 || space.Load() != 0 {
		t.Fatalf("button clicks %d, shortcut %d: a focused button takes Space", clicks, space.Load())
	}
}

func TestAcceptButtonIsClickedByEnterFromATextBox(t *testing.T) {
	ok := 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	f := keysForm(VBox(4).Add(TextBox(), okBtn)).SetAcceptButton(okBtn)
	tt := TestForm(t, f)
	tt.Find(ByRole(ui.RoleTextField)).Type("x").Key(0, ui.KeyEnter)
	if ok != 1 {
		t.Fatalf("OK clicked %d times by Enter in a text box", ok)
	}
}

func TestAcceptButtonIsNotClickedTwiceWhenItHasTheFocus(t *testing.T) {
	ok := 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	f := keysForm(VBox(4).Add(TextBox(), okBtn)).SetAcceptButton(okBtn)
	tt := TestForm(t, f)
	tt.Find(ByLabel("OK")).Key(0, ui.KeyEnter)
	if ok != 1 {
		t.Fatalf("OK clicked %d times by Enter on itself", ok)
	}
}

func TestAcceptButtonLeavesEnterToAnotherFocusedButtonAndAMultilineTextBox(t *testing.T) {
	ok, other := 0, 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	f := keysForm(VBox(4).Add(MultilineTextBox(), Button("Other").OnClick(func() { other++ }), okBtn)).SetAcceptButton(okBtn)
	tt := TestForm(t, f)
	tt.Find(ByLabel("Other")).Key(0, ui.KeyEnter)
	if other != 1 || ok != 0 {
		t.Fatalf("Enter on Other: other %d, OK %d", other, ok)
	}
	tt.Find(ByRole(ui.RoleTextField)).Type("a").Key(0, ui.KeyEnter)
	if ok != 0 {
		t.Fatalf("Enter in a multiline text box clicked OK %d times", ok)
	}
}

func TestAcceptButtonDisabledOrHiddenOrClearedGetsNoEnter(t *testing.T) {
	ok := 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	f := keysForm(VBox(4).Add(TextBox(), okBtn)).SetAcceptButton(okBtn)
	tt := TestForm(t, f)
	field := tt.Find(ByRole(ui.RoleTextField)).Focus()
	okBtn.SetDisabled(true)
	tt.Settle()
	field.Key(0, ui.KeyEnter)
	okBtn.SetDisabled(false).SetVisible(false)
	tt.Settle()
	field.Key(0, ui.KeyEnter)
	okBtn.SetVisible(true)
	f.SetAcceptButton(nil)
	tt.Settle()
	field.Key(0, ui.KeyEnter)
	if ok != 0 {
		t.Fatalf("OK clicked %d times while disabled, hidden or cleared", ok)
	}
}

func TestCancelButtonWinsOverTheDialogsEscapeClose(t *testing.T) {
	asked := make(chan struct{}, 4)
	cancels := 0
	var d *Dialog
	cancel := Button("Cancel").OnClick(func() { cancels++; d.Close() })
	d = NewDialog(nil, "D", "", 300, 200, VBox(4).Add(TextBox(), cancel))
	d.OnClose(func() bool { asked <- struct{}{}; return true })
	d.SetCancelButton(cancel)
	dt := TestDialog(t, d)
	dt.Escape()
	flushEvents(t)
	select {
	case <-asked:
		t.Fatal("Escape went to the close request although a cancel button is set")
	default:
	}
	if cancels != 1 || !dt.Closed() {
		t.Fatalf("cancel clicked %d times, closed %v", cancels, dt.Closed())
	}
}

func TestCancelButtonThatDoesNothingKeepsTheDialogOpen(t *testing.T) {
	cancel := Button("Cancel")
	d := NewDialog(nil, "D", "", 300, 200, VBox(4).Add(TextBox(), cancel))
	d.SetCancelButton(cancel)
	dt := TestDialog(t, d)
	dt.Escape()
	flushEvents(t)
	if dt.Closed() {
		t.Fatal("Escape closed the dialog past a cancel button that does nothing")
	}
}

func TestDisabledCancelButtonFallsBackToEscapeClose(t *testing.T) {
	cancel := Button("Cancel")
	d := NewDialog(nil, "D", "", 300, 200, VBox(4).Add(TextBox(), cancel))
	d.SetCancelButton(cancel)
	cancel.SetDisabled(true)
	dt := TestDialog(t, d)
	dt.Escape()
	flushEvents(t)
	if !dt.Closed() {
		t.Fatal("a disabled cancel button must not block Escape-close")
	}
}

func TestEscapeClosesAnOpenComboBeforeTheCancelButton(t *testing.T) {
	cancels := 0
	cancel := Button("Cancel").OnClick(func() { cancels++ })
	combo := ComboBox(Option{"a", "Apple"}, Option{"b", "Banana"}).SetValue("a")
	f := keysForm(VBox(4).Add(combo, cancel)).SetCancelButton(cancel)
	tt := TestForm(t, f)
	tt.Find(ByLabel("Apple")).Click() // opens the popup
	if !tt.Has(ByLabel("Banana")) {
		t.Fatal("popup not open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Settle()
	if cancels != 0 || tt.Has(ByLabel("Banana")) {
		t.Fatalf("cancel clicked %d times; popup still open %v", cancels, tt.Has(ByLabel("Banana")))
	}
}

func TestAFormBehindAModalDialogGetsNoShortcuts(t *testing.T) {
	var n atomic.Int32
	parent := keysForm(Button("p")).Shortcut(0, ui.KeyF5, func() { n.Add(1) })
	dlg := NewDialog(parent, "M", "", 300, 200, Label("modal"))
	_ = dlg
	tt := TestForm(t, parent)
	tt.Find(ByLabel("p")).Key(0, ui.KeyF5)
	if n.Load() != 1 {
		t.Fatalf("shortcut ran %d times with no dialog open", n.Load())
	}
}

func TestTextBoxWithOnEnterRunsOnlyThatAndNotTheAcceptButton(t *testing.T) {
	ok, entered := 0, 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	f := keysForm(VBox(4).Add(TextBox().OnEnter(func() { entered++ }), okBtn)).SetAcceptButton(okBtn)
	tt := TestForm(t, f)
	tt.Find(ByRole(ui.RoleTextField)).Type("x").Key(0, ui.KeyEnter)
	if entered != 1 || ok != 0 {
		t.Fatalf("OnEnter ran %d times, OK clicked %d: OnEnter alone handles Enter", entered, ok)
	}
}

func TestEnterInAMultilineTextBoxIsNotAnAcceptClick(t *testing.T) {
	ok := 0
	okBtn := Button("OK").OnClick(func() { ok++ })
	box := MultilineTextBox()
	f := keysForm(VBox(4).Add(box, okBtn)).SetAcceptButton(okBtn)
	tt := TestForm(t, f)
	tt.Find(ByRole(ui.RoleTextField)).Type("a").Key(0, ui.KeyEnter).Type("b")
	if ok != 0 || box.Text() != "a\nb" {
		t.Fatalf("OK clicked %d times, text %q: Enter belongs to the multiline box", ok, box.Text())
	}
}

// A shortcut and a handler: the shortcut's handler is queued behind the one
// that was running, never run inside it.
func TestFormShortcutRunsAfterTheHandlerThatIsRunning(t *testing.T) {
	var order []string
	gate := make(chan struct{})
	f := keysForm(Button("x")).Shortcut(0, ui.KeyF5, func() { order = append(order, "shortcut") })
	tt := TestForm(t, f)
	tt.Find(ByLabel("x")).Focus()
	Post(func() { <-gate; order = append(order, "running") })
	tt.Tester.Key(0, ui.KeyF5)
	tt.Tester.Frame()
	close(gate)
	<-posted()
	if len(order) != 2 || order[0] != "running" || order[1] != "shortcut" {
		t.Fatalf("order %q", order)
	}
}

func TestFormShortcutAndButtonsFromWorkerGoroutinesAreRaceFree(t *testing.T) {
	var n atomic.Int32
	okBtn := Button("OK")
	f := keysForm(VBox(4).Add(TextBox(), okBtn))
	tt := TestForm(t, f)
	done := make(chan struct{})
	for g := 0; g < 4; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < 100; i++ {
				f.Shortcut(0, ui.KeyF5, func() { n.Add(1) })
				f.SetAcceptButton(okBtn)
				f.SetCancelButton(okBtn)
				f.Shortcut(0, ui.KeyF5, nil)
				f.SetAcceptButton(nil)
			}
		}()
	}
	for g := 0; g < 4; g++ {
		<-done
	}
	tt.Settle()
}

func TestDialogTesterEscapeClicksTheCancelButton(t *testing.T) {
	cancels := 0
	var d *Dialog
	cancel := Button("Cancel").OnClick(func() { cancels++ })
	d = NewDialog(nil, "D", "", 300, 200, VBox(4).Add(Button("x"), cancel))
	d.SetCancelButton(cancel)
	dt := TestDialog(t, d)
	dt.Find(ByLabel("x")).Focus()
	dt.Escape()
	flushEvents(t)
	if cancels != 1 || dt.Closed() {
		t.Fatalf("cancel clicked %d times, closed %v", cancels, dt.Closed())
	}
}
