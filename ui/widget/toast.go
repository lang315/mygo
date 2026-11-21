package widget

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
)

// Toast is a short message a Form shows near the bottom of its window for a
// few seconds (ShowToast), as the outcome of what the user just did.
type Toast struct {
	// ID names the toast: showing one with the ID of a toast showing
	// changes that toast. ShowToast makes one up when it is empty.
	ID string
	// Status colors the toast and picks its icon; StatusNone is neutral.
	// StatusWarning and StatusDanger are announced as alerts.
	Status Status
	// Title, in semibold, and Message, which wraps.
	Title, Message string
	// Timeout: 4 seconds when zero (8 with an Action), until closed when
	// negative. The time stops while the pointer rests on the toasts.
	Timeout time.Duration
	// Action labels a button that runs OnAction and closes the toast.
	Action   string
	OnAction func()
	// OnClose runs when the toast closes, for any reason.
	OnClose func()
}

// toastOp is a ShowToast of t, or a CloseToast of t.ID when close, that the
// form's next frame hands to ui.
type toastOp struct {
	t     Toast
	close bool
}

// toastHook builds the toasts of the form w in the view: it hands ui the
// queue, and gives the toasts the look of the package. The first ShowToast
// sets it, under mu, so that an app showing none links none of it.
var toastHook func(c *ui.Context, w *Form)

// lastToast numbers the toasts of made-up IDs.
var lastToast atomic.Uint64

// ShowToast shows t in the form's window, the newest of three at most, and
// returns its ID. It is safe from any goroutine; a form not shown yet shows
// it once it shows.
func (w *Form) ShowToast(t Toast) string {
	if t.ID == "" {
		t.ID = "widget-toast-" + strconv.FormatUint(lastToast.Add(1), 10)
	}
	w.change(func() {
		toastHook = buildToasts
		w.toasts = queueToast(w.toasts, t)
	})
	return t.ID
}

// queueToast queues t, in the place of a ShowToast of its ID queued since
// the last CloseToast that closes it. Under mu.
func queueToast(ops []toastOp, t Toast) []toastOp {
	for i := len(ops) - 1; i >= 0; i-- {
		if op := &ops[i]; op.close && (op.t.ID == "" || op.t.ID == t.ID) {
			break
		} else if !op.close && op.t.ID == t.ID {
			op.t = t
			return ops
		}
	}
	return append(ops, toastOp{t: t})
}

// CloseToast closes the toast of that ID; "" closes them all.
func (w *Form) CloseToast(id string) {
	w.change(func() {
		if toastHook != nil { // else no ShowToast ran: there is none to close
			w.toasts = append(w.toasts, toastOp{t: Toast{ID: id}, close: true})
		}
	})
}

// postedToast returns fn run on the event goroutine, nil for nil: ui calls
// a toast's callbacks while the view builds, under mu.
func postedToast(fn func()) func() {
	if fn == nil {
		return nil
	}
	return func() { Post(fn) }
}

// buildToasts is toastHook: it hands ui the toasts queued since the last
// frame, then builds those showing. A measure leaves them to the frame.
// Under mu.
func buildToasts(c *ui.Context, w *Form) {
	if c.Measuring() {
		return
	}
	for _, op := range w.toasts {
		if op.close {
			c.CloseToast(op.t.ID)
			continue
		}
		t := op.t
		timeout := t.Timeout
		if timeout == 0 && t.Action != "" {
			timeout = 8 * time.Second
		}
		c.AddToast(ui.Toast{ID: t.ID, Title: t.Title, Description: t.Message, Type: t.Status.String(),
			Timeout: timeout, Action: t.Action, OnAction: postedToast(t.OnAction), OnClose: postedToast(t.OnClose), Data: t.Status})
	}
	clear(w.toasts)
	w.toasts = w.toasts[:0]
	th := c.Theme()
	width, _ := c.Size()
	width = min(400, width-2*th.Space(4))
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Bottom(th.Space(6)).AlignItems(ui.Center).Gap(th.Space(2))
		for _, t := range toasts {
			buildToast(c, t, width)
		}
	})
}

// buildToast builds t, at most width wide: the icon of its status, its
// title and message, its action and a close button, on the status's fill
// over Surface, opaque. A toast of ui code, of no Status, is neutral.
func buildToast(c *ui.Context, t ui.Toast, width float32) {
	th := c.Theme()
	scale := c.Preferences().TextScale
	s, _ := t.Data.(Status)
	fill, text, border, base := statusColors(c, s)
	p := ui.ToastBase(c, t)
	card := p.Root.Row().AlignItems(ui.Start).Gap(th.Space(3)).Padding(th.Space(3)).MaxWidth(width).Radius(th.Radius).
		Background(fill.Over(th.Surface)).TextColor(text).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.25))
	if s == StatusWarning || s == StatusDanger {
		card.Role(ui.RoleAlert)
	}
	if border.A != 0 {
		card.Border(1, border)
	}
	icon := statusIcon(s, base, glyphOn(c, s, base))
	card.Children(func() {
		side := 20 * scale
		ui.Image(c, icon).Size(side, side).Shrink(0)
		ui.Column(c).Grow(1).Shrink(1).Gap(th.Space(0.5)).Children(func() {
			if t.Title != "" {
				ui.Text(c, t.Title).FontWeight(600)
			}
			if t.Description != "" {
				ui.Text(c, t.Description)
			}
		})
		if t.Action != "" {
			b := p.ActionButton().Shrink(0).Padding(th.Space(1), th.Space(2.5)).Radius(th.Radius).Border(1, text.Alpha(0.4)).FontWeight(600)
			if b.Hovered() {
				b.Background(text.Alpha(0.12))
			}
			b.Children(func() { ui.Text(c, t.Action) })
		}
		b := p.CloseButton().Shrink(0).Label(l10n.Close()).Padding(th.Space(1)).Radius(th.Radius)
		if b.Hovered() {
			b.Background(text.Alpha(0.12))
		}
		b.Children(func() { ui.Icon(c, closeIcon).FontSize(16 * scale) })
	})
}
