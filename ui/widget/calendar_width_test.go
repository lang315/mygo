package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// calendarDays returns the left and the right edge of the first week's day cells
// of meeting's month (weeks start on Sunday in the test's region): the
// day grid's own width, which no container stretches.
func calendarDays(tt *Tester) (left, right float32) {
	first := tt.Find(ByLabel("December 28, 2025")).Bounds()
	last := tt.Find(ByLabel("January 3, 2026")).Bounds()
	return first.X, last.X + last.W
}

// A MonthCalendar in a stretching container is as wide as its days.
func TestCalendarDoesNotStretch(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tt := NewTester(t, VBox(0).Add(MonthCalendar().SetValue(meeting)), 1200, 600)
		tt.SetDark(dark)
		cal := tt.Find(ByRole(ui.RoleTable)).Bounds()
		left, right := calendarDays(tt)
		next := tt.Find(ByLabel("Next month")).Bounds()
		if d := cal.W - (right - left); d < -1 || d > 1 {
			t.Errorf("dark=%v: calendar %v DIPs wide, its days %v", dark, cal.W, right-left)
		}
		if next.X+next.W > right+1 {
			t.Errorf("dark=%v: Next month ends at %v, past the days' end %v", dark, next.X+next.W, right)
		}
	}
}

// An explicit width applies: the header spans it, and the days are not cut.
func TestCalendarHeaderInFixedWidth(t *testing.T) {
	tt := NewTester(t, VBox(0).Add(MonthCalendar().SetValue(meeting).SetSize(400, 0)), 1200, 600)
	cal := tt.Find(ByRole(ui.RoleTable)).Bounds()
	left, right := calendarDays(tt)
	next := tt.Find(ByLabel("Next month")).Bounds()
	if d := cal.W - 400; d < -1 || d > 1 {
		t.Errorf("calendar %v DIPs wide, want 400", cal.W)
	}
	if end := next.X + next.W - cal.X; end < 399 || end > 401 {
		t.Errorf("Next month ends %v DIPs into the calendar, want 400", end)
	}
	if left < cal.X-1 || right > cal.X+cal.W+1 {
		t.Errorf("days %v..%v are cut by the calendar %v", left, right, cal)
	}
}

// The calendar keeps its natural width without overriding what its
// container does with a child: Pack's room centers it, so it is as wide as
// its days and centered.
func TestCalendarFollowsItsContainersAlignment(t *testing.T) {
	tt := NewTester(t, VBox(0).Pack(MonthCalendar().SetValue(meeting), true, false), 1200, 600)
	cal := tt.Find(ByRole(ui.RoleTable)).Bounds()
	left, right := calendarDays(tt)
	if d := cal.W - (right - left); d < -1 || d > 1 {
		t.Errorf("calendar %v DIPs wide, its days %v", cal.W, right-left)
	}
	if mid := cal.X + cal.W/2; mid < 599 || mid > 601 {
		t.Errorf("calendar's center at %v, the container's at 600", mid)
	}
}

// The date picker's popup keeps the width it had: its days' width.
func TestDatePickerPopupUnchanged(t *testing.T) {
	tt := NewTester(t, DateTimePicker().SetFormat(DateOnly).SetValue(meeting), 1200, 600)
	tt.Find(ByRole(ui.RolePopUpButton)).Click()
	cal := tt.Find(ByRole(ui.RoleTable)).Bounds()
	const want = 224 // measured on origin/main before the change
	if cal.W != want {
		t.Errorf("popup calendar %v DIPs wide, want %v", cal.W, want)
	}
}
