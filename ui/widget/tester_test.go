package widget

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
)

// fakeTB records a Fatalf and stops the goroutine's test code with a panic,
// so a test can read the message a failing Find gives.
type fakeTB struct{ msg string }

type fatal struct{}

func (f *fakeTB) Helper()                   {}
func (f *fakeTB) Cleanup(func())            {}
func (f *fakeTB) Fatalf(s string, a ...any) { f.msg = fmt.Sprintf(s, a...); panic(fatal{}) }

// failure runs fn and returns the message of the Fatalf it ended with, ""
// when it ended without one.
func failure(fn func(tb *fakeTB)) (msg string) {
	tb := &fakeTB{}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(fatal); !ok {
				panic(r)
			}
			msg = tb.msg
		}
	}()
	fn(tb)
	return ""
}

func TestTesterFindsAButtonByLabelAndClicksIt(t *testing.T) {
	clicks := 0
	tt := NewTester(t, VBox(4).Add(Button("Cancel"), Button("OK").OnClick(func() { clicks++ })), 0, 0)
	tt.Find(ByLabel("OK")).Click()
	if clicks != 1 {
		t.Fatalf("OK clicked %d times", clicks)
	}
}

func TestTesterFindsByRoleAndNth(t *testing.T) {
	var got []string
	tt := NewTester(t, VBox(4).Add(
		Button("A").OnClick(func() { got = append(got, "A") }),
		Button("B").OnClick(func() { got = append(got, "B") }),
		Label("text"),
	), 0, 0)
	if n := len(tt.FindAll(ByRole(ui.RoleButton))); n != 2 {
		t.Fatalf("%d buttons", n)
	}
	tt.Find(ByRole(ui.RoleButton).Nth(1)).Click()
	tt.Find(ByRole(ui.RoleButton).Label("A")).Click()
	if strings.Join(got, "") != "BA" {
		t.Fatalf("clicked %q", got)
	}
	if !tt.Has(ByLabel("text")) || tt.Has(ByLabel("nope")) {
		t.Fatal("Has")
	}
}

func TestTesterLabelsMatchExactlyAndCaseSensitively(t *testing.T) {
	tt := NewTester(t, VBox(4).Add(Button("OK now"), Button("ok")), 0, 0)
	if tt.Has(ByLabel("OK")) {
		t.Fatal("ByLabel(OK) matched OK now or ok")
	}
}

