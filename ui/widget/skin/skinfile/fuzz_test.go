package skinfile

import (
	"testing"

	"github.com/egoist/mygo/ui/widget/skin"
)

// FuzzParse: Parse never panics, and a skin it returns survives Format
// with no base.
func FuzzParse(f *testing.F) {
	for _, s := range roundTrips() {
		f.Add(Format(s, skin.Calm))
		f.Add(Format(s, nil))
	}
	for _, c := range errorCases() {
		f.Add([]byte(c.in))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := Parse(data)
		if err != nil {
			if s != nil {
				t.Fatal("a skin with an error")
			}
			return
		}
		out := Format(s, nil)
		s2, err := Parse(out)
		if err != nil {
			t.Fatalf("Format's file does not parse: %v\n%s", err, out)
		}
		if s2.Tokens(false) != s.Tokens(false) || s2.Tokens(true) != s.Tokens(true) {
			t.Fatalf("tokens changed through Format:\n%s", out)
		}
		if s2.Name() != s.Name() || s2.Looks() != s.Looks() || s2.Density() != s.Density() {
			t.Fatalf("name, looks or density changed through Format:\n%s", out)
		}
	})
}
