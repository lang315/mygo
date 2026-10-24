package widget

import (
	"testing"

	"github.com/egoist/mygo/i18n"
	_ "github.com/egoist/mygo/locales/vi"
)

// inLocale makes the app show tag until the test ends.
func inLocale(t *testing.T, tag string) {
	t.Helper()
	i18n.SetLocaleFunc(func() string { return tag })
	t.Cleanup(func() { i18n.SetLocaleFunc(nil) })
}

func TestDataGridViewPanelSpeaksTheAppsLanguage(t *testing.T) {
	inLocale(t, "vi")
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).BottomPanel()
	tt := tester(t, tb)
	for _, s := range []string{"Số hàng: 2", "Đã chọn: 0", "Tìm"} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
}

func TestDataGridViewPanelTextOverridesTheLanguage(t *testing.T) {
	inLocale(t, "vi")
	tb := testTable(acct{1, "ann", 1}).BottomPanel().PanelText(PanelText{Rows: "N=%d", Selected: "S=%d", Checked: "C=%d", Search: "Go"})
	tt := tester(t, tb)
	if !tt.HasText("N=1") || tt.HasText("Số hàng: 1") {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestTreeViewLoadingRowSpeaksTheAppsLanguage(t *testing.T) {
	inLocale(t, "vi")
	tv := newTree().SetRoots([]tnode{{"A", "Alpha"}}).SetHasChildren("A", true)
	tt := tester(t, tv)
	x, y := arrow(t, tt, "Alpha")
	tt.ClickAt(x, y)
	tt.Frame()
	if !shows(tt, "Đang tải…") {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestColorDialogTitleSpeaksTheAppsLanguage(t *testing.T) {
	inLocale(t, "vi")
	if got := ColorDialog().title; got != "Màu sắc" {
		t.Errorf("title %q", got)
	}
}
