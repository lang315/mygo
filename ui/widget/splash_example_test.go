package widget_test

import "github.com/egoist/mygo/ui/widget"

// The splash of docs/ui/widget/splash.md: shown in main before Run, and
// closed once the app's form has shown.
func ExampleShowSplash() {
	splash := widget.ShowSplash(widget.Splash{Title: "Orders", Status: "Starting"})
	widget.Run("Orders", func(w *widget.Form) {
		splash.SetStatus("Loading the customers")
		w.SetContent(widget.Label("Customers"))
		splash.CloseWhenShown(w)
	})
}
