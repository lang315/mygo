package widget

// Dialog is a window over another, its parent, which it blocks while it
// is open: a sheet on the parent on macOS, a modal window on Linux (which
// blocks every window of the app) and Windows. Make it with NewDialog and
// show it with Run; embed *Dialog in a type of your own for a dialog that
// returns a value, as gota's dialogs do. Escape closes it as its close
// button does.
type Dialog struct {
	*Form
	title, subtitle *LabelWidget
	onBeforeRun     func()
	ran             bool // a headless Run returned
}

// NewDialog makes a dialog over parent (nil for none) titled title, with
// subtitle under it ("" for none), showing content in a window width by
// height DIPs, title bar included. A macOS sheet has none: it shows that
// much shorter, and the title is drawn in the dialog. To open a dialog
// from a dialog, pass the outer one's Form as parent.
func NewDialog(parent *Form, title, subtitle string, width, height int, content Control) *Dialog {
	d := &Dialog{
		Form:     NewForm(title, width, height),
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
	d.Form.SetTitle(s)
	return d
}

// SetSubtitle changes the line under the title; "" hides it.
func (d *Dialog) SetSubtitle(s string) *Dialog {
	d.subtitle.SetText(s).SetVisible(s != "")
	return d
}

// OnBeforeRun runs fn in Run, before the dialog shows.
func (d *Dialog) OnBeforeRun(fn func()) *Dialog { d.change(func() { d.onBeforeRun = fn }); return d }

// SetDeletable lets the user close the dialog with the window's close
// button or keys, or not. Close and the parent closing close it anyway.
func (d *Dialog) SetDeletable(on bool) *Dialog { d.change(func() { d.deletable = on }); return d }

// SetResizable lets the user resize the dialog, or not.
func (d *Dialog) SetResizable(on bool) *Dialog { d.Form.SetResizable(on); return d }

// Run shows the dialog and returns when it closes: by Close, by the user,
// or with its parent. In a handler it runs the other handlers meanwhile,
// as gtk_dialog_run runs the main loop, so the dialog's buttons work; on
// another goroutine it waits; on the main thread it panics.
func (d *Dialog) Run() {
	notMain("Dialog.Run")
	mu.Lock()
	if d.headless != nil && d.ran { // a DialogTester's: a Run after a Run is a new show
		d.headless, d.closing, d.ran = make(chan struct{}), false, false
	}
	before := d.onBeforeRun
	mu.Unlock()
	if before != nil {
		before()
	}
	mu.Lock()
	done := d.headless
	headless := done != nil
	mu.Unlock()
	if done == nil { // a DialogTester has none, and shows no window
		d.Show()
		mu.Lock()
		done = d.done
		mu.Unlock()
	}
	block("Dialog.Run", done)
	if headless {
		mu.Lock()
		d.ran = true
		mu.Unlock()
	}
}
