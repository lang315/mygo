package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// PopupEditWidget is a field showing a text, whose drop-down holds any
// controls, as DevExpress's PopupContainerEdit: a click on it, Enter, Space
// or Alt+Down opens the drop-down; Escape, a click outside it or on the field
// closes it. The app reads the controls in OnClosed and sets the field's
// text. It is ui.Popover underneath, below a field with a ComboBox's look.
type PopupEditWidget struct {
	common[PopupEditWidget]
	text      string
	controls  []Control
	open      bool // ui's Popover writes it while the view builds, under mu
	seenHides int  // the hides the last build saw: a hide closes the drop-down
	onClosed  func()
}

// PopupEdit makes a closed PopupEdit whose drop-down holds controls, in a
// column.
func PopupEdit(controls ...Control) *PopupEditWidget {
	p := &PopupEditWidget{controls: controls}
	p.init(p)
	p.lookAs(skinhook.KindComboBox)
	for _, w := range controls {
		adopt(&p.core, w)
	}
	return p
}

// SetText sets the text the field shows.
func (p *PopupEditWidget) SetText(s string) *PopupEditWidget {
	p.change(func() { p.text = s })
	return p
}

// Text returns the text the field shows.
func (p *PopupEditWidget) Text() string { mu.Lock(); defer mu.Unlock(); return p.text }

// ShowPopup opens the drop-down, ClosePopup closes it. Neither runs
// OnClosed, as a setter runs no handler.
func (p *PopupEditWidget) ShowPopup() *PopupEditWidget {
	p.change(func() { p.open = true })
	return p
}

func (p *PopupEditWidget) ClosePopup() *PopupEditWidget {
	p.change(func() { p.open = false })
	return p
}

// IsOpen reports whether the drop-down shows.
func (p *PopupEditWidget) IsOpen() bool { mu.Lock(); defer mu.Unlock(); return p.open }

// OnClosed runs fn, on the event goroutine, after the user closed the
// drop-down: with Escape, a click outside it or a click on the field.
func (p *PopupEditWidget) OnClosed(fn func()) *PopupEditWidget {
	p.change(func() { p.onClosed = fn })
	return p
}

func (p *PopupEditWidget) build(c *ui.Context) ui.Element {
	if p.seenHides != p.hides { // out of sight since the last build: it closes
		p.open, p.seenHides = false, p.hides
	}
	was := p.open
	field := ui.ButtonBase(c).Role(ui.RolePopUpButton).Value(p.text)
	comboTrigger(c, field, p.text)
	if field.Clicked() {
		p.open = !p.open // a press on the field is not outside the drop-down: it toggles
	} else if field.Shortcut(ui.Alt, ui.KeyDown) {
		p.open = true
	}
	field.Expanded(p.open)
	if p.open {
		ui.PopoverBase(c, field, &p.open, func(panel ui.Element) {
			comboPanel(c, panel)
			panel.Padding(c.Theme().Space(2)).Gap(c.Theme().Space(2))
			outer := axis
			axis = vertical
			for _, w := range p.controls {
				render(c, w)
			}
			axis = outer
		})
	}
	if was && !p.open && p.onClosed != nil {
		Post(p.onClosed)
	}
	return field
}

func (p *PopupEditWidget) children() []Control { return p.controls }
