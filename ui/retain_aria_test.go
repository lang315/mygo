package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// DescribedBy, ErrorMessage and Controls point at other elements of the pass.
// A kept row would hold copies that point at elements the pass recycled, and
// share the slices with them: a row that has one is built, and shows to
// assistive technology what the twin that builds every row shows.
func TestRetainRowsWithAriaRelationsAreBuilt(t *testing.T) {
	for _, relation := range []string{"describedBy", "errorMessage", "controls"} {
		t.Run(relation, func(t *testing.T) {
			p := newPair(t, 300, 300, func() func(c *Context) {
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
				return func(c *Context) {
					List(c, s, 28, func(i int) {
						Column(c).Children(func() {
							hint := Textf(c, "hint %d", i)
							item := Textf(c, "item %d", i)
							switch relation {
							case "describedBy":
								item.DescribedBy(hint)
							case "errorMessage":
								item.Invalid(true).ErrorMessage(hint)
							default:
								item.Controls(hint)
							}
						})
					}).Grow(1)
				}
			})
			p.both(func(tt *Tester) { tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn}) })
			same := func(what string) {
				t.Helper()
				view := func(tt *Tester) (out []string) {
					for _, n := range tt.h.access.Nodes {
						out = append(out, fmt.Sprint(n.Label, "|", n.Description, "|", n.Controls))
					}
					return out
				}
				a, b := view(p.kept), view(p.built)
				if len(a) != len(b) {
					t.Fatalf("after %s: the tree has %d nodes kept, %d built", what, len(a), len(b))
				}
				for i := range a {
					if a[i] != b[i] {
						t.Fatalf("after %s: node %d is %q kept, %q built", what, i, a[i], b[i])
					}
				}
			}
			for i := range 6 {
				p.frame(fmt.Sprintf("frame %d", i))
				same(fmt.Sprintf("frame %d", i))
			}
			p.both(func(tt *Tester) { tt.Scroll(50, 100, 0, 80) })
			p.frame("scrolled")
			same("scrolled")
			p.frame("settled")
			same("settled")
		})
	}
}
