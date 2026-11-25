package widget

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// tagsCase is the setup of the cases: a TagSelect of a, b and c shown as
// A, B and C, ordered or not, with the number of OnChange runs, and the
// toggle of an option text.
type tagsCase struct {
	g  *TagSelectWidget
	tt *Tester
	n  *int
}

func newTagsCase(t *testing.T, ordered bool) *tagsCase {
	t.Helper()
	fn, n := counted()
	g := TagSelect(Option{"a", "A"}, Option{"b", "B"}, Option{"c", "C"}).OnChange(fn)
	if ordered {
		g.SetOrdered(true)
	}
	return &tagsCase{g, NewTester(t, g, 400, 300), n}
}

// toggle finds the toggle of option text s, not its chip.
func (tc *tagsCase) toggle(s string) *Node {
	return tc.tt.Find(ByLabel(s).Role(ui.RoleToggleButton))
}

func (tc *tagsCase) wantValues(t *testing.T, want ...string) {
	t.Helper()
	if got := tc.g.Values(); !slices.Equal(got, want) || len(got) != len(want) {
		t.Fatalf("Values() = %q, want %q", got, want)
	}
}

func (tc *tagsCase) wantChanges(t *testing.T, want int) {
	t.Helper()
	if *tc.n != want {
		t.Fatalf("OnChange ran %d times, want %d", *tc.n, want)
	}
}

// removeLabels returns the labels of the chips' remove buttons, in tree order.
func (tc *tagsCase) removeLabels() []string {
	var ls []string
	for _, n := range tc.tt.Tester.Nodes() {
		if strings.HasPrefix(n.Label, "Remove") {
			ls = append(ls, n.Label)
		}
	}
	return ls
}

// firstText returns where s first shows among the texts of the form, or -1.
func (tc *tagsCase) firstText(s string) int { return slices.Index(tc.tt.Texts(), s) }

// Case 1: nothing chosen shows no chip, no remove button and no empty row:
// the toggles stand where those of a TagSelect without SetOrdered do.
func TestTagSelectOrderedEmpty(t *testing.T) {
	tc := newTagsCase(t, true)
	plain := newTagsCase(t, false)
	tc.wantValues(t)
	if ls := tc.removeLabels(); len(ls) != 0 {
		t.Fatalf("remove buttons %q with nothing chosen", ls)
	}
	if got, want := tc.tt.Texts(), plain.tt.Texts(); !slices.Equal(got, want) {
		t.Fatalf("texts %q, want those of a plain TagSelect %q", got, want)
	}
	if got, want := tc.toggle("A").Bounds(), plain.toggle("A").Bounds(); got != want {
		t.Fatalf("toggle A at %v, want %v as without SetOrdered", got, want)
	}
}

// Case 2: C then A gives [c a], two OnChange, the chips C before A.
func TestTagSelectOrderedPickOrder(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.toggle("C").Click()
	tc.toggle("A").Click()
	tc.wantValues(t, "c", "a")
	tc.wantChanges(t, 2)
	if ls := tc.removeLabels(); !slices.Equal(ls, []string{"Remove C", "Remove A"}) {
		t.Fatalf("remove buttons %q, want Remove C, Remove A", ls)
	}
	if c, a := tc.firstText("C"), tc.firstText("A"); c < 0 || a < 0 || c > a {
		t.Fatalf("texts %q: want the chip C before A", tc.tt.Texts())
	}
}

// Case 3: Remove C takes c out, turns its toggle off, and runs OnChange.
func TestTagSelectOrderedRemoveButton(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.toggle("C").Click()
	tc.toggle("A").Click()
	tc.tt.Find(ByLabel("Remove C")).Click()
	tc.wantValues(t, "a")
	if tc.toggle("C").Checked() {
		t.Fatal("C's toggle is still on")
	}
	tc.wantChanges(t, 3)
	if ls := tc.removeLabels(); !slices.Equal(ls, []string{"Remove A"}) {
		t.Fatalf("remove buttons %q, want Remove A", ls)
	}
}

// Case 4: turning the last toggle off leaves no chip.
func TestTagSelectOrderedToggleOff(t *testing.T) {
	tc := newTagsCase(t, true)
	empty := tc.tt.Texts()
	tc.toggle("C").Click()
	tc.toggle("A").Click()
	tc.tt.Find(ByLabel("Remove C")).Click()
	tc.toggle("A").Click()
	tc.wantValues(t)
	tc.wantChanges(t, 4)
	if ls := tc.removeLabels(); len(ls) != 0 {
		t.Fatalf("remove buttons %q with nothing chosen", ls)
	}
	if got := tc.tt.Texts(); !slices.Equal(got, empty) {
		t.Fatalf("texts %q, want %q as with nothing chosen", got, empty)
	}
}

