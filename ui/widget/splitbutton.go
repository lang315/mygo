package widget

import "github.com/egoist/mygo/ui"

// SplitButtonWidget is a button joined to an arrow, as Save beside Save
// As…: a click on the button runs OnClick, and the arrow opens a menu of
// related actions. It is ui.SplitButton underneath, as DevExpress's
// DropDownButton with an arrow of its own.
type SplitButtonWidget struct {
	common[SplitButtonWidget]
	text      string
	icon      *ui.SVG
	items     []MenuItem
	suggested bool
	onClick   func()
	clicked   func()           // onClicked, made once
	menu      func(m *ui.Menu) // buildItems, made once
}

// SplitButton makes a split button showing text, whose arrow opens items,
// which take the same MenuItem as DropDownButton's.
func SplitButton(text string, items ...MenuItem) *SplitButtonWidget {
	b := &SplitButtonWidget{text: text, items: items}
	b.init(b)
	return b
}

// OnClick runs fn, on the event goroutine, when the button is clicked, not
// the arrow.
func (b *SplitButtonWidget) OnClick(fn func()) *SplitButtonWidget {
	b.change(func() { b.onClick = fn })
	return b
}

// Suggested draws the split button as the main action, in the accent
// color, as ui.PrimarySplitButton.
func (b *SplitButtonWidget) Suggested() *SplitButtonWidget {
	b.change(func() { b.suggested = true })
	return b
}

// SetIcon shows an icon before the text, in the color of the text, as
// ButtonWidget's; nil removes it.
func (b *SplitButtonWidget) SetIcon(s *ui.SVG) *SplitButtonWidget {
	b.change(func() { b.icon = s })
	return b
}

func (b *SplitButtonWidget) build(c *ui.Context) ui.Element {
	text := b.text
	if b.icon != nil {
		text = ""
	}
	var p ui.SplitButtonParts
	if b.suggested {
		p = ui.PrimarySplitButton(c, text)
	} else {
		p = ui.SplitButton(c, text)
	}
	if b.icon != nil {
		iconFace(c, p.Button, b.icon, nil, b.text, b.tooltip, false, 0)
	}
	if b.menu == nil {
		b.menu, b.clicked = b.buildItems, b.onClicked // once: a closure made in each build is an allocation each
	}
	p.Arrow.Menu(b.menu)
	p.Button.OnClick(b.clicked)
	return p.Group
}

// buildItems is the arrow's menu. ui calls it while the view builds, under
// mu.
func (b *SplitButtonWidget) buildItems(m *ui.Menu) { buildMenu(m, b.items) }

// onClicked is ui's OnClick callback: it runs the OnClick of the moment.
func (b *SplitButtonWidget) onClicked() {
	mu.Lock()
	h := b.onClick
	mu.Unlock()
	if h != nil {
		Post(h)
	}
}
