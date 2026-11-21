package widget

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// Copy, the context menu and Enter work on the rows, not on the places of the
// icon view, which count the headers.
func TestListViewGroupsCopyMenuAndActivateActOnTheRows(t *testing.T) {
	for _, m := range []ViewMode{ViewList, ViewLargeIcon} {
		t.Run(m.String(), func(t *testing.T) {
			activated := make(chan string, 4)
			menuFor := make(chan []string, 4)
			lv := groupedView(m).
				CopyFunc(func(ks []string) string { return strings.Join(ks, ",") }).
				ContextMenuFunc(func(ks []string) []MenuItem {
					menuFor <- ks
					return []MenuItem{{Text: "Open"}}
				}).
				OnActivate(func(k string) { activated <- k })
			tt := tester(t, lv)
			click(t, tt, "Banana") // the place 3, the header berry being 2
			tt.Command("copy")
			if got := tt.Clipboard(); got != "b" {
				t.Fatalf("Copy of Banana copied %q, want \"b\"", got)
			}
			if err := tt.ClickWith(ui.Cmd, "Date"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			tt.Command("copy")
			if got := tt.Clipboard(); got != "b,d" {
				t.Fatalf("Copy of Banana and Date copied %q, want \"b,d\"", got)
			}
			click(t, tt, "Cherry")
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if k := wait.For(t, "Enter to activate", activated); k != "c" {
				t.Fatalf("Enter on Cherry activated %q", k)
			}
			if err := tt.RightClick("Apple"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			<-posted()
			if ks := wait.For(t, "the context menu to be built", menuFor); strings.Join(ks, ",") != "a" {
				t.Fatalf("the context menu of Apple was built for %v, want [a]", ks)
			}
		})
	}
}
