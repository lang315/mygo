//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"testing"
)

// The response buffer takes the size the app's handler announced, up to a
// bound, and a body of another length is still whole.
func TestSchemeResponsePresizes(t *testing.T) {
	for _, tc := range []struct {
		name, length string
		min, max     int
	}{
		{"a megabyte", "1048576", 1 << 20, 1<<20 + 64},
		{"no header", "", 0, 64},
		{"not a number", "lots", 0, 64},
		{"negative", "-5", 0, 64},
		{"zero", "0", 0, 64},
		{"huge", "99999999999", maxPresized, maxPresized + 64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &schemeResponse{}
			h := map[string][]string{}
			if tc.length != "" {
				h["Content-Length"] = []string{tc.length}
			}
			r.Respond(200, h)
			if c := r.body.Cap(); c < tc.min || c > tc.max {
				t.Errorf("capacity %d, want %d..%d", c, tc.min, tc.max)
			}
		})
	}
	t.Run("longer than announced", func(t *testing.T) {
		r := &schemeResponse{}
		r.Respond(200, map[string][]string{"Content-Length": {"10"}})
		want := bytes.Repeat([]byte("x"), 300<<10)
		for p := want; len(p) > 0; {
			n := min(len(p), 256<<10)
			r.Write(p[:n])
			p = p[n:]
		}
		if !bytes.Equal(r.body.Bytes(), want) {
			t.Errorf("the body is %d bytes, want %d", r.body.Len(), len(want))
		}
	})
}
