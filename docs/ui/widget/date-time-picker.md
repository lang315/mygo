# DateTimePicker and MonthCalendar

Both edit a `time.Time` and follow the formats of the user's region
(`1/5/2026` in the United States, `05/01/2026` in the United Kingdom). They are
[`ui.DateInput`](../date-input.md), [`ui.TimeInput`](../time-input.md) and
[`ui.Calendar`](../calendar.md) underneath.

## DateTimePicker

```go
var due time.Time
var picker *widget.DateTimePickerWidget
picker = widget.DateTimePicker().
	SetValue(time.Now().Add(24 * time.Hour)).
	OnChange(func() { due = picker.Value() })
```

A date field opens a calendar below it on a click, Enter or Space; a time
field has its hours, minutes and AM/PM (or 24 hours) to edit with the arrows.

- `SetFormat(widget.DateAndTime)` is the default; `DateOnly` and `TimeOnly`
  show one field.
- `Value()` and `SetValue(t)`. Choosing a day keeps the time of day and the
  location of the value (the wall-clock time, where the day has it: in a day
  that lacks 02:30 to a clock change, the time is the one the zone gives);
  editing the time keeps the day. There is no empty value (WinForms'
  `ShowCheckBox`): a new picker holds the current time, so set the value you
  mean, or `time.Time{}` shows 1/1/1.
- `OnChange(fn)` runs on the event goroutine after the user changed the time,
  once per edit. A setter does not run it.

`MinDate` and `MaxDate` of WinForms are not here: `ui.DateInput` has no range
yet.

## MonthCalendar

```go
var cal *widget.MonthCalendarWidget
cal = widget.MonthCalendar().SetValue(day).OnChange(func() { show(cal.Value()) })
```

A month with the arrows, Page Up and Page Down, Home and End, and a click on a
day to choose it. All of these choose the day they land on; Page Up/Down move
it a month, clamped to the end of a shorter month. The month buttons only page:
they change the month shown, not the day chosen, so `OnChange` does not run for
them, and paging from the 31st neither skips a month nor stays in one. After
paging, the keys act on the month shown. `Value()`, `SetValue(t)` (which shows
the value's month) and `OnChange(fn)` work as above.

In tests, days are buttons named by their dates:
`tt.Find(widget.ByLabel("January 12, 2026")).Click()`
(see [`widget.Tester`](tester.md)).
