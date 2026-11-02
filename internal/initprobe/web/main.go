// Web imports what hello does and starts nothing (TEMPORARY).
package main

import (
	"os"

	"github.com/egoist/mygo"
)

type Greeter struct{}

func (Greeter) Greet(name string) string { return "Hello " + name }

func main() {
	if os.Getenv("INITPROBE_RUN") == "" {
		return
	}
	mygo.Bind(Greeter{})
	listModules("web at main")
	mygo.App.WhenReady(func() {
		listModules("web after App ready")
		mygo.App.Quit()
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
