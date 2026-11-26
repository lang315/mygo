# ProgressPanel and the wait form

`ProgressPanel(caption)` shows that something is busy: a
[spinner](../spinner.md) beside a caption, in bold, with a description under
it, as DevExpress's ProgressPanel. Assistive technology reads it as a status,
whose texts screen readers read out as they change.

```go
panel := widget.ProgressPanel("Loading").SetDescription("Reading the customers")
```

- `SetCaption(s)` sets the bold text; `SetDescription(s)` the text under it, ""
  for none.

## The wait form

`Form.ShowWait(caption, description)` shows a progress panel centred over the
form's content, which it dims, as DevExpress's
`SplashScreenManager.ShowWaitForm`. Clicks and keys do not reach the content,
the form's shortcuts and its accept button included, until
`Form.CloseWait()`. Both are safe from any goroutine:

```go
go func() {
	w.ShowWait("Loading", "Reading the customers")
	defer w.CloseWait()
	load()
}()
```

- A second `ShowWait` changes the panel's text; it adds no second panel.
- `CloseWait` without a wait does nothing.
- A form not shown yet waits once it shows. A dialog is a form: `d.ShowWait`
  works on it too.

An app that never calls `ShowWait` links none of the wait's drawing.
