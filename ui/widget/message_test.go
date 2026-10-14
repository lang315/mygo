package widget

import (
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

// Review Focus 3: Cancel and the close button both say no, once.
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
	type answer struct {
		s  string
		ok bool
	}
	got := runAsync(func() answer {
		s, ok := InputString(nil, Question, "Name?", "Your name", "")
		return answer{s, ok}
	})
	fw := waitFake(t, "Name?")
	typeText(fw, "Ann") // the entry has the focus
	key(fw, platform.KeyEnter)
	if a := wait(t, got); a.s != "Ann" || !a.ok {
		t.Fatalf("got %q %v, want Ann true", a.s, a.ok)
	}
}

func TestInputIntDefault(t *testing.T) {
	type answer struct {
		n  int
		ok bool
	}
	got := runAsync(func() answer {
		n, ok := InputInt(nil, Question, "Count?", "Threads", 1, 64, 1, 8)
		return answer{n, ok}
	})
	press(t, waitFake(t, "Count?"), "OK")
	if a := wait(t, got); a.n != 8 || !a.ok {
		t.Fatalf("got %d %v, want 8 true", a.n, a.ok)
	}
}

func TestInputSelectOKDisabledUntilAChoice(t *testing.T) {
	type answer struct {
		s  string
		ok bool
	}
	got := runAsync(func() answer {
		s, ok := InputSelect(nil, Question, "Pick?", "Proxy", Option{"a", "A"}, Option{"b", "B"})
		return answer{s, ok}
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
