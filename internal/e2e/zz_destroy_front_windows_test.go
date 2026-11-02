//go:build windows

package e2e

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// BISECT (throwaway): after the focused window A of an app is destroyed, does
// the front go to its other window B? With a call of A held in Go or not.
func TestZZDestroyFocusedWindow(t *testing.T) {
	for _, hold := range []bool{false, true} {
		name := "no held call"
		if hold {
			name = "held call"
		}
		t.Run(name, func(t *testing.T) {
			b := mygo.NewWindow(mygo.WindowOptions{Title: "B", Width: 300, Height: 200})
			t.Cleanup(b.Destroy)
			a := mygo.NewWindow(mygo.WindowOptions{Title: "A", Width: 300, Height: 200})
			t.Cleanup(a.Destroy)
			if err := a.Page().LoadURL("app://localhost/"); err != nil {
				t.Fatal(err)
			}
			waitFor(t, a, "window.run && window.mygo")
			if err := b.Page().LoadURL("app://localhost/"); err != nil {
				t.Fatal(err)
			}
			waitFor(t, b, "window.run && window.mygo")
			grantForeground()
			a.Focus()
			wait.Poll(a.IsFocused)
			t.Logf("A focused %v; front: %s", a.IsFocused(), foregroundState(a))
			if hold {
				benchHold = make(chan struct{})
				if _, err := a.Page().Eval("window.held = mygo.call('Bench.Hold').catch(() => 'rejected'); 0"); err != nil {
					t.Fatal(err)
				}
				<-benchHeld
			}
			start := time.Now()
			a.Destroy()
			got := wait.Poll(b.IsFocused)
			t.Logf("BISECT %s: after A is destroyed B focused=%v (after %v); front: %s", name, got, time.Since(start).Round(time.Millisecond), foregroundState(b))
			if hold {
				close(benchHold)
			}
		})
	}
}
