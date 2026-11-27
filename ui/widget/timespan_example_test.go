package widget_test

import (
	"time"

	"github.com/egoist/mygo/ui/widget"
)

// The TimeSpanEdit of docs/ui/widget/timespanedit.md: a timeout from a
// second to two hours, its days shown, bound to a setting.
func ExampleTimeSpanEdit() {
	timeout := 90 * time.Second
	var field *widget.TimeSpanEditWidget
	field = widget.TimeSpanEdit().
		SetRange(time.Second, 2*time.Hour).
		SetShowDays(true).
		SetShowSeconds(true).
		SetValue(timeout).
		OnChange(func() { println(field.Value().String()) })
	widget.Bind(field, func() time.Duration { return timeout }, func(d time.Duration) { timeout = d })
}
