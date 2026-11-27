package widget

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// spanTester shows s, 300 wide, then a button to move the focus to, and
// returns the duration's text field.
func spanTester(t *testing.T, s *TimeSpanEditWidget) (*Tester, *Node) {
	t.Helper()
	calmOff()
	tt := NewTester(t, VBox(0).Add(s.SetSize(300, 0), Button("Next")), 400, 160)
	tt.Settle()
	return tt, tt.Find(ByRole(ui.RoleTextField))
}

// spanCaret focuses the field and puts its caret i runes into its text.
func spanCaret(in *Node, i int) {
	in.Focus()
	in.Key(0, ui.KeyHome)
	for range i {
		in.Key(0, ui.KeyRight)
	}
}

// spanRetype replaces the field's text with s, typed, keeping the focus in it.
func spanRetype(in *Node, s string) {
	in.Focus()
	in.Key(0, ui.KeyEnd)
	in.Key(ui.Shift, ui.KeyHome)
	in.Type(s)
}

// spanLeave moves the focus out of the field, to Next.
func spanLeave(tt *Tester) {
	tt.Find(ByRole(ui.RoleButton).Label("Next")).Focus()
	tt.Settle()
}

const spanMax = 9999*time.Hour + 59*time.Minute + 59*time.Second

func TestTimeSpanFormat(t *testing.T) {
	for _, c := range []struct {
		name string
		make func() *TimeSpanEditWidget
		want string
	}{
		{"zero", func() *TimeSpanEditWidget { return TimeSpanEdit() }, "00:00:00"},
		{"90m", func() *TimeSpanEditWidget { return TimeSpanEdit().SetValue(90 * time.Minute) }, "01:30:00"},
		{"26h", func() *TimeSpanEditWidget { return TimeSpanEdit().SetValue(26 * time.Hour) }, "26:00:00"},
		{"26h days", func() *TimeSpanEditWidget { return TimeSpanEdit().SetShowDays(true).SetValue(26 * time.Hour) }, "1.02:00:00"},
		{"-5s", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetRange(-time.Hour, time.Hour).SetValue(-5 * time.Second)
		}, "-00:00:05"},
		{"-1s", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetRange(-time.Hour, time.Hour).SetValue(-time.Second)
		}, "-00:00:01"},
		{"9h9m9s", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetValue(9*time.Hour + 9*time.Minute + 9*time.Second)
		}, "09:09:09"},
		{"10h10m10s", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetValue(10*time.Hour + 10*time.Minute + 10*time.Second)
		}, "10:10:10"},
		{"90m no seconds", func() *TimeSpanEditWidget { return TimeSpanEdit().SetShowSeconds(false).SetValue(90 * time.Minute) }, "01:30"},
		{"max", func() *TimeSpanEditWidget { return TimeSpanEdit().SetValue(spanMax) }, "9999:59:59"},
		{"days no seconds", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetShowDays(true).SetShowSeconds(false).SetValue(26*time.Hour + 30*time.Minute)
		}, "1.02:30"},
		{"seconds hidden", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetShowSeconds(false).SetValue(90*time.Minute + 45*time.Second)
		}, "01:30"},
		{"sub-second", func() *TimeSpanEditWidget { return TimeSpanEdit().SetValue(1500 * time.Millisecond) }, "00:00:01"},
		{"minus sub-second", func() *TimeSpanEditWidget {
			return TimeSpanEdit().SetRange(-time.Hour, 0).SetValue(-500 * time.Millisecond)
		}, "00:00:00"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, in := spanTester(t, c.make())
			if got := in.Value(); got != c.want {
				t.Fatalf("the field shows %q, want %q", got, c.want)
			}
		})
	}
	// Hidden seconds stay in the value.
	s := TimeSpanEdit().SetShowSeconds(false).SetValue(90*time.Minute + 45*time.Second)
	if v := s.Value(); v != 90*time.Minute+45*time.Second {
		t.Fatalf("with seconds hidden the value is %v, want 1h30m45s", v)
	}
	// Showing them again shows them.
	_, in := spanTester(t, s)
	s.SetShowSeconds(true)
	spanSettle(t, in)
	if got := in.Value(); got != "01:30:45" {
		t.Fatalf("seconds shown again: %q, want 01:30:45", got)
	}
}

