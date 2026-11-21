package skin

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// HighContrast draws statuses flat (no subtle fills) and every copy of it
// keeps that; Calm and Classic do not; Use hands it to widget, and Use(nil)
// takes it away. It is not a token: no skin file names it.
func TestFlatStatus(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	hc := HighContrast
	for name, s := range map[string]*Skin{
		"HighContrast":              hc,
		"WithAccent":                hc.WithAccent(ui.Hex("#7c3aed")),
		"WithPalette":               hc.WithPalette(Cham),
		"WithDensity":               hc.WithDensity(Compact),
		"WithLooks":                 hc.WithLooks(Looks{Button: "border-2"}),
		"Derive":                    hc.Derive("Brand", hc.Tokens(false), hc.Tokens(true)),
		"WithDensity of WithAccent": hc.WithAccent(ui.Hex("#0f766e")).WithDensity(Compact),
	} {
		Use(s)
		if !skinhook.FlatStatus() {
			t.Errorf("%s: Use leaves FlatStatus off", name)
		}
	}
	for name, s := range map[string]*Skin{"Calm": Calm, "Classic": Classic, "New": New("X", Calm.Tokens(false), Calm.Tokens(true))} {
		Use(s)
		if skinhook.FlatStatus() {
			t.Errorf("%s: FlatStatus is on", name)
		}
	}
	Use(hc)
	Use(nil)
	if skinhook.FlatStatus() {
		t.Error("Use(nil) leaves FlatStatus on")
	}
	if _, ok := reflect.TypeFor[Tokens]().FieldByName("FlatStatus"); ok {
		t.Error("skin.Tokens has a FlatStatus field: it would be a public skin file key")
	}
}
