package widget

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

var meeting = time.Date(2026, 1, 5, 14, 30, 0, 0, time.UTC)

func TestDateTimePickerShowsDateAndTimeByDefault(t *testing.T) {
	tt := NewTester(t, DateTimePicker().SetValue(meeting), 800, 600)
	if !tt.Has(ByRole(ui.RolePopUpButton).Label("1/5/2026")) || !tt.Has(ByLabel("hours")) {
		t.Fatalf("default picker lacks the date or the time: %+v", tt.Nodes())
	}
}

func TestDateTimePickerFormats(t *testing.T) {
	date := NewTester(t, DateTimePicker().SetFormat(DateOnly).SetValue(meeting), 800, 600)
	if !date.Has(ByLabel("1/5/2026")) || date.Has(ByLabel("hours")) {
		t.Errorf("DateOnly: %q", date.Texts())
	}
	tm := NewTester(t, DateTimePicker().SetFormat(TimeOnly).SetValue(meeting), 800, 600)
	if tm.Has(ByLabel("1/5/2026")) || !tm.Has(ByLabel("hours")) {
		t.Errorf("TimeOnly: %+v", tm.Nodes())
	}
}

func TestDateTimePickerChoosingADayKeepsTheTimeAndReportsOnce(t *testing.T) {
	changed := make(chan time.Time, 4)
	var p *DateTimePickerWidget
	p = DateTimePicker().SetValue(meeting).OnChange(func() { changed <- p.Value() })
	tt := NewTester(t, p, 800, 600)
	tt.Find(ByRole(ui.RolePopUpButton)).Click()
	tt.Find(ByLabel("January 12, 2026")).Click()
	got := received(t, changed)
	want := time.Date(2026, 1, 12, 14, 30, 0, 0, time.UTC)
	if !got.Equal(want) || !p.Value().Equal(want) {
		t.Fatalf("OnChange saw %v, Value %v, want %v", got, p.Value(), want)
	}
	if got.Location() != time.UTC {
		t.Errorf("location %v", got.Location())
	}
	tt.Settle()
	select {
	case <-changed:
		t.Fatal("one choice reported twice")
	default:
	}
}

func TestDateTimePickerTimeSegmentsChangeTheTime(t *testing.T) {
	changed := make(chan struct{}, 4)
	p := DateTimePicker().SetValue(meeting).OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, p, 800, 600)
	tt.Find(ByLabel("hours")).Key(0, ui.KeyUp)
	received(t, changed)
	if got := p.Value(); got.Hour() != 15 || got.Minute() != 30 || got.Day() != 5 {
		t.Fatalf("after Up on the hours: %v", got)
	}
}

func TestDateTimePickerSettersDoNotReportAndGettersFollowThem(t *testing.T) {
	changed := make(chan struct{}, 2)
	p := DateTimePicker().OnChange(func() { changed <- struct{}{} })
	tt := NewTester(t, p, 800, 600)
	p.SetValue(meeting)
	tt.Settle()
	if !p.Value().Equal(meeting) || !tt.Has(ByLabel("1/5/2026")) {
		t.Fatalf("value %v, nodes %+v", p.Value(), tt.Nodes())
	}
	select {
	case <-changed:
		t.Fatal("SetValue ran OnChange")
	default:
	}
}

func TestDateTimePickerDisabledCannotBeClicked(t *testing.T) {
	tt := NewTester(t, DateTimePicker().SetValue(meeting).SetDisabled(true), 800, 600)
	if !tt.Find(ByRole(ui.RolePopUpButton)).Disabled() {
		t.Fatal("the date field is not disabled")
	}
}

func TestMonthCalendarChoosesADay(t *testing.T) {
	changed := make(chan time.Time, 4)
	var c *MonthCalendarWidget
	c = MonthCalendar().SetValue(meeting).OnChange(func() { changed <- c.Value() })
	tt := NewTester(t, c, 800, 600)
	tt.Find(ByLabel("January 20, 2026")).Click()
	want := time.Date(2026, 1, 20, 14, 30, 0, 0, time.UTC)
	if got := received(t, changed); !got.Equal(want) {
		t.Fatalf("OnChange saw %v, want %v", got, want)
	}
}

// ui.Calendar's month buttons move the chosen date by a month, as Page Up
// does, so the value follows what the calendar shows.
func TestMonthCalendarPagingMovesTheValueByAMonth(t *testing.T) {
	changed := make(chan time.Time, 2)
	var c *MonthCalendarWidget
	c = MonthCalendar().SetValue(meeting).OnChange(func() { changed <- c.Value() })
	tt := NewTester(t, c, 800, 600)
	tt.Find(ByLabel("Next month")).Click()
	if !tt.Has(ByLabel("February 10, 2026")) {
		t.Fatal("the calendar did not page")
	}
	want := time.Date(2026, 2, 5, 14, 30, 0, 0, time.UTC)
	if got := received(t, changed); !got.Equal(want) || !c.Value().Equal(want) {
		t.Fatalf("OnChange saw %v, Value %v, want %v", got, c.Value(), want)
	}
}

func TestMonthCalendarSetValueShowsItsMonth(t *testing.T) {
	c := MonthCalendar().SetValue(meeting)
	tt := NewTester(t, c, 800, 600)
	c.SetValue(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC))
	tt.Settle()
	if !tt.Has(ByLabel("March 9, 2026")) {
		t.Fatal("the calendar still shows January")
	}
}