// spanSettle settles the tester of in.
func spanSettle(t *testing.T, in *Node) { t.Helper(); in.t.Settle() }

func TestTimeSpanStepUnderCaret(t *testing.T) {
	var log editLog
	s := TimeSpanEdit().SetValue(59 * time.Minute).OnChange(log.add("change"))
	tt, in := spanTester(t, s)
	// "00:5|9:00": the caret in the minutes.
	spanCaret(in, 4)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != time.Hour {
		t.Fatalf("Up on minutes 59 made %v, want 1h: the minutes carry into the hour", v)
	}
	if got := in.Value(); got != "01:00:00" {
		t.Fatalf("after Up the field shows %q, want 01:00:00", got)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("Up ran %q, want one change", got)
	}
	// The caret stays in the minutes.
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != time.Hour+time.Minute {
		t.Fatalf("a second Up made %v, want 1h1m: the caret left the minutes", v)
	}
	in.Key(0, ui.KeyDown)
	in.Key(0, ui.KeyDown)
	if v := s.Value(); v != 59*time.Minute {
		t.Fatalf("Down twice made %v, want 59m: the hour borrows back", v)
	}
	// Up with a modifier is the input's, not a step.
	in.Key(ui.Shift, ui.KeyUp)
	if v := s.Value(); v != 59*time.Minute {
		t.Fatalf("Shift+Up made %v, want 59m: no step", v)
	}

	// Hours, seconds, and the caret at the end of the text: seconds.
	for _, c := range []struct {
		caret int
		from  time.Duration
		want  time.Duration
	}{
		{0, 0, time.Hour},
		{1, 0, time.Hour},
		{2, 0, time.Hour}, // before the colon: still the hours
		{3, 0, time.Minute},
		{6, 0, time.Second},
		{8, 0, time.Second},                             // at the end
		{8, 59 * time.Second, time.Minute},              // seconds carry into the minute
		{8, 59*time.Minute + 59*time.Second, time.Hour}, // and on into the hour
	} {
		s.SetValue(c.from)
		tt.Settle()
		spanCaret(in, c.caret)
		in.Key(0, ui.KeyUp)
		if v := s.Value(); v != c.want {
			t.Fatalf("Up with the caret at %d from %v made %v, want %v", c.caret, c.from, v, c.want)
		}
	}

	// Down below the range's min stays at the min, and runs nothing.
	s.SetValue(0)
	tt.Settle()
	log.reset()
	spanCaret(in, 4)
	in.Key(0, ui.KeyDown)
	if v := s.Value(); v != 0 {
		t.Fatalf("Down below the min made %v, want 0", v)
	}
	if got := in.Value(); got != "00:00:00" {
		t.Fatalf("Down below the min shows %q", got)
	}
	// Up at the max stays at the max.
	s.SetValue(spanMax)
	tt.Settle()
	spanCaret(in, 9)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != spanMax {
		t.Fatalf("Up at the max made %v, want %v", v, spanMax)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("steps at the bounds ran %q", got)
	}
	// A step from the max's part below it clamps there.
	s.SetValue(9999 * time.Hour)
	tt.Settle()
	spanCaret(in, 1)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != spanMax {
		t.Fatalf("Up an hour from 9999h made %v, want the max %v", v, spanMax)
	}
}

func TestTimeSpanStepDays(t *testing.T) {
	s := TimeSpanEdit().SetShowDays(true).SetValue(26 * time.Hour)
	_, in := spanTester(t, s)
	spanCaret(in, 0) // "|1.02:00:00": the days
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 50*time.Hour {
		t.Fatalf("Up on the days made %v, want 50h", v)
	}
	if got := in.Value(); got != "2.02:00:00" {
		t.Fatalf("after Up on the days the field shows %q", got)
	}
	in.Key(0, ui.KeyUp) // the caret stayed in the days
	if v := s.Value(); v != 74*time.Hour {
		t.Fatalf("a second Up on the days made %v, want 74h", v)
	}
	in.Key(0, ui.KeyDown)
	spanCaret(in, 3) // "2.0|2:00:00": the hours
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 51*time.Hour {
		t.Fatalf("Up on the hours made %v, want 51h", v)
	}
	// Hours carry into the days.
	s.SetValue(23 * time.Hour)
	in.t.Settle()
	spanCaret(in, 3) // "0.2|3:00:00"
	in.Key(0, ui.KeyUp)
	if got := in.Value(); got != "1.00:00:00" {
		t.Fatalf("Up on hours 23 shows %q, want 1.00:00:00", got)
	}
}

