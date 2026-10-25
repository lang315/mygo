package widget

import "github.com/egoist/mygo/ui"

// StatusStripWidget is a bar of status items, for the bottom of a form
// (Form.StatusStrip): StatusLabels, Springs, StatusProgress bars and
// Separators, in a row.
type StatusStripWidget struct {
	common[StatusStripWidget]
	items []Control
}

// StatusStrip makes a status bar of items.
func StatusStrip(items ...Control) *StatusStripWidget {
	s := &StatusStripWidget{items: items}
	s.init(s)
	return s
}

// Add adds items after the others.
func (s *StatusStripWidget) Add(items ...Control) *StatusStripWidget {
	s.change(func() { s.items = append(s.items, items...) })
	return s
}

func (s *StatusStripWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	return ui.Row(c).AlignItems(ui.Center).Gap(t.Space(3)).Padding(t.Space(1), t.Space(3)).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		for _, it := range s.items {
			render(c, it)
		}
	})
}

// StatusLabelWidget is text with an optional icon before it, for a status
// strip. Its setters change it while the app runs.
type StatusLabelWidget struct {
	common[StatusLabelWidget]
	text string
	icon *ui.SVG
}

// StatusLabel makes a status item showing text.
func StatusLabel(text string) *StatusLabelWidget {
	l := &StatusLabelWidget{text: text}
	l.init(l)
	return l
}

// SetText changes the text.
func (l *StatusLabelWidget) SetText(s string) *StatusLabelWidget {
	l.change(func() { l.text = s })
	return l
}

// SetIcon shows an icon before the text, in the color of the text; nil
// removes it.
func (l *StatusLabelWidget) SetIcon(icon *ui.SVG) *StatusLabelWidget {
	l.change(func() { l.icon = icon })
	return l
}

func (l *StatusLabelWidget) build(c *ui.Context) ui.Element {
	return ui.Row(c).AlignItems(ui.Center).Gap(c.Theme().Space(1.5)).Children(func() {
		if l.icon != nil {
			ui.Icon(c, l.icon)
		}
		ui.Text(c, l.text)
	})
}

// StatusProgress makes a small progress bar for a status strip: SetFraction
// sets how far the task is, 0 to 1, and a negative fraction shows a task of
// unknown length.
func StatusProgress() *ProgressBarWidget { return ProgressBar().SetSize(96, 0) }

// SpringWidget takes the free width of its row, pushing the items after it
// to the end.
type SpringWidget struct{ common[SpringWidget] }

// Spring makes an empty item that takes the free room of a status strip or
// a tool strip.
func Spring() *SpringWidget {
	s := &SpringWidget{}
	s.init(s)
	return s
}

func (s *SpringWidget) build(c *ui.Context) ui.Element { return ui.Spacer(c) }

// StatusStrip docks s at the bottom of the form, below its content; nil
// removes it.
func (w *Form) StatusStrip(s *StatusStripWidget) *Form {
	w.change(func() { w.status = s })
	return w
}

func (s *StatusStripWidget) children() []Control { return s.items }
