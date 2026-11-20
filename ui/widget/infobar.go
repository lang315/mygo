package widget

import (
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// InfoBarWidget is a message inline in a page, as WinUI's InfoBar: an icon
// of its status, a title and a message that wraps, and at the end an action
// and a close button, each if set. It spans its box. Assistive technology
// announces it as it shows: an alert for StatusWarning and StatusDanger, a
// status for the rest.
type InfoBarWidget struct {
	common[InfoBarWidget]
	title, message string
	action         *ButtonWidget // nil for none
	close          *ButtonWidget // shown while closable
	closable       bool
	onClose        func()
}

// InfoBar returns an info bar of status s, with a title and a message.
//
//	widget.InfoBar(widget.StatusWarning, "Cần xác minh", "Mở email để xác minh.").
//		SetAction("Xác minh", verify)
func InfoBar(s Status, title, message string) *InfoBarWidget {
	i := &InfoBarWidget{title: title, message: message}
	i.init(i)
	i.status = s
	i.lookAs(skinhook.KindInfoBar)
	i.close = Button("").SetIcon(closeIcon).OnClick(i.closed)
	return i
}

// closeIcon is an info bar's close button's: Lucide's x.
var closeIcon = lucide(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)

// SetStatus sets the status, which colors the bar and picks its icon.
func (i *InfoBarWidget) SetStatus(s Status) *InfoBarWidget {
	i.change(func() { i.status = s })
	return i
}

// Status returns the status.
func (i *InfoBarWidget) Status() Status { mu.Lock(); defer mu.Unlock(); return i.status }

// SetTitle sets the title, in semibold before the message.
func (i *InfoBarWidget) SetTitle(s string) *InfoBarWidget { i.change(func() { i.title = s }); return i }

// Title returns the title.
func (i *InfoBarWidget) Title() string { mu.Lock(); defer mu.Unlock(); return i.title }

// SetMessage sets the message, which wraps at the bar's width.
func (i *InfoBarWidget) SetMessage(s string) *InfoBarWidget {
	i.change(func() { i.message = s })
	return i
}

// Message returns the message.
func (i *InfoBarWidget) Message() string { mu.Lock(); defer mu.Unlock(); return i.message }

// SetAction shows a button of text at the end, which runs do on the event
// goroutine when clicked; "" removes it.
func (i *InfoBarWidget) SetAction(text string, do func()) *InfoBarWidget {
	var b *ButtonWidget
	if text != "" {
		b = Button(text).OnClick(do) // outside mu: a button's setters take it
	}
	i.change(func() { i.action = b })
	return i
}

// SetClosable shows a close button at the end, or takes it away. Clicking
// it hides the bar, then runs OnClose's function.
func (i *InfoBarWidget) SetClosable(on bool) *InfoBarWidget {
	i.change(func() { i.closable = on })
	return i
}

// OnClose runs f on the event goroutine after the close button hid the bar.
// Hiding it with SetVisible(false) does not run it.
func (i *InfoBarWidget) OnClose(f func()) *InfoBarWidget {
	i.change(func() { i.onClose = f })
	return i
}

// closed is the close button's OnClick, on the event goroutine.
func (i *InfoBarWidget) closed() {
	i.SetVisible(false)
	mu.Lock()
	f := i.onClose
	mu.Unlock()
	if f != nil {
		f()
	}
}

// iconOf returns the icon of the status in base, its glyph in glyph, parsed
// once for each (kindSVG). Under mu.
func (i *InfoBarWidget) iconOf(base, glyph ui.Color) *ui.SVG {
	k := Info
	switch i.status {
	case StatusSuccess:
		k = kindSuccess
	case StatusWarning:
		k = Warning
	case StatusDanger:
		k = Error
	}
	return kindSVG(k, base, glyph)
}

// glyphOn is the color of an icon's glyph on base: the family's On under a
// skin, else white, or black where white is under 3:1 (an amber).
func glyphOn(c *ui.Context, s Status, base ui.Color) ui.Color {
	if tk, _ := skinOf(c); tk != nil {
		switch s {
		case StatusInfo, StatusRunning:
			return tk.Info.On
		case StatusSuccess:
			return tk.Success.On
		case StatusWarning:
			return tk.Warning.On
		case StatusDanger:
			return tk.Danger.On
		}
	}
	return plainGlyphOn(base)
}

// plainGlyphOn is the glyph's color on base without a skin: white, or black
// where white is under 3:1.
func plainGlyphOn(base ui.Color) ui.Color {
	if contrastOf(ui.RGB(255, 255, 255), base) < 3 {
		return ui.RGB(0, 0, 0)
	}
	return ui.RGB(255, 255, 255)
}

func (i *InfoBarWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	scale := c.Preferences().TextScale
	fill, text, border, base := statusColors(c, i.status)
	role := ui.RoleStatus
	if i.status == StatusWarning || i.status == StatusDanger {
		role = ui.RoleAlert
	}
	bar := ui.Row(c).AlignItems(ui.Start).Gap(t.Space(3)).Padding(t.Space(3)).Background(fill).
		TextColor(text).Radius(t.Radius).Role(role) // a skin's theme has its RadiusControl
	if border.A != 0 {
		bar.Border(1, border)
	}
	icon := i.iconOf(base, glyphOn(c, i.status, base))
	return bar.Children(func() {
		side := 20 * scale
		ui.Image(c, icon).Size(side, side).Shrink(0)
		ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
			if i.title != "" {
				ui.Text(c, i.title).FontWeight(600)
			}
			if i.message != "" {
				ui.Text(c, i.message)
			}
		})
		outer := axis
		axis = horizontal
		if i.action != nil {
			render(c, i.action)
		}
		if i.closable {
			i.close.tooltip = l10n.Close() // names the icon-only button, in the language of the moment
			render(c, i.close)
		}
		axis = outer
	})
}
