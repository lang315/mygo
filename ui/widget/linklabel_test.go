package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func linkTester(t *testing.T, l *LinkLabelWidget) *ui.Tester {
	t.Helper()
	return tester(t, VBox(4).Add(l))
}

func TestLinkLabelOpensItsAddressWithoutOnClick(t *testing.T) {
	tt := linkTester(t, LinkLabel("Help", "https://example.invalid/help"))
	click(t, tt, "Help")
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://example.invalid/help"}) {
		t.Fatalf("opened %q", got)
	}
}

func TestLinkLabelOnClickReplacesTheSystemOpen(t *testing.T) {
	ran := make(chan struct{}, 4)
	l := LinkLabel("Help", "https://example.invalid/help").OnClick(func() { ran <- struct{}{} })
	tt := linkTester(t, l)
	click(t, tt, "Help")
	received(t, ran)
	if got := tt.OpenedURLs(); len(got) != 0 {
		t.Fatalf("opened %q though OnClick is set", got)
	}
	// Enter and Space with the focus run it too.
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		tt.Key(0, key)
		tt.Frame()
		received(t, ran)
	}
	if got := tt.OpenedURLs(); len(got) != 0 {
		t.Fatalf("opened %q by a key", got)
	}
	// A nil OnClick gives the open back.
	l.OnClick(nil)
	tt.Frame()
	click(t, tt, "Help")
	if got := tt.OpenedURLs(); len(got) != 1 {
		t.Fatalf("opened %q after OnClick(nil)", got)
	}
}

func TestLinkLabelOnClickRunsOnTheEventGoroutineAndMayChangeWidgets(t *testing.T) {
	done := make(chan struct{}, 1)
	status := Label("idle")
	l := LinkLabel("Help", "").OnClick(func() { status.SetText("Opened"); done <- struct{}{} })
	tt := tester(t, VBox(4).Add(l, status))
	click(t, tt, "Help")
	received(t, done)
	tt.Frame()
	if !tt.HasText("Opened") {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestLinkLabelDisabledDoesNothing(t *testing.T) {
	ran := make(chan struct{}, 4)
	l := LinkLabel("Help", "https://example.invalid/help").OnClick(func() { ran <- struct{}{} }).SetDisabled(true)
	tt := linkTester(t, l)
	click(t, tt, "Help")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	flushEvents(t)
	if len(ran) != 0 || len(tt.OpenedURLs()) != 0 {
		t.Fatalf("a disabled link ran OnClick %d times, opened %q", len(ran), tt.OpenedURLs())
	}
	l.SetDisabled(false)
	tt.Frame()
	click(t, tt, "Help")
	received(t, ran)
}

func TestLinkLabelPressedByAScreenReaderRunsOnClick(t *testing.T) {
	ran := make(chan struct{}, 4)
	f := NewForm("LinkPress", 300, 200)
	f.SetContent(VBox(4).Add(LinkLabel("Help", "https://example.invalid/help").OnClick(func() { ran <- struct{}{} })))
	f.Show()
	t.Cleanup(f.Close)
	fw := waitFake(t, "LinkPress")
	press(t, fw, "Help")
	received(t, ran)
}
