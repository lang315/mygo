package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

func TestConfirmOK(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Delete?", "It goes for good.") })
	press(t, waitFake(t, "Delete?"), "OK")
	if !wait(t, got) {
		t.Fatal("OK returned false")
	}
}

// Cancel and the close button both say no, once.
func TestConfirmCancelAndCloseSayNo(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Warning, "Cancel?", "") })
	press(t, waitFake(t, "Cancel?"), "Cancel")
	if wait(t, got) {
		t.Fatal("Cancel returned true")
	}
	got = runAsync(func() bool { return Confirm(nil, Warning, "Closed?", "") })
	fw := waitFake(t, "Closed?")
	mygo.RunOnMain(func() { fw.UserClose() }) // the close button
	mygo.RunOnMain(func() { fw.UserClose() })
	if wait(t, got) {
		t.Fatal("closing returned true")
	}
}

func TestAlertReturnsOnOK(t *testing.T) {
	done := runAsync(func() bool { Alert(nil, Error, "Failed", "Disk full"); return true })
	press(t, waitFake(t, "Failed"), "OK")
	wait(t, done)
}

func TestInputStringTypedThenEnter(t *testing.T) {
	got := runAsync(func() answer[string] {
		s, ok := InputString(nil, Question, "Name?", "Your name", "")
		return answer[string]{s, ok}
	})
	fw := waitFake(t, "Name?")
	typeText(fw, "Ann") // the entry has the focus
	key(fw, platform.KeyEnter)
	if a := wait(t, got); a.v != "Ann" || !a.ok {
		t.Fatalf("got %q %v, want Ann true", a.v, a.ok)
	}
}

func TestInputIntDefault(t *testing.T) {
	got := runAsync(func() answer[int] {
		n, ok := InputInt(nil, Question, "Count?", "Threads", 1, 64, 1, 8)
		return answer[int]{n, ok}
	})
	press(t, waitFake(t, "Count?"), "OK")
	if a := wait(t, got); a.v != 8 || !a.ok {
		t.Fatalf("got %d %v, want 8 true", a.v, a.ok)
	}
}

func TestInputSelectOKDisabledUntilAChoice(t *testing.T) {
	got := runAsync(func() answer[string] {
		s, ok := InputSelect(nil, Question, "Pick?", "Proxy", Option{"a", "A"}, Option{"b", "B"})
		return answer[string]{s, ok}
	})
	fw := waitFake(t, "Pick?")
	press(t, fw, "OK") // disabled: nothing happens
	select {
	case a := <-got:
		t.Fatalf("OK while disabled returned %v", a)
	default:
	}
	press(t, fw, "Cancel")
	if a := wait(t, got); a.ok {
		t.Fatalf("Cancel returned %v", a)
	}
}

func TestIconTakesItsSize(t *testing.T) {
	svg := ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/></svg>`))
	tt := tester(t, HBox(0).Add(Icon(svg, 40), Label("after")))
	if a := find(t, tt, "after"); a.X < 39 {
		t.Fatalf("after at %v: the icon is not 40 wide", a)
	}
}

func TestInputIntEnter(t *testing.T) {
	got := runAsync(func() answer[int] {
		n, ok := InputInt(nil, Question, "Enter?", "Threads", 1, 64, 1, 8)
		return answer[int]{n, ok}
	})
	fw := waitFake(t, "Enter?")
	frame(fw) // focuses the number
	key(fw, platform.KeyEnter)
	if a := wait(t, got); a.v != 8 || !a.ok {
		t.Fatalf("got %d %v, want 8 true", a.v, a.ok)
	}
}

// Two answers queued while the event goroutine is busy: the first wins.
func TestConfirmFirstAnswerWins(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Twice?", "") })
	fw := waitFake(t, "Twice?")
	busy, release := make(chan struct{}), make(chan struct{})
	Post(func() { close(busy); <-release })
	<-busy
	press(t, fw, "Cancel")
	press(t, fw, "OK")
	close(release)
	if wait(t, got) {
		t.Fatal("OK after Cancel won")
	}
}

// Escape closes a dialog as its close button does: a question says no.
func TestEscapeClosesADialog(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Escape?", "") })
	fw := waitFake(t, "Escape?")
	frame(fw)
	key(fw, platform.KeyEscape)
	if wait(t, got) {
		t.Fatal("Escape returned true")
	}
}

// Escape closes an open drop-down of a dialog first, then the dialog.
func TestEscapeClosesADialogsDropDownFirst(t *testing.T) {
	got := runAsync(func() answer[string] {
		s, ok := InputSelect(nil, Question, "Drop?", "Proxy", Option{"a", "Alpha"}, Option{"b", "Beta"})
		return answer[string]{s, ok}
	})
	fw := waitFake(t, "Drop?")
	frame(fw)
	press(t, fw, "") // the combo, nothing chosen
	if !slices.Contains(labels(fw), "Beta") {
		t.Fatalf("pressing the combo did not open it: %q", labels(fw))
	}
	key(fw, platform.KeyEscape)
	Update(func() {})
	if fw.IsClosed() || slices.Contains(labels(fw), "Beta") {
		t.Fatalf("the first Escape did not close only the drop-down: closed %v, %q", fw.IsClosed(), labels(fw))
	}
	key(fw, platform.KeyEscape)
	if a := wait(t, got); a.ok {
		t.Fatalf("Escape returned %v", a)
	}
}

// An undeletable dialog stays open on Escape.
func TestEscapeKeepsAnUndeletableDialog(t *testing.T) {
	var d *Dialog
	d = NewDialog(nil, "Kept", "", 300, 200, Button("Done").OnClick(func() { d.Close() }))
	d.SetDeletable(false)
	ran := runAsync(func() bool { d.Run(); return true })
	fw := waitFake(t, "Kept")
	frame(fw)
	key(fw, platform.KeyEscape)
	Update(func() {})
	if fw.IsClosed() {
		t.Fatal("Escape closed an undeletable dialog")
	}
	press(t, fw, "Done")
	wait(t, ran)
}
