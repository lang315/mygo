package skinfile

import (
	"testing"

	"github.com/egoist/mygo/ui/widget/internal/skinhook"
	"github.com/egoist/mygo/ui/widget/skin"
)

// A skin file whose base is HighContrast keeps its flat statuses, through its
// accent, density, tokens and looks; one based on Calm has none.
func TestParseKeepsFlatStatus(t *testing.T) {
	t.Cleanup(func() { skin.Use(nil) })
	for base, want := range map[string]bool{"HighContrast": true, "Calm": false} {
		s, err := Parse([]byte(`{"version":1,"name":"X","base":"` + base + `","accent":"#0f766e","density":"compact",
			"light":{"Background":"#fffffe"},"looks":{"Button":"border-2"}}`))
		if err != nil {
			t.Fatal(err)
		}
		skin.Use(s)
		if got := skinhook.FlatStatus(); got != want {
			t.Errorf("base %s: FlatStatus is %v, want %v", base, got, want)
		}
	}
}
