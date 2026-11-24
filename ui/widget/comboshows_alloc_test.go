package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// comboIdleAllocs returns the allocations of an idle frame of a form holding
// two labels and a combo made by combo: the fewest of three measures, as
// another goroutine's (the race detector's, a handler's) only add to them.
func comboIdleAllocs(combo func(a, b *LabelWidget) *ComboBoxWidget) float64 {
	a, b := Label("A"), Label("B")
	tt := ui.NewTester(content(VBox(0).Add(a, b, combo(a, b))), 800, 600)
	for range 10 {
		tt.Frame()
	}
	n := testing.AllocsPerRun(20, tt.Frame)
	for range 2 {
		n = min(n, testing.AllocsPerRun(20, tt.Frame))
	}
	return n
}

func proxyCombo() *ComboBoxWidget {
	return ComboBox(Option{"none", "None"}, Option{"http", "HTTP"}, Option{"socks", "SOCKS5"}).SetValue("http")
}

// Case 11: a combo with no Shows call costs an idle frame what it did
// before Shows was added (logged, compared by hand with main's), and no
// more than the same combo with Shows calls.
func TestComboShowsIdleFrameAllocs(t *testing.T) {
	plain := comboIdleAllocs(func(a, b *LabelWidget) *ComboBoxWidget { return proxyCombo() })
	t.Logf("AllocsPerRun: %.1f an idle frame with a combo with no Shows", plain)
	shows := comboIdleAllocs(func(a, b *LabelWidget) *ComboBoxWidget {
		return proxyCombo().Shows("http", a, b).Shows("socks", b)
	})
	t.Logf("AllocsPerRun: %.1f with Shows", shows)
	if plain > shows {
		t.Errorf("AllocsPerRun: %.1f an idle frame with a combo with no Shows, %.1f with", plain, shows)
	}
}
