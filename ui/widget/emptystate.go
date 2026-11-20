package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// EmptyStateWidget is what a view shows when it has nothing in it: an
// optional icon, a title, a message and an optional action, centered both
// ways in its box.
type EmptyStateWidget struct {
	common[EmptyStateWidget]
	title, message string
	icon           *ui.SVG
	action         *ButtonWidget // nil for none
}

// EmptyState returns an empty state of a title and a message.
//
//	widget.EmptyState("Chưa có tài khoản", "Thêm một tài khoản để bắt đầu.").
//		SetAction("Thêm", add)
func EmptyState(title, message string) *EmptyStateWidget {
	e := &EmptyStateWidget{title: title, message: message}
	e.init(e)
	e.lookAs(skinhook.KindEmptyState)
	return e
}

// SetTitle sets the title.
func (e *EmptyStateWidget) SetTitle(s string) *EmptyStateWidget {
	e.change(func() { e.title = s })
	return e
}

// SetMessage sets the message, in muted text that wraps at 360 DIPs.
func (e *EmptyStateWidget) SetMessage(s string) *EmptyStateWidget {
	e.change(func() { e.message = s })
	return e
}

// SetIcon shows svg above the title, 48 DIPs high in muted text; nil for
// none, which keeps no room for it.
func (e *EmptyStateWidget) SetIcon(svg *ui.SVG) *EmptyStateWidget {
	e.change(func() { e.icon = svg })
	return e
}

// SetAction shows a Suggested button of text under the message, which runs
// do on the event goroutine when clicked; "" removes it.
func (e *EmptyStateWidget) SetAction(text string, do func()) *EmptyStateWidget {
	var b *ButtonWidget
	if text != "" {
		b = Button(text).Suggested().OnClick(do) // outside mu: a button's setters take it
	}
	e.change(func() { e.action = b })
	return e
}

func (e *EmptyStateWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	scale := c.Preferences().TextScale
	muted := t.TextMuted
	box := ui.Column(c).Grow(1).Center().Gap(t.Space(2)).Padding(t.Space(6))
	return box.Children(func() {
		if e.icon != nil {
			ui.Icon(c, e.icon).FontSize(48 * scale).TextColor(muted)
		}
		if e.title != "" {
			title := ui.Text(c, e.title).TextAlign(ui.Center)
			titleText.apply(c, title)
		}
		if e.message != "" {
			ui.Text(c, e.message).TextColor(muted).MaxWidth(360 * scale).TextAlign(ui.Center)
		}
		if e.action != nil {
			outer := axis
			axis = vertical
			render(c, e.action)
			axis = outer
		}
	})
}
