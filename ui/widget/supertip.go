package widget

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// SuperTip is a rich tooltip, as DevExpress's SuperToolTip: a bold title,
// a body with an image at its start, and a footer under a separator. Each
// part may be left empty.
type SuperTip struct {
	Title  string
	Body   string
	Image  *ui.Bitmap
	Footer string
}

// superTipWidth is how wide a SuperTip grows, in DIPs, before its text
// wraps.
const superTipWidth = 320

// superTipState is a control's SuperTip, with what its builds keep. Under
// mu.
type superTipState struct {
	SuperTip
	said string // Title, Body and Footer joined, the control's description
	// c and ctl are the context and the control of the build under way,
	// which draw reads: draw is made once, so a build allocates nothing.
	c    *ui.Context
	ctl  ui.Element
	draw func(tip ui.Element)
}

// SetSuperTip shows t while the pointer rests on the widget, in place of
// SetTooltip's text; a zero SuperTip removes it, and the text comes back.
// Assistive technology reads its title, body and footer as the widget's
// description.
func (c *common[W]) SetSuperTip(t SuperTip) *W { c.change(func() { c.setSuperTip(t) }); return c.self }

// setSuperTip is SetSuperTip, out of the generic methods so that each
// widget type does not carry it. Under mu.
func (b *core) setSuperTip(t SuperTip) {
	if t == (SuperTip{}) {
		b.superTip, b.buildTip = nil, nil
		return
	}
	s := &superTipState{SuperTip: t, said: joinSaid(t.Title, t.Body, t.Footer)}
	s.draw = s.drawTip
	b.superTip, b.buildTip = s, buildSuperTip
}

// buildSuperTip describes ctl, the control of b, by its SuperTip, and shows
// the tip as a text tooltip shows (ui.TooltipBase), unless an error's
// tooltip shows instead. Under mu, building.
func buildSuperTip(c *ui.Context, ctl ui.Element, b *core) {
	s := b.superTip
	if b.desc == "" {
		ctl.Description(s.said)
	}
	if b.errMsg != "" {
		return
	}
	s.c, s.ctl = c, ctl
	ui.TooltipBase(c, ctl, s.draw)
	s.c, s.ctl = nil, ui.Element{}
}

// drawTip styles the tip and builds its parts, as DevExpress draws a
// SuperToolTip, in the look of a text tooltip (Element.Tooltip): the skin's
// tooltip colors, the theme's Inverse.
func (s *superTipState) drawTip(tip ui.Element) {
	c, t := s.c, s.c.Theme()
	fill, text := RoleInverse.resolve(t), RoleInverseText.resolve(t)
	tip.MaxWidth(superTipWidth).Padding(t.Space(2)).Radius(t.Space(1.25)).
		Background(fill).TextColor(text).FontSize(t.FontSize - 1)
	tip.Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.2))
	tip.AttachTo(s.ctl, ui.AnchorTop, ui.AnchorBottom).Margin(0, 0, t.Space(1.5), 0)
	tip.Children(func() {
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			if s.Title != "" {
				ui.Text(c, s.Title).Bold()
			}
			if s.Body != "" || s.Image != nil {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Start).Children(func() {
					if s.Image != nil {
						ui.Image(c, s.Image)
					}
					if s.Body != "" {
						ui.Text(c, s.Body)
					}
				})
			}
			if s.Footer != "" {
				if s.Title != "" || s.Body != "" || s.Image != nil {
					ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(text.Mix(fill, 0.6))
				}
				ui.Text(c, s.Footer)
			}
		})
	})
}

// joinSaid joins the parts that are not empty into one description: a part
// that ends a sentence takes a space after it, others ". ", as a Fields
// row's description joins its tooltip.
func joinSaid(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if s := b.String(); s != "" {
			if last, _ := utf8.DecodeLastRuneInString(s); strings.ContainsRune(".!?…:", last) {
				b.WriteString(" ")
			} else {
				b.WriteString(". ")
			}
		}
		b.WriteString(p)
	}
	return b.String()
}
