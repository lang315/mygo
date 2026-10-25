package ui

import "testing"

// ScrollTo and ScrollIntoView called while the view is built, after the grid,
// scroll the frame being built: nothing else asks for another frame.
func TestGridViewScrollCalledInTheBuildAfterTheGrid(t *testing.T) {
	for _, headed := range []bool{false, true} {
		for _, into := range []bool{false, true} {
			name := "no headers"
			if headed {
				name = "headers"
			}
			if into {
				name += ", ScrollIntoView"
			} else {
				name += ", ScrollTo"
			}
			t.Run(name, func(t *testing.T) {
				s := &GridState{}
				if headed {
					s.Header = func(i int) bool { return i%10 == 0 }
				}
				flag := false
				tt := coreNewTester(func(c *context) {
					coreColumn(c).Fill().Children(func() {
						coreGridView(c, s, 1000, 100, 40, func(i int) {
							if s.Header != nil && s.Header(i) {
								coreTextf(c, "H%d", i)
							} else {
								coreTextf(c, "Item %d", i)
							}
						}).Grow(1)
						if flag {
							if into {
								s.ScrollIntoView(901)
							} else {
								s.ScrollTo(901, Start)
							}
						}
					})
				}, 448, 300)
				tt.Frame()
				tt.Update(func() { flag = true })
				tt.Frame()
				if _, ok := tt.Find("Item 901"); !ok {
					t.Fatalf("the call in the build after the grid did not scroll it: %v", tt.Texts())
				}
			})
		}
	}
}