// Keys faster than the frames (KeysAtOnce: no frame between them) each
// step: the second sees the text the first left.
func TestTimeSpanStepKeysAtOnce(t *testing.T) {
	s := TimeSpanEdit().SetValue(58 * time.Minute)
	tt, in := spanTester(t, s)
	spanCaret(in, 4)
	tt.KeysAtOnce(ui.KeyUp, ui.KeyUp, ui.KeyUp)
	if v := s.Value(); v != time.Hour+time.Minute {
		t.Fatalf("three Ups at once on 58m made %v, want 1h1m", v)
	}
}

// A step that narrows the hours keeps the caret in its part: the offset
// it had would be in the next part.
func TestTimeSpanStepKeepsThePartAsTheTextNarrows(t *testing.T) {
	s := TimeSpanEdit().SetValue(1000 * time.Hour)
	_, in := spanTester(t, s)
	spanCaret(in, 7) // "1000:00|:00": the end of the minutes
	in.Key(0, ui.KeyDown)
	in.Key(0, ui.KeyDown) // "999:59:00", then the minutes again
	if v := s.Value(); v != 999*time.Hour+58*time.Minute {
		t.Fatalf("Down twice from 1000h on the minutes made %v, want 999h58m", v)
	}
	// The hours widen: "99:00:00" to "100:00:00", the caret after them.
	s.SetValue(99 * time.Hour)
	in.t.Settle()
	spanCaret(in, 2) // "99|:00:00"
	in.Key(0, ui.KeyUp)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 101*time.Hour {
		t.Fatalf("Up twice from 99h on the hours made %v, want 101h", v)
	}
}

// The caret on the sign steps the hours.
func TestTimeSpanStepNegative(t *testing.T) {
	s := TimeSpanEdit().SetRange(-10*time.Hour, 10*time.Hour).SetValue(-5 * time.Second)
	_, in := spanTester(t, s)
	spanCaret(in, 0)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != time.Hour-5*time.Second {
		t.Fatalf("Up with the caret on the sign made %v, want 59m55s", v)
	}
}

// The side buttons step the part under the caret, and leave the focus in
// the text.
func TestTimeSpanSideButtons(t *testing.T) {
	var log editLog
	s := TimeSpanEdit().SetValue(time.Hour).OnChange(log.add("change"))
	tt, in := spanTester(t, s)
	spanCaret(in, 4)
	tt.Find(ByRole(ui.RoleButton).Label("Increase")).Click()
	if v := s.Value(); v != time.Hour+time.Minute {
		t.Fatalf("Increase with the caret in the minutes made %v, want 1h1m", v)
	}
	if !in.Focused() {
		t.Fatal("a click on Increase moved the focus out of the text")
	}
	tt.Find(ByRole(ui.RoleButton).Label("Decrease")).Click()
	tt.Find(ByRole(ui.RoleButton).Label("Decrease")).Click()
	if v := s.Value(); v != time.Hour-time.Minute {
		t.Fatalf("Decrease twice made %v, want 59m", v)
	}
	if got := in.Value(); got != "00:59:00" {
		t.Fatalf("after the buttons the field shows %q", got)
	}
	if got := log.all(t); len(got) != 3 {
		t.Fatalf("three clicks ran %q, want three changes", got)
	}
	// The buttons did not count as leaving the field: typed text stays
	// typed until the focus leaves.
	if !in.Focused() {
		t.Fatal("the focus left the text")
	}
}

