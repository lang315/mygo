package mygo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"testing"
)

// The preferences are written by hand, and what is written is what
// encoding/json writes for themePreferencesJSON, mygo-runtime's
// ThemePreferences.
func TestThemePreferencesJSONIsMarshal(t *testing.T) {
	for _, accent := range []color.NRGBA{{}, {R: 1, G: 2, B: 3, A: 255}, {R: 255, G: 0, B: 128, A: 7}, {A: 1}} {
		for _, scale := range []float64{0, 1, 1.25, 1.5, 0.1, 2e-7, 1e21} {
			for bits := range 8 {
				p := ThemePreferences{Accent: accent, ReduceMotion: bits&1 != 0, ReduceTransparency: bits&2 != 0, HighContrast: bits&4 != 0, TextScale: scale}
				v := themePreferencesJSON{ReduceMotion: p.ReduceMotion, ReduceTransparency: p.ReduceTransparency, HighContrast: p.HighContrast, TextScale: p.TextScale}
				if accent.A != 0 {
					s := fmt.Sprintf("#%02x%02x%02x", accent.R, accent.G, accent.B)
					if accent.A != 255 {
						s += fmt.Sprintf("%02x", accent.A)
					}
					v.Accent = &s
				}
				want, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				got, err := p.appendJSON(nil)
				if err != nil || !bytes.Equal(got, want) {
					t.Errorf("%+v: written %s, %v; encoding/json writes %s", p, got, err, want)
				}
				if m, err := p.MarshalJSON(); err != nil || !bytes.Equal(m, want) {
					t.Errorf("%+v: MarshalJSON %s, %v", p, m, err)
				}
				var back ThemePreferences
				if err := json.Unmarshal(got, &back); err != nil || back != p {
					t.Errorf("%s reads back as %+v, %v", got, back, err)
				}
			}
		}
	}
	if _, err := (ThemePreferences{TextScale: math.NaN()}).appendJSON(nil); err == nil {
		t.Error("a text scale of NaN was written")
	}
}
