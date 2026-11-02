//go:build windows && (amd64 || arm64)

package windows

import "testing"

func TestKeptScripts(t *testing.T) {
	added := func(srcs ...string) []addedScript {
		var a []addedScript
		for _, s := range srcs {
			a = append(a, addedScript{src: s, id: "id" + s})
		}
		return a
	}
	for _, tc := range []struct {
		name  string
		added []addedScript
		srcs  []string
		want  int
	}{
		{"nothing added", nil, []string{"h", "b"}, 0},
		{"same", added("h", "b", "p"), []string{"h", "b", "p"}, 3},
		{"last changed", added("h", "b", "p"), []string{"h", "b", "q"}, 2},
		{"bridge changed", added("h", "b", "p"), []string{"h", "c", "p"}, 1},
		{"first changed", added("h", "b"), []string{"x", "b"}, 0},
		{"fewer", added("h", "b", "p"), []string{"h", "b"}, 2},
		{"more", added("h", "b"), []string{"h", "b", "p", "q"}, 2},
	} {
		if got := keptScripts(tc.added, tc.srcs); got != tc.want {
			t.Errorf("%s: kept %d, want %d", tc.name, got, tc.want)
		}
	}
}
