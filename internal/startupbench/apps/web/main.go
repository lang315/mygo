// Web is hello with the milestones of its start written to the file
// MYGO_STARTUP_LOG (a measurement of internal/startupbench).
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egoist/mygo"
)

func mark(name string) {
	if p := os.Getenv("MYGO_STARTUP_LOG"); p != "" {
		if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintf(f, "%s %d\n", name, time.Now().UnixNano())
			f.Close()
		}
	}
}

type Greeter struct{}

func (Greeter) Greet(name string) string { return "Hello " + name }

const page = `<!doctype html><html><head><title>Hello MyGo</title></head><body><h1>Hello MyGo</h1>
<p id="greeting">…</p><script>mygo.call("Greeter.Greet", "world").then((t) => { document.getElementById("greeting").textContent = t; });</script></body></html>`

func main() {
	mark("main")
	mygo.Bind(Greeter{})
	mygo.App.OnWindowCreated(func(w *mygo.Window) {
		w.OnShow(func() { mark("show") })
		w.Page().OnDOMReady(func() { mark("dom") })
		w.OnReadyToShow(func() {
			mark("ready")
			mygo.App.Quit()
		})
	})
	mygo.App.WhenReady(func() {
		mark("appready")
		win := mygo.NewWindow(mygo.WindowOptions{Width: 480, Height: 320})
		win.Page().LoadHTML(page, "")
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
