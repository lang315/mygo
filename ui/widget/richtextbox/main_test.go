package richtextbox

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// fb is the fake backend the tests run mygo on.
var fb *fake.Backend

func TestMain(m *testing.M) {
	fb = fake.New()
	platform.TestBackend = fb
	mygo.App.SetName("RichTextBoxTest")
	mygo.App.OnWindowAllClosed(func() {}) // keep running when tests close windows
	mygo.App.WhenReady(func() {
		go func() { os.Exit(m.Run()) }()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
