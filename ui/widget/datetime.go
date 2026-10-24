package widget

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// DateTimeFormat says what a DateTimePicker edits.
type DateTimeFormat uint8

const (
	// DateAndTime shows a date field and a time field side by side.
	DateAndTime DateTimeFormat = iota
	// DateOnly shows the date field alone.
	DateOnly
	// TimeOnly shows the time field alone.
	TimeOnly
)

// DateTimePickerWidget edits a time.Time: a date field that opens a
// calendar, a time field, or both. It is ui.DateInput and ui.TimeInput
// underneath, in the formats of the user's region.
type DateTimePickerWidget struct {
	common[DateTimePickerWidget]
	value    mirror[time.Time]
	format   DateTimeFormat
	onChange func()
}

// DateTimePicker makes a picker of the current time, for a date and a
// time.
func DateTimePicker() *DateTimePickerWidget {
	p := &DateTimePickerWidget{}
	p.value.set(time.Now())
	p.init(p)
	return p
}

// Value returns the time as the user left it. Choosing a day keeps the
// time of day and the location of the value, and changing the time keeps
// the day.
func (p *DateTimePickerWidget) Value() time.Time {
	mu.Lock()
	defer mu.Unlock()
	return p.value.value
}

// SetValue changes the time, without running OnChange.
func (p *DateTimePickerWidget) SetValue(t time.Time) *DateTimePickerWidget {
	p.change(func() { p.value.set(t) })
	return p
}

// SetFormat chooses what the picker shows: DateAndTime (the default),
// DateOnly or TimeOnly.
func (p *DateTimePickerWidget) SetFormat(f DateTimeFormat) *DateTimePickerWidget {
	p.change(func() { p.format = f })
	return p
}

// OnChange runs fn, on the event goroutine, after the user changed the
// time. A setter does not run it.
func (p *DateTimePickerWidget) OnChange(fn func()) *DateTimePickerWidget {
	p.change(func() { p.onChange = fn })
	return p
}

func (p *DateTimePickerWidget) build(c *ui.Context) ui.Element {
	v, changed := p.value.bind()
	var moved bool
	row := ui.Row(c).Gap(c.Theme().Space(2)).Children(func() {
		if p.format != TimeOnly {
			moved = ui.DateInput(c, v).Changed() || moved
		}
		if p.format != DateOnly {
			moved = ui.TimeInput(c, v).Changed() || moved
		}
	})
	reportDate(&p.value, changed, moved, p.onChange)
	return row
}

// reportDate runs h, on the event goroutine, when the user changed the
// date m holds: ui changed it since the last build (changed) or in this
// one (moved). Under mu, while building.
func reportDate(m *mirror[time.Time], changed, moved bool, h func()) {
	if moved {
		m.value, m.dirty = m.shown, false
		changed = true
	}
	if changed && h != nil {
		Post(h)
	}
}

// MonthCalendarWidget shows a month and chooses a day of a time.Time. It
// is ui.Calendar underneath.
type MonthCalendarWidget struct {
	common[MonthCalendarWidget]
	value    mirror[time.Time]
	onChange func()
}

// MonthCalendar makes a calendar of the current day.
func MonthCalendar() *MonthCalendarWidget {
	m := &MonthCalendarWidget{}
	m.value.set(time.Now())
	m.init(m)
	return m
}

// Value returns the chosen day, with the time of day and the location of
// the value kept.
func (m *MonthCalendarWidget) Value() time.Time {
	mu.Lock()
	defer mu.Unlock()
	return m.value.value
}

// SetValue chooses a day, without running OnChange, and shows its month.
func (m *MonthCalendarWidget) SetValue(t time.Time) *MonthCalendarWidget {
	m.change(func() { m.value.set(t) })
	return m
}

// OnChange runs fn, on the event goroutine, after the user chose a day.
// Paging to another month is not a choice. A setter does not run it.
func (m *MonthCalendarWidget) OnChange(fn func()) *MonthCalendarWidget {
	m.change(func() { m.onChange = fn })
	return m
}

func (m *MonthCalendarWidget) build(c *ui.Context) ui.Element {
	v, changed := m.value.bind()
	e := ui.Calendar(c, v)
	reportDate(&m.value, changed, e.Changed(), m.onChange)
	return e
}
