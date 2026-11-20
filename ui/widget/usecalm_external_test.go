package widget_test

import (
	"reflect"

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
	widget.UseSkinNamed = func(name string, compact bool) {
		s := map[string]*skin.Skin{"Calm": skin.Calm, "Classic": skin.Classic, "HighContrast": skin.HighContrast}[name]
		if compact {
			s = s.WithDensity(skin.Compact)
		}
		skin.Use(s)
	}
	widget.UseLooks = func(looks map[string]string) {
		var l skin.Looks
		v := reflect.ValueOf(&l).Elem()
		for k, cl := range looks {
			v.FieldByName(k).SetString(cl)
		}
		skin.Use(skin.Calm.WithLooks(l))
	}
}
