package widget

import "testing"

func checks(rs ...*RadioButtonWidget) []bool {
	out := make([]bool, len(rs))
	for i, r := range rs {
		out[i] = r.Checked()
	}
	return out
}

func wantChecks(t *testing.T, got []bool, want ...bool) {
	t.Helper()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("checked %v, want %v", got, want)
		}
	}
}

func TestRadioButtonsOfOneBoxAreOneGroup(t *testing.T) {
	a, b, c := RadioButton("A"), RadioButton("B"), RadioButton("C")
	box := VBox(4).Add(a, b, c)
	tt := tester(t, box)
	a.SetChecked(true)
	tt.Frame()
	wantChecks(t, checks(a, b, c), true, false, false)
	click(t, tt, "B")
	wantChecks(t, checks(a, b, c), false, true, false)
	b.SetChecked(false) // none checked is allowed
	c.SetChecked(true)  // and SetChecked unchecks the others at once, before a frame
	wantChecks(t, checks(a, b, c), false, false, true)
	a.SetChecked(true)
	wantChecks(t, checks(a, b, c), true, false, false)
	tt.Frame()
	click(t, tt, "C")
	wantChecks(t, checks(a, b, c), false, false, true)
}

func TestRadioButtonsInSeparateGroupBoxesAreSeparateGroups(t *testing.T) {
	a, b := RadioButton("A"), RadioButton("B")
	c, d := RadioButton("C"), RadioButton("D")
	left, right := GroupBox("Left"), GroupBox("Right")
	left.Add(VBox(2).Add(a, b))
	right.Add(VBox(2).Add(c, d))
	tt := tester(t, HBox(8).Add(left, right))
	click(t, tt, "A")
	click(t, tt, "C")
	wantChecks(t, checks(a, b, c, d), true, false, true, false)
	click(t, tt, "D")
	wantChecks(t, checks(a, b, c, d), true, false, false, true)
	b.SetChecked(true)
	wantChecks(t, checks(a, b, c, d), false, true, false, true)
}

func TestRadioButtonsDirectlyInAGroupBoxOrFieldsGroup(t *testing.T) {
	a, b := RadioButton("A"), RadioButton("B")
	f := Fields().Row("a", a).Row("b", b)
	tt := tester(t, f)
	a.SetChecked(true)
	b.SetChecked(true)
	tt.Frame()
	wantChecks(t, checks(a, b), false, true)
}

func TestRadioButtonOnChangeRunsForTheUserNotForSetChecked(t *testing.T) {
	a, b := RadioButton("A"), RadioButton("B")
	ran := make(chan string, 8)
	a.OnChange(func() { ran <- "a" })
	b.OnChange(func() { ran <- "b" })
	tt := tester(t, VBox(2).Add(a, b))
	a.SetChecked(true)
	tt.Frame()
	click(t, tt, "B")
	got := map[string]bool{soon(t, ran): true, soon(t, ran): true}
	if !got["a"] || !got["b"] {
		t.Fatalf("OnChange ran for %v, want both: the one checked and the one unchecked by the user", got)
	}
	select {
	case s := <-ran:
		t.Fatalf("an extra OnChange: %q (SetChecked must not run it)", s)
	default:
	}
}

func TestRadioButtonOutsideAContainerStandsAlone(t *testing.T) {
	a := RadioButton("Solo")
	tt := tester(t, a)
	click(t, tt, "Solo")
	if !a.Checked() {
		t.Fatal("not checked")
	}
}

func TestRadioButtonRemovedFromItsBoxLeavesTheGroup(t *testing.T) {
	a, b := RadioButton("A"), RadioButton("B")
	box := VBox(2).Add(a, b)
	box.Remove(b)
	a.SetChecked(true)
	b.SetChecked(true)
	wantChecks(t, checks(a, b), true, true)
}

func TestRadioButtonsAddedCheckedLeaveOneChecked(t *testing.T) {
	a, b := RadioButton("A").SetChecked(true), RadioButton("B").SetChecked(true)
	VBox(2).Add(a, b)
	wantChecks(t, checks(a, b), false, true) // the last added wins
	c := RadioButton("C").SetChecked(true)
	box := VBox(2).Add(a, b)
	box.Add(c)
	if n := 0; true {
		for _, on := range checks(a, b, c) {
			if on {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("checked %v, want one", checks(a, b, c))
		}
	}
}

// soon returns what the event goroutine has sent to ch by now, failing at
// once when it sent nothing: posted work is drained first, so a missing
// OnChange is a failure, not a wait for the one-minute guard.
func soon[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	<-posted()
	select {
	case v := <-ch:
		return v
	default:
		t.Fatal("the handler did not run")
		panic("unreachable")
	}
}