// Case 5: SetValues keeps its order, drops non-options and repeats, and
// runs no OnChange.
func TestTagSelectOrderedSetValues(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.g.SetValues("b", "zzz", "a", "b")
	tc.tt.Settle()
	tc.wantValues(t, "b", "a")
	tc.wantChanges(t, 0)
	if !tc.toggle("B").Checked() || !tc.toggle("A").Checked() || tc.toggle("C").Checked() {
		t.Fatal("want the toggles of b and a on, c off")
	}
	if ls := tc.removeLabels(); !slices.Equal(ls, []string{"Remove B", "Remove A"}) {
		t.Fatalf("remove buttons %q, want Remove B, Remove A", ls)
	}
}

// Case 6: without SetOrdered, Values keeps option order and there is no
// remove button.
func TestTagSelectUnorderedKeepsOptionOrder(t *testing.T) {
	tc := newTagsCase(t, false)
	tc.toggle("C").Click()
	tc.toggle("A").Click()
	tc.wantValues(t, "a", "c")
	tc.wantChanges(t, 2)
	if ls := tc.removeLabels(); len(ls) != 0 {
		t.Fatalf("remove buttons %q without SetOrdered", ls)
	}
}

// Case 7: SetOrdered(false) after picks goes back to option order and the
// look of a plain TagSelect.
func TestTagSelectSetOrderedFalse(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.toggle("C").Click()
	tc.toggle("A").Click()
	tc.g.SetOrdered(false)
	tc.tt.Settle()
	tc.wantValues(t, "a", "c")
	if ls := tc.removeLabels(); len(ls) != 0 {
		t.Fatalf("remove buttons %q after SetOrdered(false)", ls)
	}
	// And back: the picks still on keep their order.
	tc.g.SetOrdered(true)
	tc.tt.Settle()
	tc.wantValues(t, "c", "a")
}

// Turning on SetOrdered after picks made without it orders them as the
// options, then appends the next picks.
func TestTagSelectSetOrderedAfterPlainPicks(t *testing.T) {
	tc := newTagsCase(t, false)
	tc.toggle("C").Click()
	tc.toggle("A").Click()
	tc.g.SetOrdered(true)
	tc.tt.Settle()
	tc.wantValues(t, "a", "c")
	tc.toggle("B").Click()
	tc.wantValues(t, "a", "c", "b")
}

// Case 8, the first order: a SetValues after the user's edit, before the
// next frame, applies on that frame.
func TestTagSelectOrderedSetterAfterUserEdit(t *testing.T) {
	tc := newTagsCase(t, true)
	b := tc.toggle("B").Bounds()
	tc.tt.Tester.ClickAt(b.X+b.W/2, b.Y+b.H/2) // no settle: OnChange is posted, not run
	tc.tt.Frame()
	tc.g.SetValues("c")
	tc.tt.Settle()
	tc.wantValues(t, "c")
	if tc.toggle("B").Checked() || !tc.toggle("C").Checked() {
		t.Fatal("want B's toggle off and C's on")
	}
	tc.wantChanges(t, 1)
}

// Case 8, the second order: a SetValues after the frame was built and
// before ui commits the user's edit loses to the edit for the toggle the
// user changed, as mirror.commit does; the setter's other toggles hold, and
// the user's pick comes after them. The harness cannot interleave a setter
// there, so the test does what ui does: it changes the shown value and
// calls the toggle's commit.
func TestTagSelectOrderedUserEditBeatsSetterBefore(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.g.SetValues("c")     // after the build that showed nothing chosen
	tc.g.on[1].shown = true // the user turns B on, as ui writes it
	tc.g.picked(1)()
	tc.tt.Settle()
	tc.wantValues(t, "c", "b")
	if !tc.toggle("B").Checked() || !tc.toggle("C").Checked() {
		t.Fatal("want the toggles of B and C on")
	}
	if ls := tc.removeLabels(); !slices.Equal(ls, []string{"Remove C", "Remove B"}) {
		t.Fatalf("remove buttons %q, want Remove C, Remove B", ls)
	}
	tc.wantChanges(t, 1)
}

