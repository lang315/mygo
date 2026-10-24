package widget_test

import (
	"time"

	"github.com/egoist/mygo/ui/widget"
)

// The snippets of docs/ui/widget/date-time-picker.md, which must compile.
func ExampleDateTimePicker() {
	var due time.Time
	var picker *widget.DateTimePickerWidget
	picker = widget.DateTimePicker().
		SetValue(time.Now().Add(24 * time.Hour)).
		OnChange(func() { due = picker.Value() })
	_ = due
}

func ExampleMonthCalendar() {
	var cal *widget.MonthCalendarWidget
	show := func(time.Time) {}
	cal = widget.MonthCalendar().SetValue(time.Now()).OnChange(func() { show(cal.Value()) })
}
