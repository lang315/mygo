package editor

import (
	"strings"
	"testing"
)

// A find bar opened again brings its match into view in a frame whose pass
// is built again (the query taking the focus), and asks no more after that
// frame: the user scrolls away from it, and it stays where the user put it.
// (The fork builds a pass again when the focus moves, which lost a reveal
// asked in the first pass alone: buildReveal, paint.)
func TestFitContentRevealOnceAcrossPasses(t *testing.T) {
	st, scroll, tt := fitEditor(t, strings.Repeat("line\n", 300)+"needle\n")
	st.Find("needle")
	tt.Frame()
	tt.Frame()
	st.ed.find.open = false
	scroll.Y = 0
	tt.Frame()
	st.ed.find.show(&st.ed, false)
	tt.Frame()
	if scroll.Y == 0 {
		t.Fatal("the bar opened without bringing its match into view in its frame")
	}
	if st.ed.revealAt != -1 || st.ed.revealBuilt {
		t.Fatalf("the reveal is still asked after the frame painted: at %d, built %v", st.ed.revealAt, st.ed.revealBuilt)
	}
	scroll.Y = 0
	tt.Frame()
	tt.Frame()
	if scroll.Y != 0 {
		t.Fatalf("the view scrolled to %v on its own after the user scrolled away", scroll.Y)
	}
}
