package skin

import (
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// builtins are the skins MyGo ships, each of which the contrast, sentinel
// and sizes tests run over.
var builtins = []*Skin{Calm, Classic, HighContrast}

// A lazy skin is built once, on its first use, whichever goroutines use it
// first; Use keeps the skin as given for Active, with the built tokens.
func TestLazySkinBuildsOnce(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	var builds atomic.Int32
	s := &Skin{lazy: &lazySkin{build: func() *Skin {
		builds.Add(1)
		return Calm.WithDensity(Compact)
	}}}
	if n := builds.Load(); n != 0 {
		t.Fatalf("built %d times before use, want 0", n)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.Density() != Compact || s.Name() != "Calm" {
				t.Errorf("a lazy skin reads %v %q, want Compact Calm", s.Density(), s.Name())
			}
		}()
	}
	wg.Wait()
	if n := builds.Load(); n != 1 {
		t.Errorf("built %d times, want 1", n)
	}
	Use(s)
	if Active() != s || skinhook.Get(false).ControlHeight != 30 {
		t.Errorf("after Use, Active %p (want %p), ControlHeight %v (want Compact's 30)", Active(), s, skinhook.Get(false).ControlHeight)
	}
}

// The ClassicIndigo palette is filled on its first use, by WithPalette.
func TestLazyPalette(t *testing.T) {
	p := Classic.WithPalette(ClassicSlate).WithPalette(ClassicIndigo)
	if p.Tokens(false).Accent.Base != ui.Hex("#3949ab") || ClassicIndigo.Name() != "Indigo" {
		t.Errorf("Classic with ClassicIndigo has the accent %v, want #3949ab", hexOf(p.Tokens(false).Accent.Base))
	}
}

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
