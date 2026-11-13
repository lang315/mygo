package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

func TestConfirmOK(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Delete?", "It goes for good.") })
	press(t, waitFake(t, "Delete?"), "Yes")
	if !received(t, got) {
		t.Fatal("Yes returned false")
	}
}

// Cancel and the close button both say no, once.
func TestConfirmCancelAndCloseSayNo(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Warning, "Cancel?", "") })
	press(t, waitFake(t, "Cancel?"), "No")
	if received(t, got) {
		t.Fatal("No returned true")
	}
	got = runAsync(func() bool { return Confirm(nil, Warning, "Closed?", "") })
	fw := waitFake(t, "Closed?")
	mygo.RunOnMain(func() { fw.UserClose() }) // the close button
	mygo.RunOnMain(func() { fw.UserClose() })
	if received(t, got) {
		t.Fatal("closing returned true")
	}
}

// A dialog whose window closes before Show watches it still returns:
// here it closes as it is made, on the main thread, as NewWindow fires
// OnWindowCreated.
func TestConfirmClosedAsItOpens(t *testing.T) {
	off := mygo.App.OnWindowCreated(func(w *mygo.Window) {
		if w.Title() == "At once?" {
			w.Close()
		}
	})
	defer off()
	got := runAsync(func() bool { return Confirm(nil, Warning, "At once?", "") })
	if received(t, got) {
		t.Fatal("closing returned true")
	}
}

func TestAlertReturnsOnOK(t *testing.T) {
	done := runAsync(func() bool { Alert(nil, Error, "Failed", "Disk full"); return true })
	press(t, waitFake(t, "Failed"), "OK")
	received(t, done)
}

func TestInputStringTypedThenEnter(t *testing.T) {
	got := runAsync(func() answer[string] {
		s, ok := InputString(nil, Question, "Name?", "Your name", "")
		return answer[string]{s, ok}
	})
	fw := waitFake(t, "Name?")
	typeText(fw, "Ann") // the entry has the focus
	key(fw, platform.KeyEnter)
	if a := received(t, got); a.v != "Ann" || !a.ok {
		t.Fatalf("got %q %v, want Ann true", a.v, a.ok)
	}
}

func TestInputIntDefault(t *testing.T) {
	got := runAsync(func() answer[int] {
		n, ok := InputInt(nil, Question, "Count?", "Threads", 1, 64, 1, 8)
		return answer[int]{n, ok}
	})
	press(t, waitFake(t, "Count?"), "OK")
	if a := received(t, got); a.v != 8 || !a.ok {
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
	if a := received(t, got); a.ok {
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
	if a := received(t, got); a.v != 8 || !a.ok {
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
	press(t, fw, "No")
	press(t, fw, "Yes")
	close(release)
	if received(t, got) {
		t.Fatal("Yes after No won")
	}
}

// Escape closes a dialog as its close button does: a question says no.
func TestEscapeClosesADialog(t *testing.T) {
	got := runAsync(func() bool { return Confirm(nil, Question, "Escape?", "") })
	fw := waitFake(t, "Escape?")
	frame(fw)
	key(fw, platform.KeyEscape)
	if received(t, got) {
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
	if a := received(t, got); a.ok {
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
	received(t, ran)
}

// With a skin, a message icon is the family of its kind: its base and the
// glyph on it, Question the accent and the text on it; without one, today's
// icons. The same kind and colors are parsed once.
func TestMessageIconsUseSkinColors(t *testing.T) {
	light, dark := useTestSkin(t)
	colors := func(tk *skinhook.Tokens, k Kind) (base, glyph ui.Color) {
		switch k {
		case Info:
			return tk.Info.Base, tk.Info.On
		case Warning:
			return tk.Warning.Base, tk.Warning.On
		case Error:
			return tk.Danger.Base, tk.Danger.On
		}
		return tk.Accent.Base, tk.OnAccent
	}
	for _, k := range []Kind{Info, Warning, Question, Error} {
		for _, d := range []bool{false, true} {
			tk := light
			if d {
				tk = dark
			}
			base, glyph := colors(tk, k)
			tt := NewTester(t, kindIcon(k), 120, 80)
			tt.SetDark(d)
			tt.Settle()
			img := tt.Image()
			if n := countNear(img, ui.Rect{}, true, base, 4); n < 200 {
				t.Errorf("kind %d, dark %v: %d pixels of the base %v, want 200 or more", k, d, n, base)
			}
			if n := countNear(img, ui.Rect{}, true, glyph, 4); n < 15 {
				t.Errorf("kind %d, dark %v: %d pixels of the glyph %v, want 15 or more", k, d, n, glyph)
			}
			if k == Warning {
				// The dot under the bar is the glyph's color too: the bottom
				// quarter of the triangle holds the dot alone.
				b := iconBounds(img)
				q := ui.Rect{X: float32(b.Min.X), Y: float32(b.Max.Y - b.Dy()/4), W: float32(b.Dx()), H: float32(b.Dy() / 4)}
				if n := countNear(img, q, false, glyph, 4); n < 3 {
					t.Errorf("dark %v: %d pixels of the glyph %v in the dot", d, n, glyph)
				}
			}
		}
	}

	// Switching the tokens while the icon shows: its next frame takes them.
	tt := NewTester(t, kindIcon(Error), 120, 80)
	tt.Settle()
	other := *light
	other.Danger.Base, other.Danger.On = ui.Hex("#0d47a1"), ui.Hex("#ffd600")
	skinhook.Set(&other, dark)
	ui.SetDefaultThemes(nil, nil)
	tt.Settle()
	img := tt.Image()
	if n := countNear(img, ui.Rect{}, true, other.Danger.Base, 4); n < 200 {
		t.Errorf("after switching: %d pixels of the new base, want 200 or more", n)
	}
	if n := countNear(img, ui.Rect{}, true, light.Danger.Base, 4); n > 0 {
		t.Errorf("after switching: %d pixels of the old base", n)
	}

	// Parsed once a kind and colors; another color or kind is another icon.
	a := kindSVG(Info, light.Info.Base, light.Info.On)
	if b := kindSVG(Info, light.Info.Base, light.Info.On); b != a {
		t.Error("the same kind and colors gave a second icon")
	}
	if b := kindSVG(Info, light.Info.Base, dark.Info.On); b == a {
		t.Error("another glyph color gave the same icon")
	}
	if b := kindSVG(Question, light.Info.Base, light.Info.On); b == a {
		t.Error("another kind gave the same icon")
	}

	// Without a skin, today's icons.
	skinhook.Set(nil, nil)
	for _, k := range []Kind{Info, Warning, Question, Error} {
		if got := kindIconSVG(k, false); got != plainKindIcons[k] {
			t.Errorf("kind %d without a skin is not its plain icon", k)
		}
	}
}