// Never focused, the field steps its first part: the hours.
func TestTimeSpanSideButtonsUnfocused(t *testing.T) {
	s := TimeSpanEdit()
	tt, _ := spanTester(t, s)
	tt.Find(ByRole(ui.RoleButton).Label("Increase")).Click()
	if v := s.Value(); v != time.Hour {
		t.Fatalf("Increase on a field never focused made %v, want 1h", v)
	}
}

// A step while typed text is in the field steps from that text when it
// parses, from the last value when it does not.
func TestTimeSpanStepTypedText(t *testing.T) {
	s := TimeSpanEdit().SetValue(time.Hour)
	_, in := spanTester(t, s)
	spanRetype(in, "2:00:00")
	in.Key(0, ui.KeyHome)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 3*time.Hour {
		t.Fatalf("Up on typed 2:00:00 made %v, want 3h", v)
	}
	spanRetype(in, "zz")
	in.Key(0, ui.KeyHome)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 4*time.Hour {
		t.Fatalf("Up on typed zz made %v, want 4h: the last value, stepped", v)
	}
	if got := in.Value(); got != "04:00:00" {
		t.Fatalf("Up on typed zz shows %q", got)
	}
	// Days typed while they are hidden: a day steps, and the caret, with
	// no days to stay in, goes to the end, the seconds.
	spanRetype(in, "1.02:00:00")
	in.Key(0, ui.KeyHome)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 50*time.Hour {
		t.Fatalf("Up on typed days 1.02:00:00 made %v, want 50h", v)
	}
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 50*time.Hour+time.Second {
		t.Fatalf("the next Up made %v, want 50h0m1s: the caret at the end", v)
	}
}

func TestTimeSpanParse(t *testing.T) {
	var log editLog
	s := TimeSpanEdit().SetValue(time.Minute).OnChange(log.add("change"))
	tt, in := spanTester(t, s)
	spanRetype(in, "1:2:3")
	if v := s.Value(); v != time.Minute {
		t.Fatalf("typing changed the value to %v before leaving", v)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("typing ran %q before leaving", got)
	}
	spanLeave(tt)
	if v := s.Value(); v != time.Hour+2*time.Minute+3*time.Second {
		t.Fatalf("1:2:3 made %v, want 1h2m3s", v)
	}
	if got := in.Value(); got != "01:02:03" {
		t.Fatalf("after leaving the field shows %q, want 01:02:03", got)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change"}) {
		t.Fatalf("leaving after 1:2:3 ran %q, want one change", got)
	}
	log.reset()
	spanRetype(in, "abc")
	spanLeave(tt)
	if v := s.Value(); v != time.Hour+2*time.Minute+3*time.Second {
		t.Fatalf("abc changed the value to %v", v)
	}
	if got := in.Value(); got != "01:02:03" {
		t.Fatalf("after abc the field shows %q, want the last value", got)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("abc ran %q", got)
	}
	// Focus in and out without typing changes nothing.
	in.Focus()
	spanLeave(tt)
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("passing through ran %q", got)
	}

	for _, c := range []struct {
		text string
		want time.Duration
	}{
		{"1:2", time.Hour + 2*time.Minute},
		{"5", 5 * time.Hour},
		{" 2:30 ", 2*time.Hour + 30*time.Minute},
		{"1.02:00:00", 26 * time.Hour}, // days, though not shown
		{"30:00:00", 30 * time.Hour},   // hours past 23
		{"10000:00:00", spanMax},       // clamped to the max
		{"-1:00:00", 0},                // clamped to the min
		{"1:60:00", 0},                 // minutes past 59: back to the last value
		{"1:00:60", 0},                 // seconds past 59
		{"99999999999:00:00", 0},       // too many hours for a Duration
		{"1:2:3:4", 0},                 // four parts
		{"1:", 0},                      // a part missing
		{"", 0},                        // nothing
		{"+1:00", 0},                   // a sign inside a part
		{"1:00.5", 0},                  // a fraction
		{"1.5", 29 * time.Hour},        // a day and five hours
		{"104166.00:00", spanMax},      // 2499984 hours: read, then clamped
		{"104167.00:00", 0},            // 2500008 hours: too many
		{"2500000:00", spanMax},        // the most hours read
		{"2500001:00", 0},              // one more
		{"18446744073709551617:00", 0}, // digits that would wrap an int64 to 1
	} {
		s.SetValue(0)
		tt.Settle()
		spanRetype(in, c.text)
		spanLeave(tt)
		if v := s.Value(); v != c.want {
			t.Fatalf("%q made %v, want %v", c.text, v, c.want)
		}
	}
}

