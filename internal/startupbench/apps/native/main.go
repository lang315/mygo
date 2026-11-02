// Native is counter-native with the milestones of its start written to the
// file MYGO_STARTUP_LOG (a measurement of internal/startupbench).
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func mark(name string) {
	if p := os.Getenv("MYGO_STARTUP_LOG"); p != "" {
		if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "%s %d\n", name, time.Now().UnixNano())
			f.Close()
		}
	}
}

type counter struct{ n int }

func (s *counter) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(16).Children(func() {
		ui.Textf(c, "%d", s.n).FontSize(56).Bold()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c.Key("decrement"), "−").Label("Decrement").Width(44).Clicked() {
				s.n--
			}
			if ui.PrimaryButton(c.Key("increment"), "+").Label("Increment").Width(44).Clicked() {
				s.n++
			}
		})
		ui.Text(c, "↑ and ↓ count too").FontSize(12).TextColor(t.TextMuted)
	})
}

func main() {
	mark("main")
	s := &counter{}
	mygo.App.OnWindowCreated(func(w *mygo.Window) {
		w.OnShow(func() { mark("show") })
		w.OnReadyToShow(func() {
			mark("ready")
			mygo.App.Quit()
		})
	})
	mygo.App.WhenReady(func() {
		mark("appready")
		mygo.NewWindow(mygo.WindowOptions{Title: "Counter", Width: 360, Height: 260, Content: ui.View(s.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
