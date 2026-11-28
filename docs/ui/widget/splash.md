# Splash screen

`ShowSplash(Splash{...})` shows a splash while the app starts, as
DevExpress's `SplashScreenManager`: a frameless window centered on the
screen, above other windows and with no taskbar button. It holds an image, a
title, a status line and a progress bar of unknown length, in the theme's
colors.

```go
func main() {
	splash := widget.ShowSplash(widget.Splash{Image: logo, Title: "Orders", Status: "Starting"})
	widget.Run("Orders", func(w *widget.Form) {
		splash.SetStatus("Loading the customers")
		w.SetContent(customers())
		splash.CloseWhenShown(w)
	})
}
```

- `Splash` has `Image` (a `*ui.Bitmap`, scaled down to fit, keeping its
  aspect ratio; nil for none), `Title`, `Status`, and `Width` and `Height` in
  DIPs. A side left 0 is the default's: 480 wide, 300 high.
- `ShowSplash` may be called from any goroutine, and in `main` before `Run`:
  it does not wait, and the window opens once the app is ready, before
  `Run`'s form. Work that holds the main goroutine before `Run` keeps the
  splash from showing until it ends: do it in `Run`'s func, or on a goroutine.
- `SetStatus(text)` changes the status line, from any goroutine; the window
  redraws on its next frame. Screen readers read it out, as a status.
- `CloseWhenShown(form)` closes the splash once the form has shown, at once
  if it shows already, so the form appears before the splash goes. Use it
  for the first form: an app quits when its last window closes, so a
  `Close()` before the form shows can end it.
- `Close()` closes the splash; a second call does nothing. A splash closed
  before its window opened never opens.
