package widget

import (
	"sync"

	"github.com/egoist/mygo"
)

// NotifyIcon is an icon in the menu bar (macOS) or the notification area
// (Windows, and Linux panels that have one), with a tooltip and a context
// menu. Make it with NewNotifyIcon, on any goroutine once the app is ready
// (a handler, or after mygo.App.WhenReady); every method is safe from any
// goroutine, and the handlers run on the event goroutine.
//
// Platform gaps are mygo's, not hidden here:
//   - OnClick runs when the icon is clicked and it has no context menu; with
//     a menu, a click opens it. OnRightClick runs on a right click.
//   - There is no OnDoubleClick: mygo's tray reports no double click on any
//     platform (macOS has none, and the Windows and Linux backends do not
//     tell it), and a double click made of two clicks would be a guess.
//   - Linux panels place icons themselves, and the tooltip and the icon
//     depend on the panel.
type NotifyIcon struct {
	omu sync.Mutex // serializes what calls mygo's tray; taken before smu

	smu      sync.Mutex // guards the state below, never held across a call into mygo
	icon     []byte
	template bool
	text     string
	menu     []MenuItem
	visible  bool
	disposed bool
	onClick  func()
	onRight  func()
	tray     *mygo.Tray
}

// NewNotifyIcon makes an icon, visible, with no image: give it one with
// SetIcon. One call makes one icon.
func NewNotifyIcon() (*NotifyIcon, error) {
	n := &NotifyIcon{visible: true}
	n.omu.Lock()
	defer n.omu.Unlock()
	if err := n.show(); err != nil {
		return nil, err
	}
	return n, nil
}

// show makes the tray of the icon from its state. Under omu.
func (n *NotifyIcon) show() error {
	n.smu.Lock()
	opts := mygo.TrayOptions{Icon: n.icon, IconIsTemplate: n.template, ToolTip: n.text}
	if len(n.menu) > 0 {
		opts.Menu = mygo.NewMenu(nativeMenu(n.menu))
	}
	n.smu.Unlock()
	t, err := mygo.NewTray(opts)
	if err != nil {
		return err
	}
	t.OnClick(func() { n.fire(t, func() func() { return n.onClick }) })
	t.OnRightClick(func() { n.fire(t, func() func() { return n.onRight }) })
	n.smu.Lock()
	n.tray = t
	n.smu.Unlock()
	return nil
}

// fire posts the handler pick returns, if there is one, for a click of tray
// t: not one of a tray that was hidden or disposed since the click was
// queued. mygo calls it on the main thread: it takes only smu, which no one
// holds while waiting for it.
func (n *NotifyIcon) fire(t *mygo.Tray, pick func() func()) {
	n.smu.Lock()
	var h func()
	if !n.disposed && n.tray == t {
		h = pick()
	}
	n.smu.Unlock()
	if h != nil {
		Post(h)
	}
}

// current returns the tray shown, nil when hidden or disposed.
func (n *NotifyIcon) current() *mygo.Tray {
	n.smu.Lock()
	defer n.smu.Unlock()
	return n.tray
}

// SetIcon sets the icon's image, a PNG, ideally 16 by 16 points (32 by 32
// pixels for sharp screens). A template image is black and transparent, and
// macOS tints it to match the menu bar. It returns the error of a PNG the
// system cannot show.
func (n *NotifyIcon) SetIcon(png []byte, template bool) error {
	n.omu.Lock()
	defer n.omu.Unlock()
	n.smu.Lock()
	n.icon, n.template = png, template
	n.smu.Unlock()
	if t := n.current(); t != nil {
		return t.SetIcon(png, template)
	}
	return nil
}

// SetText sets the tooltip shown when the pointer rests on the icon.
func (n *NotifyIcon) SetText(s string) *NotifyIcon {
	n.omu.Lock()
	defer n.omu.Unlock()
	n.smu.Lock()
	n.text = s
	n.smu.Unlock()
	if t := n.current(); t != nil {
		t.SetToolTip(s)
	}
	return n
}

// SetContextMenu sets the menu the icon opens; with no items there is none,
// and a click reaches OnClick. The items' OnClick run on the event goroutine.
func (n *NotifyIcon) SetContextMenu(items ...MenuItem) *NotifyIcon {
	n.omu.Lock()
	defer n.omu.Unlock()
	n.smu.Lock()
	n.menu = items
	n.smu.Unlock()
	if t := n.current(); t != nil {
		if len(items) == 0 {
			t.SetMenu(nil)
		} else {
			t.SetMenu(mygo.NewMenu(nativeMenu(items)))
		}
	}
	return n
}

// OnClick runs fn on the event goroutine when the user clicks the icon and
// it has no context menu.
func (n *NotifyIcon) OnClick(fn func()) *NotifyIcon {
	n.smu.Lock()
	n.onClick = fn
	n.smu.Unlock()
	return n
}

// OnRightClick runs fn on the event goroutine when the user right-clicks
// the icon.
func (n *NotifyIcon) OnRightClick(fn func()) *NotifyIcon {
	n.smu.Lock()
	n.onRight = fn
	n.smu.Unlock()
	return n
}

// Visible reports whether the icon is shown.
func (n *NotifyIcon) Visible() bool {
	n.smu.Lock()
	defer n.smu.Unlock()
	return n.visible && !n.disposed
}

// SetVisible shows or hides the icon. A hidden icon is removed from the bar,
// and shown again with its image, tooltip, menu and handlers. A disposed
// icon stays gone. Showing it returns the error if the system cannot make
// the icon, as of an image it cannot decode that was set while it was
// hidden; the icon is then not visible.
func (n *NotifyIcon) SetVisible(v bool) error {
	n.omu.Lock()
	defer n.omu.Unlock()
	n.smu.Lock()
	if n.disposed || n.visible == v {
		n.smu.Unlock()
		return nil
	}
	n.visible = v
	t := n.tray
	if !v {
		n.tray = nil
	}
	n.smu.Unlock()
	if !v {
		t.Destroy()
		return nil
	}
	if err := n.show(); err != nil {
		n.smu.Lock()
		n.visible = false // no icon could be made: say so
		n.smu.Unlock()
		return err
	}
	return nil
}

// Dispose removes the icon for good; the icon's methods do nothing after.
func (n *NotifyIcon) Dispose() {
	n.omu.Lock()
	defer n.omu.Unlock()
	n.smu.Lock()
	t := n.tray
	n.tray, n.disposed, n.visible = nil, true, false
	n.smu.Unlock()
	if t != nil {
		t.Destroy()
	}
}
