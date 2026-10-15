package widget

// Dialog is a window over another, its parent, which it blocks while it
// is open: a sheet on the parent on macOS, a modal window on Linux (which
// blocks every window of the app) and Windows. Make it with NewDialog and
// show it with Run; embed *Dialog in a type of your own for a dialog that
// returns a value, as gota's dialogs do. Escape closes it as its close
// button does.
type Dialog struct {
	*Window
	title, subtitle *LabelWidget
	onBeforeRun     func()
}

// NewDialog makes a dialog over parent (nil for none) titled title, with
// subtitle under it ("" for none), showing content in a window width by
// height DIPs, its title bar included: a macOS sheet, which has none,
// shows that much shorter.
// The title is drawn in the dialog too, as a macOS sheet shows none. To
// open a dialog from a dialog, pass the outer one's Window as parent.
func NewDialog(parent *Window, title, subtitle string, width, height int, content Widget) *Dialog {
	d := &Dialog{
		Window:   NewWindow(title, width, height),
		title:    Label(title).Bold().TextSize(LG),
		subtitle: Label(subtitle).Muted().SetVisible(subtitle != ""),
	}
	d.parent, d.dialog = parent, true
	d.SetContent(VBox(8).Add(d.title, d.subtitle).Pack(content, true, true).Padding(4))
	return d
}

// SetTitle changes the title, in the window's title bar and in the dialog.
func (d *Dialog) SetTitle(s string) *Dialog {
	d.title.SetText(s)
	d.Window.SetTitle(s)
	return d
}

// SetSubtitle changes the line under the title; "" hides it.
func (d *Dialog) SetSubtitle(s string) *Dialog {
	d.subtitle.SetText(s).SetVisible(s != "")
	return d
}

// OnBeforeRun runs fn in Run, before the dialog shows.
func (d *Dialog) OnBeforeRun(fn func()) *Dialog { change(func() { d.onBeforeRun = fn }); return d }

// SetDeletable lets the user close the dialog with the window's close
// button or keys, or not. Close and the parent closing close it anyway.
func (d *Dialog) SetDeletable(on bool) *Dialog { change(func() { d.deletable = on }); return d }

// SetResizable lets the user resize the dialog, or not.
func (d *Dialog) SetResizable(on bool) *Dialog { d.Window.SetResizable(on); return d }

// Run shows the dialog and returns when it closes: by Close, by the user,
// or with its parent. In a handler it runs the other handlers meanwhile,
// as gtk_dialog_run runs the main loop, so the dialog's buttons work; on
// another goroutine it waits; on the main thread it panics.
func (d *Dialog) Run() {
	notMain("Dialog.Run")
	mu.Lock()
	before := d.onBeforeRun
	mu.Unlock()
	if before != nil {
		before()
	}
	d.Show()
	mu.Lock()
	done := d.done
	mu.Unlock()
	block("Dialog.Run", done)
}
