package widget_test

import (
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/skin"
)

func init() {
	widget.UseCalm = func(on bool) {
		if on {
			skin.Use(skin.Calm)
			return
		}
		skin.Use(nil)
	}
}
