package widget

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// skinOf returns the active skin's tokens for c's appearance, and the text
// scale the strips' sizes take from them (ui scales Theme.Metrics itself:
// never scale those twice); nil when no skin is on, and the caller keeps
// its plain look.
func skinOf(c *ui.Context) (*skinhook.Tokens, float32) {
	tk := skinhook.Get(c.Theme().Dark)
	if tk == nil {
		return nil, 1
	}
	scale := c.Preferences().TextScale
	if scale <= 0 {
		scale = 1
	}
	return tk, scale
}

// skinChecked gives a toggle that is on a skin's look: the accent's border
// color as its fill, ringed in the accent, which stands out from one that is
// off as the plain pressed face does not. The control's own style wins.
func skinChecked(c *ui.Context, e ui.Element, on bool) {
	if tk, _ := skinOf(c); tk != nil && on {
		e.Background(tk.Accent.Border).Border(1, tk.Accent.Base)
	}
}