// A commit with nothing changed moves no pick and runs no OnChange.
func TestTagSelectOrderedCommitWithoutChange(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.g.SetValues("a", "c")
	tc.tt.Settle()
	tc.g.picked(0)()
	tc.tt.Settle()
	tc.wantValues(t, "a", "c")
	tc.wantChanges(t, 0)
}

// Case 9: the remove buttons are buttons that Tab passes by.
func TestTagSelectOrderedRemoveButtonsUnfocusable(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.g.SetValues("c", "a")
	tc.tt.Settle()
	for _, l := range []string{"Remove C", "Remove A"} {
		if n := tc.tt.Find(ByLabel(l)); n.Role() != ui.RoleButton {
			t.Fatalf("%s has role %v, want button", l, n.Role())
		}
	}
	focused := 0
	for range 8 {
		tc.tt.Tester.Key(0, ui.KeyTab)
		tc.tt.Frame()
		for _, n := range tc.tt.Tester.Nodes() {
			if !n.Focused {
				continue
			}
			focused++
			if strings.HasPrefix(n.Label, "Remove") {
				t.Fatalf("Tab focused %q", n.Label)
			}
		}
	}
	if focused == 0 {
		t.Fatal("Tab focused nothing: the test sees no focus")
	}
}

// Case 10: setters and getters on other goroutines while the user clicks;
// run under -race.
func TestTagSelectOrderedRace(t *testing.T) {
	tc := newTagsCase(t, true)
	tc.g.SetValues("c", "a")
	tc.tt.Settle()
	var wg sync.WaitGroup
	run := func(stop chan struct{}, fn func(i int)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				fn(i)
			}
		}()
	}
	stop := make(chan struct{})
	run(stop, func(i int) {
		_ = tc.g.Values()
		if i%50 == 0 {
			tc.g.SetOrdered(true)
		}
	})
	tc.toggle("B").Click()
	tc.tt.Find(ByLabel("Remove C")).Click()
	tc.toggle("C").Click()
	close(stop)
	wg.Wait()

	// SetOrdered(false) rebuilds the toggles under another parent: the user
	// clicks where they were, whatever stands there now.
	at := map[string]ui.Rect{}
	for _, s := range []string{"A", "B", "C"} {
		at[s] = tc.toggle(s).Bounds()
	}
	stop = make(chan struct{})
	run(stop, func(i int) {
		_ = tc.g.Values()
		switch i % 60 {
		case 0:
			tc.g.SetValues("b", "a")
		case 20:
			tc.g.SetOrdered(false)
		case 40:
			tc.g.SetOrdered(true)
		}
	})
	for _, s := range []string{"A", "C", "B", "A"} {
		r := at[s]
		tc.tt.Tester.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		tc.tt.Settle()
	}
	close(stop)
	wg.Wait()
	tc.tt.Settle()
	// Whatever won, the order holds the toggles that are on.
	got := tc.g.Values()
	for i, s := range []string{"A", "B", "C"} {
		v := tc.g.opts[i].Value
		if tc.toggle(s).Checked() != slices.Contains(got, v) {
			t.Fatalf("Values %q, toggle %s on: %v", got, s, tc.toggle(s).Checked())
		}
	}
}

// Case 11: an idle frame of a TagSelect without SetOrdered allocates no
// more than one built as before SetOrdered existed.
func TestTagSelectIdleAllocations(t *testing.T) {
	opts := []Option{{"a", "A"}, {"b", "B"}, {"c", "C"}}
	frame := func(tt *Tester) float64 {
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(10, tt.Frame)
	}
	on := make([]mirror[bool], len(opts))
	on[1].set(true)
	h := func() {}
	before := Custom(func(c *ui.Context) ui.Element {
		return ui.Row(c).Wrap().Gap(c.Theme().Space(1.5)).Children(func() {
			for i, o := range opts {
				p, changed := on[i].bind()
				skinChecked(c, ui.Toggle(c.Key(i), p, o.Text).OnChange(onChange(&on[i], h)), *p)
				if changed {
					Post(h)
				}
			}
		})
	})
	bare := frame(NewTester(t, before, 400, 300))
	mine := frame(NewTester(t, TagSelect(opts...).SetValues("b").OnChange(h), 400, 300))
	t.Logf("before %.1f allocations a frame, TagSelect %.1f", bare, mine)
	if mine > bare {
		t.Errorf("TagSelect allocates %.1f a frame, as before %.1f", mine, bare)
	}
}
