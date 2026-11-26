package widget

import "github.com/egoist/mygo/ui"

// ProgressPanelWidget shows that something is busy: a spinner beside a
// caption, in bold, and a description under it, as DevExpress's
// ProgressPanel. Assistive technology reads it as a status.
type ProgressPanelWidget struct {
	common[ProgressPanelWidget]
	caption, description string
}

// ProgressPanel makes a progress panel showing caption.
func ProgressPanel(caption string) *ProgressPanelWidget {
	p := &ProgressPanelWidget{caption: caption}
	p.init(p)
	return p
}

// SetCaption sets the bold text beside the spinner.
func (p *ProgressPanelWidget) SetCaption(s string) *ProgressPanelWidget {
	p.change(func() { p.caption = s })
	return p
}

// SetDescription sets the text under the caption; "" shows none.
func (p *ProgressPanelWidget) SetDescription(s string) *ProgressPanelWidget {
	p.change(func() { p.description = s })
	return p
}

func (p *ProgressPanelWidget) build(c *ui.Context) ui.Element {
	return progressPanel(c, p.caption, p.description)
}

// progressPanel builds a progress panel's content: what a ProgressPanel and
// a form's wait show. Under mu, building.
func progressPanel(c *ui.Context, caption, description string) ui.Element {
	t := c.Theme()
	// A status: screen readers read its texts out as they change.
	return ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Role(ui.RoleStatus).
		Children(func() {
			ui.Spinner(c)
			ui.Column(c).Gap(t.Space(0.5)).Children(func() {
				ui.Text(c, caption).Bold()
				if description != "" {
					ui.Text(c, description).TextColor(t.TextMuted)
				}
			})
		})
}

// waitHook builds the wait of the form w over its content, if it waits. The
// first ShowWait sets it, under mu, so that an app showing none links none of
// it.
var waitHook func(c *ui.Context, w *Form)

// ShowWait shows a progress panel of caption and description centred over
// the form's content, which it dims, as DevExpress's
// SplashScreenManager.ShowWaitForm: clicks and keys do not reach the content
// until CloseWait. Called again, it changes the panel's text. It is safe
// from any goroutine; a form not shown yet waits once it shows.
func (w *Form) ShowWait(caption, description string) {
	w.change(func() {
		waitHook = buildWait
		w.wait = &waitText{caption, description}
	})
}

// CloseWait takes away what ShowWait shows; without one it does nothing. It
// is safe from any goroutine.
func (w *Form) CloseWait() {
	w.change(func() { w.wait = nil })
}

// waitText is what a form's wait shows.
type waitText struct{ caption, description string }

// buildWait is waitHook: a full-window backdrop, modal so that the content
// behind it takes no input, centring the panel. A measure leaves it to the
// frame. Under mu.
func buildWait(c *ui.Context, w *Form) {
	if w.wait == nil || c.Measuring() {
		return
	}
	t, wt := c.Theme(), w.wait
	ui.Overlay(c, func() {
		ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().Modal().
			Background(ui.RGBA(0, 0, 0, 0.3)).Children(func() {
			ui.Box(c).Padding(t.Space(5)).Radius(t.Radius+2).Background(t.Background).Border(1, t.Border).
				Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3)).Children(func() {
				progressPanel(c, wt.caption, wt.description)
			})
		})
	})
}
