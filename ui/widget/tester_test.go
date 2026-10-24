package widget

import (
	"fmt"
	"strings"
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
