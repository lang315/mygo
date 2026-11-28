//go:build (darwin && !ios) || (windows && (amd64 || arm64))

package e2e

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// Activation clicks must preserve a player's state when requested, for
// both kinds of content and for pages embedded in native UI.
func TestConsumeActivationClick(t *testing.T) {
	for _, kind := range []string{"page", "native", "embedded"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", kind, enabled), func(t *testing.T) {
				var clicks, pressesAtSecondClick, frames, viewFrames, nativeDowns, nativeUps atomic.Int32
				var view *mygo.WebView
				opts := mygo.WindowOptions{
					Width: 300, Height: 200, X: 40, Y: 40,
					ConsumeActivationClick: enabled,
				}
				if kind != "page" {
					opts.Content = ui.View(func(c *ui.Context) {
						frames.Add(1)
						if kind == "native" {
							ui.Row(c).Fill().AlignItems(ui.Stretch).ObserveInput(func(ev ui.InputEvent) {
								switch ev.Kind {
								case ui.InputPointerDown:
									nativeDowns.Add(1)
								case ui.InputPointerUp:
									nativeUps.Add(1)
								}
							}).Children(func() {
								if ui.Box(c).Grow(1).Clicked() {
									clicks.Add(1)
								}
								if ui.Box(c).Grow(1).Clicked() {
									pressesAtSecondClick.Store(clicks.Add(1))
								}
							})
						} else if view != nil {
							viewFrames.Add(1)
							ui.WebView(c, view).Fill()
						}
					})
				}
				w := newWindow(t, opts)
				page := w.Page()
				if kind == "embedded" {
					v, err := w.NewWebView(mygo.WebViewOptions{})
					if err != nil {
						t.Fatal(err)
					}
					w.Update(func() {
						frames.Store(0)
						view = v
					})
					page = v.Page()
				}
				if kind != "page" {
					eventually(t, "the native frame built", func() bool { return frames.Load() > 0 })
				}
				if kind == "embedded" {
					// Update posts its function: frames can count a build from
					// before it, without the web view.
					eventually(t, "a frame built with the web view", func() bool { return viewFrames.Load() > 0 })
				}
				// The frames built with the web view as each click went, for a
				// failure.
				var viewFramesAt [2]int32
				if page != nil {
					page.LoadHTML(`<body style="margin:0;height:100vh"
						onmousedown="window.presses++; if (event.clientX >= 150) window.pressesAtSecondClick = window.presses"
						onmouseup="window.ups++; if (event.clientX >= 150) window.upsAtSecondClick = window.ups"
						onclick="window.clicks++; if (event.clientX >= 150) window.clicksAtSecondClick = window.clicks"
						onkeydown="window.keys++">
						<script>window.presses=0; window.ups=0; window.clicks=0; window.keys=0</script>`, "")
					waitForPage(t, page, "window.presses === 0")
					// What the page saw, for a failure: none of the second
					// click, half of it, or all of it at the wrong place.
					// Deferred, as the test's context ends before its cleanups.
					defer func() {
						if t.Failed() {
							v, err := page.EvalContext(wait.Context(t), `JSON.stringify([window.presses, window.ups, window.clicks, window.pressesAtSecondClick, window.upsAtSecondClick, window.clicksAtSecondClick, document.hasFocus()])`)
							t.Logf("the page's presses, ups, clicks, the same at the second click, and whether it has the keyboard: %v (%v); frames built with the web view at the clicks %v, at the failure %d", v, err, viewFramesAt, viewFrames.Load())
						}
					}()
				}
				other := newWindow(t, mygo.WindowOptions{Width: 200, Height: 150, X: 400, Y: 40, Content: ui.View(func(c *ui.Context) { ui.Text(c, "Other") })})
				other.Focus()
				requireForeground(t, other)
				eventually(t, "the other window focused", other.IsFocused)
				checkActivationHit(t, w, kind, enabled)
				viewFramesAt[0] = viewFrames.Load()
				if !appClick(t, w, 100, 100) {
					t.Skip("click automation not available on this platform")
				}
				eventually(t, "the clicked window focused", w.IsFocused)
				if kind == "page" {
					waitForPage(t, page, "document.hasFocus()")
					if !pressPageLetter(w, 'K') {
						t.Skip("key automation not available on this platform")
					}
					waitForPage(t, page, "window.keys === 1")
				}
				// Use a different target for the second click. Its handler
				// records the count after both presses have been handled,
				// so a transient count of one cannot satisfy the assertion.
				viewFramesAt[1] = viewFrames.Load()
				appClick(t, w, 200, 100)
				want := int32(2)
				if enabled {
					want = 1
				}
				if page == nil {
					eventually(t, "the second native click handled", func() bool { return pressesAtSecondClick.Load() > 0 })
					if got := pressesAtSecondClick.Load(); got != want {
						t.Fatalf("clicks after the second click = %d, want %d", got, want)
					}
					if got := nativeDowns.Load(); got != want {
						t.Fatalf("pointer downs after the second click = %d, want %d", got, want)
					}
					if got := nativeUps.Load(); got != want {
						t.Fatalf("pointer ups after the second click = %d, want %d", got, want)
					}
				} else {
					waitForPage(t, page, "Number.isInteger(window.pressesAtSecondClick) && Number.isInteger(window.upsAtSecondClick) && Number.isInteger(window.clicksAtSecondClick)")
					for _, expr := range []string{"window.pressesAtSecondClick", "window.upsAtSecondClick", "window.clicksAtSecondClick"} {
						n, err := mygo.EvalAs[int](page, expr)
						if err != nil {
							t.Fatal(err)
						}
						if int32(n) != want {
							t.Fatalf("%s = %d, want %d", expr, n, want)
						}
					}
				}
			})
		}
	}
}
