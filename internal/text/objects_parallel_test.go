package text

import (
	"fmt"
	"sync"
	"testing"
)

// TestObjectsInParallelSystems lays out text with inline objects in several
// Systems at once, as Testers each with its own do (#1730): what an engine
// keeps of the objects of the text it lays out is its own, so each object
// keeps its width, and -race finds nothing shared.
func TestObjectsInParallelSystems(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for g := range 4 {
		wg.Go(func() {
			s := New()
			for i := range 40 {
				w := float32(5 + g*10 + i%3)
				l := s.Layout(Params{Text: fmt.Sprintf("a%d￼", i), Style: Style{Size: 16},
					Spans: EncodeSpans([]Span{{End: 2 + len(fmt.Sprint(i)) - 1}, {End: 3 + len(fmt.Sprint(i)) - 1, ObjectWidth: w, ObjectHeight: 10}})})
				obj := len([]rune(fmt.Sprintf("a%d", i)))
				found := false
				for _, line := range l.Lines {
					for _, gl := range line.Glyphs {
						if gl.Cluster == obj {
							found = true
							if gl.Advance != w {
								errs <- fmt.Errorf("system %d, text %d: the object is %v wide, want %v", g, i, gl.Advance, w)
								return
							}
						}
					}
				}
				if !found {
					errs <- fmt.Errorf("system %d, text %d: no glyph of the object", g, i)
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
