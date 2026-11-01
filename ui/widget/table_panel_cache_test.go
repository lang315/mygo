package widget

import "testing"

// The panel's counts are worked out again only when a count or the language
// changes: a frame with none changed formats nothing (its words go through the
// catalog, which allocates).
func TestPanelCountsAreNotFormattedAgainWhenNothingChanged(t *testing.T) {
	for name, tb := range map[string]*DataGridViewWidget[acct, int]{
		"language":  testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).CheckboxColumn().BottomPanel(),
		"PanelText": testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).CheckboxColumn().BottomPanel().PanelText(PanelText{Rows: "N=%d", Selected: "S=%d", Checked: "C=%d", Search: "Go"}),
	} {
		t.Run(name, func(t *testing.T) {
			tt := tester(t, tb)
			tt.Frame()
			allocs := testing.AllocsPerRun(50, func() {
				mu.Lock()
				tb.counts()
				mu.Unlock()
			})
			if allocs != 0 {
				t.Fatalf("%v allocations to set counts that did not change", allocs)
			}
		})
	}
}

// What changed shows: the counts, and the language.
func TestPanelCountsFollowTheirChanges(t *testing.T) {
	tb := testTable(acct{1, "ann", 1}, acct{2, "bob", 2}).CheckboxColumn().BottomPanel()
	tt := tester(t, tb)
	if !tt.HasText("Rows: 2") || !tt.HasText("Selected: 0") {
		t.Fatalf("texts %q", tt.Texts())
	}
	tb.AddRow(acct{3, "cat", 3})
	tb.SetChecked([]int{1, 2}, true)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Rows: 3") || !tt.HasText("Checked: 2") {
		t.Fatalf("after rows and checks changed: %q", tt.Texts())
	}
	inLocale(t, "vi")
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Số dòng: 3") || !tt.HasText("Đã đánh dấu: 2") {
		t.Fatalf("after the language changed: %q", tt.Texts())
	}
	tb.PanelText(PanelText{Rows: "N=%d", Selected: "S=%d", Checked: "C=%d", Search: "Go"})
	tt.Frame()
	tt.Frame()
	if !tt.HasText("N=3") || !tt.HasText("C=2") || tt.HasText("Số dòng: 3") {
		t.Fatalf("after PanelText: %q", tt.Texts())
	}
}
