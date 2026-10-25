package widget

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Every property a Style can set has a line in the note, as does each side
// of the four-sided ones: a property added without one fails here.
func TestEveryPropertyOfAStyleIsDescribed(t *testing.T) {
	th := ui.LightTheme()
	for bit := prop(1); bit < pEnd; bit <<= 1 {
		p := props{set: bit}
		if bit == pShadows {
			p.shadows = []shadow{{x: 1}}
		}
		if got := p.describe(th, ""); len(got) == 0 {
			t.Errorf("property %#x is not described", uint32(bit))
		}
	}
	for i := range 4 {
		bit := uint8(1) << i
		p := props{padSet: bit, marginSet: bit, radiusSet: bit, borderSet: bit}
		got := p.describe(th, "")
		if len(got) != 4 {
			t.Errorf("side %d: %d lines for padding, margin, radius and border width, want 4: %v", i, len(got), got)
		}
		for _, f := range got {
			if !strings.Contains(f.Name, "-") {
				t.Errorf("side %d: %q does not say which side", i, f.Name)
			}
		}
	}
	all := props{padSet: 0b1111, pad: [4]float32{2, 2, 2, 2}}
	if got := all.describe(th, ""); len(got) != 1 || got[0].Name != "padding" {
		t.Errorf("four equal sides: %v, want one padding line", got)
	}
}
