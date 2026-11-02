// Native imports what counter-native does and starts nothing (TEMPORARY, a
// measurement of GODEBUG=inittrace=1).
package main

import (
	"os"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/ui"
)

func main() {
	if os.Getenv("INITPROBE_RUN") == "" {
		return
	}
	listModules("native at main")
	mygo.App.WhenReady(func() {
		listModules("native after App ready")
		mygo.App.Quit()
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
