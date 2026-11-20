package skin

import (
	"slices"
	"testing"
)

// builtins are the skins MyGo ships, each of which the contrast, sentinel
// and sizes tests run over.
var builtins = []*Skin{Calm, Classic, HighContrast}

// The built-in skins are the three, by name.
func TestBuiltins(t *testing.T) {
	var names []string
	for _, s := range builtins {
		names = append(names, s.Name())
	}
	if !slices.Equal(names, []string{"Calm", "Classic", "HighContrast"}) {
		t.Errorf("builtins are %q, want Calm, Classic and HighContrast", names)
	}
}