// The frame in which the focus leaves shows the value read, or the last
// one: a window draws no frame after it unless something asks.
func TestTimeSpanLeaveShowsInTheSameFrame(t *testing.T) {
	s := TimeSpanEdit().SetValue(time.Hour)
	tt, in := spanTester(t, s)
	spanRetype(in, "abc")
	next := tt.Find(ByRole(ui.RoleButton).Label("Next"))
	next.Focus()
	if got := in.Value(); got != "01:00:00" {
		t.Fatalf("the frame the focus left in shows %q, want 01:00:00", got)
	}
}

// Passing through a field with its seconds hidden keeps them, and runs
// nothing.
func TestTimeSpanHiddenSecondsKeptOnLeave(t *testing.T) {
	var log editLog
	s := TimeSpanEdit().SetShowSeconds(false).SetValue(90*time.Minute + 45*time.Second).OnChange(log.add("change"))
	tt, in := spanTester(t, s)
	in.Focus()
	spanLeave(tt)
	if v := s.Value(); v != 90*time.Minute+45*time.Second {
		t.Fatalf("passing through made %v, want 1h30m45s", v)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("passing through ran %q", got)
	}
}

// A step keeps what the field hides: seconds, and fractions of a second.
func TestTimeSpanStepKeepsWhatIsHidden(t *testing.T) {
	s := TimeSpanEdit().SetShowSeconds(false).SetValue(90*time.Minute + 45*time.Second)
	_, in := spanTester(t, s)
	spanCaret(in, 0)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 150*time.Minute+45*time.Second {
		t.Fatalf("Up on the hours with seconds hidden made %v, want 2h30m45s", v)
	}
	s2 := TimeSpanEdit().SetValue(1500 * time.Millisecond)
	_, in2 := spanTester(t, s2)
	spanCaret(in2, 8)
	in2.Key(0, ui.KeyUp)
	if v := s2.Value(); v != 2500*time.Millisecond {
		t.Fatalf("Up on the seconds of 1.5s made %v, want 2.5s", v)
	}
}

// Enter commits the typed text before the form's accept button reads it.
func TestTimeSpanEnterInAForm(t *testing.T) {
	calmOff()
	var mu sync.Mutex
	var got []time.Duration
	f := NewForm("T", 400, 200)
	s := TimeSpanEdit()
	ok := Button("OK").OnClick(func() { mu.Lock(); got = append(got, s.Value()); mu.Unlock() })
	f.SetContent(VBox(0).Add(s, ok)).SetAcceptButton(ok)
	tt := TestForm(t, f)
	tt.Settle()
	in := tt.Find(ByRole(ui.RoleTextField))
	spanRetype(in, "1:30")
	in.Key(0, ui.KeyEnter)
	if got := in.Value(); got != "01:30:00" {
		t.Fatalf("after Enter the field shows %q, want 01:30:00", got)
	}
	flushEvents(t)
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(got, []time.Duration{90 * time.Minute}) {
		t.Fatalf("OK read %v, want 1h30m once", got)
	}
}

// In a form fitted to its content (a measuring pass each refit), leaving
// and stepping fire once.
func TestTimeSpanInAFittedForm(t *testing.T) {
	calmOff()
	var log editLog
	f := NewForm("T", 400, 0)
	s := TimeSpanEdit().OnChange(log.add("change"))
	f.SetContent(VBox(0).Add(s, Button("Next")))
	tt := TestForm(t, f)
	f.SetSize(400, 0)
	tt.Settle()
	in := tt.Find(ByRole(ui.RoleTextField))
	spanRetype(in, "2:00")
	tt.Find(ByRole(ui.RoleButton).Label("Next")).Focus()
	tt.Settle()
	if v := s.Value(); v != 2*time.Hour {
		t.Fatalf("leaving in a fitted form made %v", v)
	}
	spanCaret(in, 4)
	in.Key(0, ui.KeyUp)
	if v := s.Value(); v != 2*time.Hour+time.Minute {
		t.Fatalf("Up in a fitted form made %v, want 2h1m", v)
	}
	if got := log.all(t); !slices.Equal(got, []string{"change", "change"}) {
		t.Fatalf("in a fitted form ran %q, want two changes", got)
	}
}

