// Skin-studio shows every control of ui/widget in every skin, palette and
// mode: three combo boxes in the tool strip switch them while the app runs.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/widget"
)

func main() {
	mygo.App.SetName("Skin studio")
	mygo.App.WhenReady(func() {
		widget.Post(func() { newStudio().Show() })
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
