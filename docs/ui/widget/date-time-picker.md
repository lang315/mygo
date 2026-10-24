# DateTimePicker and MonthCalendar

Both edit a `time.Time` and follow the formats of the user's region
(`1/5/2026` in the United States, `05/01/2026` in the United Kingdom). They are
[`ui.DateInput`](../date-input.md), [`ui.TimeInput`](../time-input.md) and
[`ui.Calendar`](../calendar.md) underneath.

## DateTimePicker

```go
var due time.Time
picker := widget.DateTimePicker().
	SetValue(time.Now().Add(24 * time.Hour)).
	OnChange(func() { due = picker.Value() })
```

A date field opens a calendar below it on a click, Enter or Space; a time
field has its hours, minutes and AM/PM (or 24 hours) to edit with the arrows.

- `SetFormat(widget.DateAndTime)` is the default; `DateOnly` and `TimeOnly`
  show one field.
- `Value()` and `SetValue(t)`. Choosing a day keeps the time of day and the
  location of the value; editing the time keeps the day.
- `OnChange(fn)` runs on the event goroutine after the user changed the time,
  once per edit. A setter does not run it.

`MinDate` and `MaxDate` of WinForms are not here: `ui.DateInput` has no range
yet.

## MonthCalendar

```go
cal := widget.MonthCalendar().SetValue(day).OnChange(func() { show(cal.Value()) })
```

A month with the arrows, Page Up and Page Down, Home and End, and a click on a
day to choose it. `Value()`, `SetValue(t)` (which shows the value's month) and
`OnChange(fn)` work as above. As in `ui.Calendar`, the month buttons and Page
Up/Down move the chosen day by a month and report it.

In tests, days are buttons named by their dates:
`tt.Find(widget.ByLabel("January 12, 2026")).Click()`
(see [`widget.Tester`](tester.md)).
