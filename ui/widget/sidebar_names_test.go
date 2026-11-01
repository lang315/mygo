package widget

import "testing"

func TestSidebarEntryIsFoundByLabelOnce(t *testing.T) {
	sb := Sidebar().Add("a", "Mail", Label("inbox")).Add("b", "Cal", Label("today"))
	tt := NewTester(t, sb, 600, 300)
	if n := len(tt.FindAll(ByLabel("Mail"))); n != 1 {
		t.Fatalf("%d elements named Mail", n)
	}
	tt.Find(ByLabel("Cal")).Click()
	if sb.Selected() != "b" || !tt.HasText("today") {
		t.Fatalf("selected %q, texts %q", sb.Selected(), tt.Texts())
	}
}
