package widget

import (
	"testing"

	"github.com/egoist/mygo/internal/l10n"
)

// The wait row of a TreeView says Loading in the app's language, so its
// version follows a change of it: a row kept would keep the old language.
func TestRowVersionTreeViewWaitRowFollowsTheLanguage(t *testing.T) {
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true)
	tt := tester(t, tv)
	x, y := arrow(t, tt, "Alpha")
	tt.ClickAt(x, y)
	tt.Frame()
	wait := -1
	for i := 0; i < tv.state.Rows(); i++ {
		if tv.state.Item(i).wait {
			wait = i
		}
	}
	if wait < 0 {
		t.Fatalf("no wait row in %q", tt.Texts())
	}
	before, text := tv.state.RowVersion(tv.state.Item(wait)), l10n.Loading()
	inLocale(t, "vi")
	if l10n.Loading() == text {
		t.Skip("Loading is the same in vi")
	}
	if after := tv.state.RowVersion(tv.state.Item(wait)); after == before {
		t.Errorf("the wait row has the version %d in two languages", after)
	}
}
