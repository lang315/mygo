package widget

import "github.com/egoist/mygo/ui"

// style is how a widget looks, set by the style methods.
type style struct{}

// apply styles e, under mu while building.
func (s *style) apply(c *ui.Context, e ui.Element) {}