// A measuring pass (a fitted form's refit) leaves the caret's request of a
// step to the window: it is a one-shot request.
func TestTimeSpanMeasureKeepsTheCaretRequest(t *testing.T) {
	s := TimeSpanEdit()
	spanTester(t, s)
	mu.Lock()
	s.putCaret = spanMinutes
	mu.Unlock()
	// What a fitted form's refit does (Form.measureFit).
	ui.View(View(s)).MeasureHeight(400, 1)
	mu.Lock()
	got := s.putCaret
	mu.Unlock()
	if got != spanMinutes {
		t.Fatalf("after a measure the caret's request is %d, want %d (minutes)", got, spanMinutes)
	}
}

func TestTimeSpanRange(t *testing.T) {
	s := TimeSpanEdit()
	if v := s.SetValue(-time.Second).Value(); v != 0 {
		t.Fatalf("the default min: -1s made %v, want 0", v)
	}
	if v := s.SetValue(10000 * time.Hour).Value(); v != spanMax {
		t.Fatalf("the default max: 10000h made %v, want 9999:59:59", v)
	}
	if v := s.SetValue(spanMax).Value(); v != spanMax {
		t.Fatalf("the max itself made %v", v)
	}
	s.SetRange(time.Hour, 2*time.Hour)
	if v := s.Value(); v != 2*time.Hour {
		t.Fatalf("SetRange(1h, 2h) left %v, want the value clamped to 2h", v)
	}
	if v := s.SetValue(5 * time.Hour).Value(); v != 2*time.Hour {
		t.Fatalf("SetValue(5h) in 1h..2h made %v", v)
	}
	if v := s.SetValue(0).Value(); v != time.Hour {
		t.Fatalf("SetValue(0) in 1h..2h made %v", v)
	}
	s.SetRange(3*time.Hour, time.Hour) // swapped
	for _, c := range []struct{ in, want time.Duration }{
		{90 * time.Minute, 90 * time.Minute},
		{4 * time.Hour, 3 * time.Hour},
		{0, time.Hour},
	} {
		if v := s.SetValue(c.in).Value(); v != c.want {
			t.Fatalf("SetRange(3h, 1h): SetValue(%v) made %v, want %v", c.in, v, c.want)
		}
	}
}

func TestTimeSpanSetterDoesNotFire(t *testing.T) {
	var log editLog
	s := TimeSpanEdit().OnChange(log.add("change"))
	tt, in := spanTester(t, s)
	s.SetValue(90 * time.Minute)
	tt.Settle()
	if got := in.Value(); got != "01:30:00" {
		t.Fatalf("the field shows %q, want 01:30:00", got)
	}
	s.SetRange(0, time.Hour) // clamps the value
	s.SetShowDays(true).SetShowSeconds(false)
	tt.Settle()
	if got := in.Value(); got != "0.01:00" {
		t.Fatalf("the field shows %q, want 0.01:00", got)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("setters ran %q", got)
	}
}

func TestTimeSpanFromGoroutine(t *testing.T) {
	s := TimeSpanEdit()
	tt, in := spanTester(t, s)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 40 {
			s.SetValue(time.Duration(i) * time.Minute).SetShowDays(i%2 == 0).SetShowSeconds(i%3 != 0)
			s.SetRange(0, time.Duration(100+i)*time.Hour)
			_ = s.Value()
		}
		s.SetShowDays(false).SetShowSeconds(true).SetValue(90 * time.Minute)
	}()
	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
			tt.Frame()
		}
	}
	tt.Settle()
	if got := in.Value(); got != "01:30:00" {
		t.Fatalf("the field shows %q, want 01:30:00", got)
	}
}

