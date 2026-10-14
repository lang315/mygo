package widget

import "github.com/egoist/mygo/ui"

// LabelWidget shows a line of text.
type LabelWidget struct {
	common[LabelWidget]
	text string
}

// Label makes a label showing s.
func Label(s string) *LabelWidget {
	l := &LabelWidget{text: s}
	l.init(l)
	return l
}

// SetText changes the text.
func (l *LabelWidget) SetText(s string) *LabelWidget { change(func() { l.text = s }); return l }

// Text returns the text.
func (l *LabelWidget) Text() string { mu.Lock(); defer mu.Unlock(); return l.text }

func (l *LabelWidget) build(c *ui.Context) ui.Element { return ui.Text(c, l.text) }
