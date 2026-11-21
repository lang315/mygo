package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// BadgeWidget is a short text in a pill colored by its status, as "Live" or
// "Sai mật khẩu" in a list of accounts. A running badge has a dot before its
// text and may show its current step after the pill.
type BadgeWidget struct {
	common[BadgeWidget]
	text, detail string
	name         string // what assistive technology reads: the text, then the detail when shown
}

// Badge returns a badge of text, of StatusNone.
//
//	widget.Badge("Live").SetStatus(widget.StatusSuccess)
func Badge(text string) *BadgeWidget {
	b := &BadgeWidget{text: text, name: text}
	b.init(b)
	return b
}

// SetText sets the text in the pill.
func (b *BadgeWidget) SetText(s string) *BadgeWidget {
	b.change(func() { b.text = s; b.rename() })
	return b
}

// Text returns the text in the pill.
func (b *BadgeWidget) Text() string { mu.Lock(); defer mu.Unlock(); return b.text }

// SetStatus sets the status, which colors the pill.
func (b *BadgeWidget) SetStatus(s Status) *BadgeWidget {
	b.change(func() { b.status = s; b.rename() })
	return b
}

// Status returns the status.
func (b *BadgeWidget) Status() Status { mu.Lock(); defer mu.Unlock(); return b.status }

// SetDetail sets the current step of a running badge, drawn after the pill
// in muted text and read after the text. It shows only for StatusRunning.
func (b *BadgeWidget) SetDetail(s string) *BadgeWidget {
	b.change(func() { b.detail = s; b.rename() })
	return b
}

// Detail returns the detail SetDetail set.
func (b *BadgeWidget) Detail() string { mu.Lock(); defer mu.Unlock(); return b.detail }

// showsDetail reports whether the detail is drawn and read. Under mu.
func (b *BadgeWidget) showsDetail() bool { return b.status == StatusRunning && b.detail != "" }

// rename joins the accessible name once, in the setters, so that a frame
// builds no string. Under mu.
func (b *BadgeWidget) rename() {
	b.name = b.text
	if b.showsDetail() {
		b.name = b.text + ", " + b.detail
	}
}

func (b *BadgeWidget) build(c *ui.Context) ui.Element {
	detail := ""
	if b.showsDetail() {
		detail = b.detail
	}
	return badgePill(c, b.status, b.text, detail, b.name, b.disabled, b.textKey)
}

// badgePill builds a badge of status s: a pill of text, with a dot before it
// when s is StatusRunning, and detail after it unless it is "", the whole
// read as name. A grid's badge cells build it too. The status roles of the
// skin's Badge look resolve to s. key keys the text color's animation.
// Under mu.
func badgePill(c *ui.Context, s Status, text, detail, name string, disabled bool, key any) ui.Element {
	t := c.Theme()
	scale := c.Preferences().TextScale // ui makes an unset one 1
	fill, fg, _, base := statusColors(c, s)
	h := 20 * scale
	// The badge is read as one text, its name; its pill and detail are not
	// read apart. It never grows: an HExpand widens the row, not the pill.
	row := ui.Row(c).AlignItems(ui.Center).Gap(t.Space(1.5)).Shrink(0).Role(ui.RoleText).Label(name)
	return row.Children(func() {
		pill := ui.Row(c).AlignItems(ui.Center).Height(h).Padding(0, t.Space(2)).Radius(h / 2).
			Background(fill).TextColor(fg).FontSize(12 * scale).FontWeight(600).Shrink(0)
		running := s == StatusRunning
		if running {
			pill.Gap(t.Space(1.5))
		}
		// The skin's Badge look styles the pill, not the row with the detail.
		if look := lookOf(skinhook.KindBadge); look != nil {
			outer := styling
			styling = s // a grid's cell has no control whose build set it
			st := look.effective(t.Dark, pill.Hovered(), false, false, pill.Pressed(), disabled)
			st.apply(c, pill, key)
			styling = outer
		}
		pill.Children(func() {
			if running {
				d := 6 * scale
				ui.Box(c).Width(d).Height(d).Radius(d / 2).Background(base).Shrink(0)
			}
			ui.Text(c, text).SingleLine()
		})
		if detail != "" {
			d := ui.Text(c, detail).SingleLine().FontSize(12 * scale)
			if !c.RowSelected() { // on a selected row, the selection's text reads where TextMuted may not
				d.TextColor(t.TextMuted)
			}
		}
	})
}
