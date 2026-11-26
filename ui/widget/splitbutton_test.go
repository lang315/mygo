package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSplitButtonMainClick(t *testing.T) {
	clicked := make(chan bool, 4)
	b := SplitButton("Save", MenuItem{Text: "Save As…"}).OnClick(func() { clicked <- onEvents() })
	tt := NewTester(t, b, 400, 300)
	tt.Find(ByLabel("Save")).Click()
	if !received(t, clicked) {
		t.Fatal("OnClick did not run on the event goroutine")
	}
	select {
	case <-clicked:
		t.Fatal("OnClick ran twice for one click")
	default:
	}
	if menu := tt.Menu(); len(menu) != 0 {
		t.Fatalf("a click on the main part opened the menu %q", menu)
	}
}

func TestSplitButtonArrowOpensItems(t *testing.T) {
	clicked := make(chan struct{}, 4)
	chose := make(chan bool, 4)
	b := SplitButton("Save",
		MenuItem{Text: "Save As…", OnClick: func() { chose <- onEvents() }},
		MenuItem{Text: "Save All"},
	).OnClick(func() { clicked <- struct{}{} })
	tt := NewTester(t, b, 400, 300)
	tt.Find(ByLabel("More options")).Click()
	if menu := tt.Menu(); len(menu) != 2 || menu[0] != "Save As…" || menu[1] != "Save All" {
		t.Fatalf("the arrow opened %q", menu)
	}
	if err := tt.ChooseMenuItem("Save As…"); err != nil {
		t.Fatal(err)
	}
	tt.Settle()
	if !received(t, chose) {
		t.Fatal("the item's OnClick did not run on the event goroutine")
	}
	select {
	case <-clicked:
		t.Fatal("the arrow ran the button's OnClick")
	default:
	}
}

func TestSplitButtonDisabled(t *testing.T) {
	clicked := make(chan struct{}, 4)
	b := SplitButton("Save", MenuItem{Text: "Save As…"}).OnClick(func() { clicked <- struct{}{} })
	tt := NewTester(t, b.SetDisabled(true), 400, 300)
	tt.ClickAt(middle(tt.Find(ByLabel("Save")).Bounds()))
	tt.ClickAt(middle(tt.Find(ByLabel("More options")).Bounds()))
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if menu := tt.Menu(); len(menu) != 0 {
		t.Fatalf("a disabled SplitButton opened %q", menu)
	}
	select {
	case <-clicked:
		t.Fatal("a disabled SplitButton ran OnClick")
	default:
	}
}

func TestSplitButtonSuggestedAndIcon(t *testing.T) {
	clicked := make(chan struct{}, 1)
	b := SplitButton("Save").Suggested().SetIcon(chevronDown).OnClick(func() { clicked <- struct{}{} })
	tt := NewTester(t, b, 400, 300)
	tt.Find(ByLabel("Save")).Click()
	received(t, clicked)
}
