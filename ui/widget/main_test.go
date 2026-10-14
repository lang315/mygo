package widget

import (
	"fmt"
	"os"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

// fb is the fake backend the tests run mygo on.
var fb *fake.Backend

func TestMain(m *testing.M) {
	fb = fake.New()
	platform.TestBackend = fb
	mygo.App.SetName("WidgetTest")
	mygo.App.OnWindowAllClosed(func() {}) // keep running when tests close windows
	mygo.App.WhenReady(func() {
		go func() { os.Exit(m.Run()) }()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// tester shows w alone in a headless window.
func tester(t *testing.T, w Widget) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(content(w), 800, 600)
	tt.Frame()
	return tt
}
