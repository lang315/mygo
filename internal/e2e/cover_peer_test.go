//go:build !js

package e2e

import (
	"bufio"
	"fmt"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// coverPeer is a process with an always-on-top window over the screen
// rectangle MYGO_COVER_BOUNDS gives ("x,y,width,height"), another app's
// window over a point a test clicks. It prints "ready" once the window is
// made and quits when its stdin is closed or written to.
func coverPeer() {
	var x, y, w, h int
	if _, err := fmt.Sscanf(os.Getenv("MYGO_COVER_BOUNDS"), "%d,%d,%d,%d", &x, &y, &w, &h); err != nil {
		fmt.Fprintln(os.Stderr, "cover peer:", err)
		os.Exit(1)
	}
	mygo.App.SetName("MyGoCoverPeer")
	mygo.App.WhenReady(func() {
		go func() {
			// Shown without activating: the foreground stays where it was, and
			// is not left to another app when this process exits.
			mygo.NewWindow(mygo.WindowOptions{Title: "Cover", X: x, Y: y, Width: w, Height: h, AlwaysOnTop: true, Hidden: true,
				Content: ui.View(func(c *ui.Context) { ui.Text(c, "cover") })}).ShowInactive()
			fmt.Println("ready")
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