func TestTimeSpanBind(t *testing.T) {
	var mu sync.Mutex
	v := 2 * time.Hour
	get := func() time.Duration { mu.Lock(); defer mu.Unlock(); return v }
	s := Bind(TimeSpanEdit(), get, func(x time.Duration) { mu.Lock(); v = x; mu.Unlock() })
	tt, in := spanTester(t, s)
	if got := in.Value(); got != "02:00:00" {
		t.Fatalf("the field shows %q, not the source's 2h", got)
	}
	spanCaret(in, 4)
	in.Key(0, ui.KeyUp)
	flushEvents(t)
	if d := get(); d != 2*time.Hour+time.Minute {
		t.Fatalf("after Up the source has %v", d)
	}
	spanRetype(in, "3:00")
	spanLeave(tt)
	flushEvents(t)
	if d := get(); d != 3*time.Hour {
		t.Fatalf("after typing 3:00 and leaving the source has %v", d)
	}
	mu.Lock()
	v = 45 * time.Minute
	mu.Unlock()
	Reload(s)
	tt.Settle()
	if got := in.Value(); got != "00:45:00" {
		t.Fatalf("after Reload the field shows %q", got)
	}
	// A source value out of range is clamped, not written back.
	mu.Lock()
	v = -time.Hour
	mu.Unlock()
	Reload(s)
	flushEvents(t)
	if d, got := get(), s.Value(); d != -time.Hour || got != 0 {
		t.Fatalf("Reload of -1h: source %v, control %v; want -1h and 0", d, got)
	}
}

// A disabled TimeSpanEdit takes no keys and its buttons no click.
func TestTimeSpanDisabled(t *testing.T) {
	var log editLog
	s := TimeSpanEdit().SetValue(time.Hour).OnChange(log.add("change")).SetDisabled(true)
	tt, in := spanTester(t, s)
	for _, name := range []string{"Increase", "Decrease"} {
		r := tt.Find(ByRole(ui.RoleButton).Label(name)).Bounds()
		tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		tt.Settle()
	}
	r := in.Bounds()
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Key(0, ui.KeyUp)
	tt.Type("5")
	tt.Key(0, ui.KeyTab)
	tt.Settle()
	if v := s.Value(); v != time.Hour {
		t.Fatalf("a disabled TimeSpanEdit changed to %v", v)
	}
	if got := log.all(t); len(got) != 0 {
		t.Fatalf("a disabled TimeSpanEdit ran %q", got)
	}
}

// The buttons sit at the field's end, inside its frame: at the right, or
// the left right to left.
func TestTimeSpanButtonsAtTheEnd(t *testing.T) {
	s := TimeSpanEdit()
	tt, in := spanTester(t, s)
	check := func(rtl bool) {
		t.Helper()
		f := in.Bounds()
		left, right := float32(0), float32(300)
		if rtl {
			left, right = 100, 400
		}
		for _, name := range []string{"Decrease", "Increase"} {
			r := tt.Find(ByRole(ui.RoleButton).Label(name)).Bounds()
			if r.W <= 0 || r.X < left-0.5 || r.X+r.W > right+0.5 {
				t.Fatalf("rtl %v: %s at %v is outside the frame [%v, %v]", rtl, name, r, left, right)
			}
			if !rtl && r.X < f.X+f.W-0.5 || rtl && r.X+r.W > f.X+0.5 {
				t.Fatalf("rtl %v: %s at %v is not past the text %v", rtl, name, r, f)
			}
		}
	}
	check(false)
	tt.SetDirection(ui.RightToLeft)
	tt.Settle()
	check(true)
}

func TestTimeSpanAllocNothingAtRest(t *testing.T) {
	calmOff()
	box := ui.NewTester(content(TextBox().SetText("hello")), 400, 120)
	span := ui.NewTester(content(TimeSpanEdit().SetShowDays(true).SetValue(26*time.Hour+5*time.Second)), 400, 120)
	for range 10 {
		box.Frame()
		span.Frame()
	}
	want, got := allocrun.Least(30, box.Frame), allocrun.Least(30, span.Frame)
	if got > want {
		t.Fatalf("a frame of the TimeSpanEdit at rest allocates %v, more than a TextBox's %v", got, want)
	}
	t.Logf("a frame allocates %v, a TextBox's %v", got, want)
}