func TestTesterFindFailsWithWhatItFound(t *testing.T) {
	tt := NewTester(t, VBox(4).Add(Button("Ok"), Button("Cancel")), 0, 0)
	msg := failure(func(tb *fakeTB) {
		tt.tb = tb
		tt.Find(ByRole(ui.RoleButton).Label("OK"))
	})
	for _, want := range []string{`no button labeled "OK"`, `button "Ok"`, `button "Cancel"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q lacks %q", msg, want)
		}
	}
	msg = failure(func(tb *fakeTB) { tt.tb = tb; tt.Find(ByRole(ui.RoleButton)) })
	if !strings.Contains(msg, "2 elements match") || !strings.Contains(msg, `button "Ok"`) {
		t.Errorf("ambiguous message %q", msg)
	}
	msg = failure(func(tb *fakeTB) { tt.tb = tb; tt.Find(ByRole(ui.RoleButton).Nth(5)) })
	if !strings.Contains(msg, "Nth(5)") {
		t.Errorf("out of range message %q", msg)
	}
}

func TestTesterTypesAndReadsValuesAndStates(t *testing.T) {
	name, agree := TextBox(), CheckBox("I agree")
	tt := NewTester(t, VBox(4).Add(name, agree), 0, 0)
	field := tt.Find(ByRole(ui.RoleTextField))
	field.Type("Ada")
	if field.Value() != "Ada" || name.Text() != "Ada" {
		t.Fatalf("value %q, text %q", field.Value(), name.Text())
	}
	box := tt.Find(ByLabel("I agree"))
	if box.Checked() || !box.Click().Checked() || !agree.Checked() {
		t.Fatal("the check box did not check")
	}
	if tt.Find(ByLabel("I agree")).Disabled() {
		t.Fatal("disabled")
	}
}

func TestTesterActionFailuresSayWhy(t *testing.T) {
	ok := Button("OK")
	gone := Button("Gone")
	tt := NewTester(t, VBox(4).Add(ok, gone), 0, 0)
	btn := tt.Find(ByLabel("OK"))
	ok.SetDisabled(true)
	tt.Settle()
	msg := failure(func(tb *fakeTB) { tt.tb = tb; btn.Click() })
	if !strings.Contains(msg, `"OK"`) || !strings.Contains(msg, "disabled") {
		t.Errorf("disabled: %q", msg)
	}
	g := tt.Find(ByLabel("Gone"))
	gone.SetVisible(false)
	tt.Settle()
	msg = failure(func(tb *fakeTB) { tt.tb = tb; g.Click() })
	if !strings.Contains(msg, `"Gone"`) || !strings.Contains(msg, "no longer") {
		t.Errorf("gone: %q", msg)
	}
	tall := make([]Control, 30)
	for i := range tall {
		tall[i] = Button(fmt.Sprint("row ", i))
	}
	small := NewTester(t, VBox(4).Add(tall...), 200, 100)
	msg = failure(func(tb *fakeTB) { small.tb = tb; small.Find(ByLabel("row 29")).Click() })
	if !strings.Contains(msg, `"row 29"`) || !strings.Contains(msg, "outside the window") {
		t.Errorf("off window: %q", msg)
	}
}

func TestTesterReadsProgress(t *testing.T) {
	p := ProgressBar().SetFraction(0.25)
	tt := NewTester(t, p, 0, 0)
	n := tt.Find(ByRole(ui.RoleProgress))
	if n.Min() != 0 || n.Max() != 1 && n.Max() != 100 || n.Now() == 0 {
		t.Fatalf("progress %v..%v at %v", n.Min(), n.Max(), n.Now())
	}
}

func TestTesterSettleRunsWhatWasPosted(t *testing.T) {
	ran := false
	tt := NewTester(t, Label("x"), 0, 0)
	Post(func() { ran = true })
	tt.Settle()
	if !ran {
		t.Fatal("Settle did not wait for the event goroutine")
	}
}

func TestTesterFormDrawsTheStatusStrip(t *testing.T) {
	f := NewForm("F", 300, 200).SetContent(Label("body")).StatusStrip(StatusStrip(Label("ready")))
	tt := TestForm(t, f)
	if !tt.Has(ByLabel("ready")) || !tt.Has(ByLabel("body")) {
		t.Fatalf("texts %q", tt.Texts())
	}
}

func TestTesterRightClickAndDoubleClick(t *testing.T) {
	var doubles, rights int
	c := Custom(func(c *ui.Context) ui.Element {
		e := ui.Text(c, "target")
		if e.DoubleClicked() {
			doubles++
		}
		if e.RightClicked() {
			rights++
		}
		return e
	})
	tt := NewTester(t, c, 0, 0)
	tt.Find(ByLabel("target")).DoubleClick()
	tt.Find(ByLabel("target")).RightClick()
	if doubles != 1 || rights != 1 {
		t.Fatalf("double clicks %d, right clicks %d", doubles, rights)
	}
}

func TestTesterClicksANodeScrolledOutOfViewAfterScrollingToIt(t *testing.T) {
	var clicked []int
	rows := make([]Control, 20)
	for i := range rows {
		rows[i] = Button(fmt.Sprint("row ", i)).OnClick(func() { clicked = append(clicked, i) })
	}
	tt := NewTester(t, ScrollPanel(VBox(0).Add(rows...)), 300, 100)
	tt.Find(ByLabel("row 15")).Click()
	if len(clicked) != 1 || clicked[0] != 15 {
		t.Fatalf("clicked %v, want only row 15", clicked)
	}
}

func TestTesterNodeFollowsItsElementNotItsPlace(t *testing.T) {
	show := []string{"A", "B", "C"}
	got := ""
	c := Custom(func(c *ui.Context) ui.Element {
		return ui.Column(c).Children(func() {
			for _, s := range show {
				if ui.Button(c, s).Clicked() {
					got += s
				}
			}
		})
	})
	tt := NewTester(t, c, 0, 0)
	b := tt.Find(ByLabel("B"))
	Change(func() { show = []string{"B", "C"} })
	tt.Settle()
	tt.Find(ByLabel("C")) // still there
	msg := failure(func(tb *fakeTB) { tt.tb = tb; b.Click() })
	if got != "" || msg == "" {
		t.Fatalf("clicked %q, message %q: the node of B must not become C's", got, msg)
	}
}

func TestTesterNodesOfEqualLabelsKeepTheirOwnIdentity(t *testing.T) {
	var a, b int
	tt := NewTester(t, VBox(4).Add(Button("X").OnClick(func() { a++ }), Button("X").OnClick(func() { b++ })), 0, 0)
	second := tt.Find(ByLabel("X").Nth(1))
	second.Click()
	second.Click()
	if a != 0 || b != 2 {
		t.Fatalf("first clicked %d, second %d times", a, b)
	}
}

func TestTesterKeysDoNotPressAFocusedButtonTwice(t *testing.T) {
	clicks := 0
	tt := NewTester(t, Button("OK").OnClick(func() { clicks++ }), 0, 0)
	tt.Find(ByLabel("OK")).Key(0, ui.KeyEnter)
	if clicks != 1 {
		t.Fatalf("Enter on OK ran OnClick %d times", clicks)
	}
}

func TestTesterFocusDoesNotActivateAndFocusedTellsIt(t *testing.T) {
	box, other := CheckBox("Agree"), TextBox()
	tt := NewTester(t, VBox(4).Add(box, other), 0, 0)
	n := tt.Find(ByLabel("Agree"))
	if n.Focused() {
		t.Fatal("focused before Focus")
	}
	n.Focus()
	if box.Checked() || !n.Focused() {
		t.Fatalf("checked %v, focused %v after Focus", box.Checked(), n.Focused())
	}
	tt.Find(ByRole(ui.RoleTextField)).Focus()
	if n.Focused() {
		t.Fatal("the check box kept the focus")
	}
}

func TestTesterTypeAppendsToTheEndOfExistingText(t *testing.T) {
	box := TextBox().SetText("0123456789012345678901234567890123456789")
	tt := NewTester(t, box, 0, 0)
	tt.Find(ByRole(ui.RoleTextField)).Type("X")
	if got := box.Text(); got != "0123456789012345678901234567890123456789X" {
		t.Fatalf("text %q", got)
	}
}

func TestTesterSettleRunsWhatHandlersPostInTurn(t *testing.T) {
	done := false
	tt := NewTester(t, Label("x"), 0, 0)
	Post(func() { Post(func() { Post(func() { done = true }) }) })
	tt.Settle()
	if !done {
		t.Fatal("Settle returned before the work a handler posted ran")
	}
}

func TestTesterSettleGivesUpOnWorkThatNeverEnds(t *testing.T) {
	var stop atomic.Bool
	var again func()
	again = func() {
		if !stop.Load() {
			Post(again)
		}
	}
	tt := NewTester(t, Label("x"), 0, 0)
	Post(again)
	msg := failure(func(tb *fakeTB) { tt.tb = tb; tt.Settle() })
	stop.Store(true)
	<-posted()
	if !strings.Contains(msg, "still posting") {
		t.Fatalf("message %q", msg)
	}
}

func TestTesterFollowsTheWindowBeingResized(t *testing.T) {
	rows := make([]Control, 30)
	for i := range rows {
		rows[i] = Button(fmt.Sprint("row ", i))
	}
	tt := NewTester(t, VBox(0).Add(rows...), 200, 100)
	tt.SetSize(200, 3000)
	tt.Settle()
	tt.Find(ByLabel("row 29")).Click() // outside the first size, inside this one
}

func TestTesterHasHonoursNthAndEmptyBounds(t *testing.T) {
	tt := NewTester(t, Button("only"), 0, 0)
	if tt.Has(ByRole(ui.RoleButton).Nth(5)) {
		t.Fatal("Has(Nth(5)) is true with one button")
	}
	if !tt.Has(ByRole(ui.RoleButton).Nth(0)) {
		t.Fatal("Has(Nth(0)) is false")
	}
}

func TestTesterMessageForAGoneNodeSaysWhatIsThereNow(t *testing.T) {
	b := Button("Start")
	tt := NewTester(t, b, 0, 0)
	n := tt.Find(ByLabel("Start"))
	b.SetText("Stop")
	tt.Settle()
	msg := failure(func(tb *fakeTB) { tt.tb = tb; n.Value() })
	if !strings.Contains(msg, "no longer shown") || !strings.Contains(msg, `"Stop"`) {
		t.Fatalf("message %q", msg)
	}
}

// A node is bound to the element it was found as. A button that renames
// itself is found again; Label reads what it shows now.
func TestTesterNodeAfterItsOwnRelabelingNeedsFindingAgain(t *testing.T) {
	var b *ButtonWidget
	b = Button("Start").OnClick(func() { b.SetText("Stop") })
	tt := NewTester(t, b, 0, 0)
	n := tt.Find(ByLabel("Start"))
	if got := n.Click().Label(); got != "Stop" {
		t.Fatalf("Label reads %q after the click, want what the button shows now", got)
	}
	msg := failure(func(tb *fakeTB) { tt.tb = tb; n.Click() })
	if !strings.Contains(msg, "no longer shown") || !strings.Contains(msg, `"Stop"`) {
		t.Fatalf("a second click on the node found as Start: %q", msg)
	}
	tt.Find(ByLabel("Stop")) // found again, it is a node of the new label
}

// An element that removes itself on a click can leave its identity to
// another: the node must not follow it.
func TestTesterNodeDoesNotMoveToAnotherElementAfterItsOwnAction(t *testing.T) {
	show := []string{"A", "B"}
	var clicked []string
	c := Custom(func(c *ui.Context) ui.Element {
		return ui.Column(c).Children(func() {
			for _, s := range show {
				if ui.Button(c, s).Clicked() {
					clicked = append(clicked, s)
					if s == "A" {
						show = []string{"B"}
					}
				}
			}
		})
	})
	tt := NewTester(t, c, 0, 0)
	a := tt.Find(ByLabel("A"))
	a.Click()
	msg := failure(func(tb *fakeTB) { tt.tb = tb; a.Click() })
	if strings.Join(clicked, "") != "A" || msg == "" {
		t.Fatalf("clicked %q, message %q: the node of A must not become B's", clicked, msg)
	}
}

func TestTesterFocusFailsOnAnElementThatCannotTakeIt(t *testing.T) {
	name := TextBox()
	tt := NewTester(t, VBox(4).Add(Label("Name"), name), 0, 0)
	tt.Find(ByRole(ui.RoleTextField)).Focus()
	msg := failure(func(tb *fakeTB) { tt.tb = tb; tt.Find(ByLabel("Name")).Type("Ada") })
	if !strings.Contains(msg, "cannot take the keyboard focus") || name.Text() != "" {
		t.Fatalf("message %q, field %q: typing into a label went to the field that had the focus", msg, name.Text())
	}
}
