package widget_test

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui/widget"
)

// The snippet of docs/ui/widget/timer.md, which must compile.
func ExampleTimer() {
	status := widget.Label("")
	var tick *widget.TimerWidget
	n := 0
	tick = widget.Timer(time.Second).OnTick(func() {
		n++
		status.SetText(fmt.Sprint("Ticks: ", n))
	}).Start()
	defer tick.Stop()
}
