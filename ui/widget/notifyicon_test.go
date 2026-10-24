package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
)

// newIcon makes a notify icon and returns it with its fake tray; the icon
// is disposed when the test ends.
func newIcon(t *testing.T) (*NotifyIcon, *fake.Tray) {
	t.Helper()
	before := len(fb.Trays())
	n, err := NewNotifyIcon()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(n.Dispose)
	trays := fb.Trays()
	if len(trays) != before+1 {
		t.Fatalf("%d trays after making one, had %d", len(trays), before)
	}
	return n, trays[len(trays)-1]
}

func TestNotifyIconReachesTheTray(t *testing.T) {
	n, tray := newIcon(t)
	png := []byte("\x89PNG icon")
	if err := n.SetIcon(png, true); err != nil {
		t.Fatal(err)
	}
	n.SetText("MyApp is running")
	if got, tmpl := tray.Image(); !slices.Equal(got, png) || !tmpl {
		t.Fatalf("tray image %q template %v", got, tmpl)
	}
	if tray.ToolTip() != "MyApp is running" {
		t.Fatalf("tooltip %q", tray.ToolTip())
	}
}

func TestNotifyIconContextMenuReachesTheTrayAndItsClicksArePosted(t *testing.T) {
	ran := make(chan bool, 1)
	n, tray := newIcon(t)
	n.SetContextMenu(MenuItem{Text: "Show", OnClick: func() { ran <- onEvents() }}, MenuItem{}, MenuItem{Text: "Quit"})
	m := tray.Menu()
	if m == nil || len(m.Items) != 3 || m.Items[0].Label != "Show" {
		t.Fatalf("tray menu %+v", m)
	}
	id := m.Items[0].ID
	mygo.RunOnMain(func() { fb.ClickMenuItem(id) })
	if !received(t, ran) {
		t.Fatal("the menu item's OnClick did not run on the event goroutine")
	}
	n.SetContextMenu()
	if tray.Menu() != nil && len(tray.Menu().Items) != 0 {
		t.Fatalf("an empty context menu left %+v", tray.Menu())
	}
}

func TestNotifyIconClicksArePosted(t *testing.T) {
	clicked, right := make(chan bool, 1), make(chan bool, 1)
	n, tray := newIcon(t)
	n.OnClick(func() { clicked <- onEvents() }).OnRightClick(func() { right <- onEvents() })
	mygo.RunOnMain(tray.Click)
	if !received(t, clicked) {
		t.Fatal("OnClick did not run on the event goroutine")
	}
	mygo.RunOnMain(tray.RightClick)
	if !received(t, right) {
		t.Fatal("OnRightClick did not run on the event goroutine")
	}
}

func TestNotifyIconVisibleHidesAndShowsTheTrayWithItsState(t *testing.T) {
	clicked := make(chan struct{}, 1)
	n, tray := newIcon(t)
	n.SetIcon([]byte("icon"), false)
	n.SetText("tip").OnClick(func() { clicked <- struct{}{} })
	n.SetContextMenu(MenuItem{Text: "Show"})
	if !n.Visible() {
		t.Fatal("a new icon is not visible")
	}
	if err := n.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if n.Visible() || slices.Contains(fb.Trays(), tray) {
		t.Fatalf("still shown: visible %v", n.Visible())
	}
	if err := n.SetVisible(true); err != nil {
		t.Fatal(err)
	}
	trays := fb.Trays()
	back := trays[len(trays)-1]
	if got, _ := back.Image(); string(got) != "icon" || back.ToolTip() != "tip" {
		t.Fatalf("the icon came back with image %q tooltip %q", got, back.ToolTip())
	}
	if findItem(back.Menu(), "Show") == nil {
		t.Fatalf("the icon came back without its context menu: %+v", back.Menu())
	}
	mygo.RunOnMain(back.Click)
	received(t, clicked)
}

// A click that reached the icon before it was hidden or disposed, and is
// handled after, runs nothing.
func TestNotifyIconIgnoresClicksOfATrayThatIsGone(t *testing.T) {
	ran := make(chan string, 4)
	n, tray := newIcon(t)
	n.OnClick(func() { ran <- "click" }).OnRightClick(func() { ran <- "right" })
	if err := n.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	mygo.RunOnMain(tray.Click) // the old tray, as a queued click delivers
	mygo.RunOnMain(tray.RightClick)
	n.Dispose()
	Update(func() {})
	if len(ran) != 0 {
		t.Fatalf("the handlers ran %d times for a tray that was hidden", len(ran))
	}
	n2, tray2 := newIcon(t)
	n2.OnClick(func() { ran <- "again" })
	n2.Dispose()
	mygo.RunOnMain(tray2.Click)
	Update(func() {})
	if len(ran) != 0 {
		t.Fatalf("a disposed icon's click ran its handler (%d)", len(ran))
	}
}

// An image the system cannot decode, set while the icon is hidden, is
// reported when the icon is shown.
func TestNotifyIconSetVisibleReturnsTheErrorOfAnImageItCannotShow(t *testing.T) {
	n, _ := newIcon(t)
	if err := n.SetVisible(false); err != nil {
		t.Fatal(err)
	}
	if err := n.SetIcon([]byte("corrupt"), false); err != nil {
		t.Fatalf("a hidden icon has no tray to refuse the image: %v", err)
	}
	if err := n.SetVisible(true); err == nil {
		t.Fatal("SetVisible(true) hid the error of the image")
	}
	if n.Visible() {
		t.Fatal("an icon that could not be shown says it is visible")
	}
	if err := n.SetIcon([]byte("icon"), false); err != nil {
		t.Fatal(err)
	}
	if err := n.SetVisible(true); err != nil || !n.Visible() {
		t.Fatalf("with a good image it shows: %v, visible %v", err, n.Visible())
	}
}

func TestNotifyIconDisposeRemovesTheTrayForGood(t *testing.T) {
	n, tray := newIcon(t)
	n.Dispose()
	if slices.Contains(fb.Trays(), tray) {
		t.Fatal("the tray is still there after Dispose")
	}
	live := len(fb.Trays())
	n.SetText("late").SetVisible(true) // does nothing, and does not panic
	if len(fb.Trays()) != live || n.Visible() {
		t.Fatal("a disposed icon came back")
	}
	n.Dispose() // twice is fine
}

func TestNotifyIconManyCallsFromManyGoroutinesAreRaceFree(t *testing.T) {
	n, _ := newIcon(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 50 {
			n.SetIcon([]byte("i"), false)
			n.SetText("tip").OnClick(func() {})
			n.SetVisible(i%2 == 0)
		}
	}()
	for range 50 {
		n.SetContextMenu(MenuItem{Text: "A"})
		n.OnRightClick(func() {})
	}
	<-done
}
