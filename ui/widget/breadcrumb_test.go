package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBreadCrumbSelect(t *testing.T) {
	picked := make(chan int, 4)
	b := BreadCrumb("C:", "Users", "ada").OnSelect(func(i int) { picked <- i })
	tt := NewTester(t, b, 500, 100)
	tt.Find(ByLabel("Users").Role(ui.RoleLink)).Click()
	if got := received(t, picked); got != 1 {
		t.Fatalf("a click on Users selected %d, want 1", got)
	}
	// The same crumb again is a navigation again.
	tt.Find(ByLabel("Users").Role(ui.RoleLink)).Click()
	if got := received(t, picked); got != 1 {
		t.Fatalf("a second click on Users selected %d, want 1", got)
	}
}

func TestBreadCrumbSetItems(t *testing.T) {
	b := BreadCrumb("C:", "Users", "ada")
	tt := NewTester(t, b, 500, 100)
	b.SetItems("D:", "Projects")
	tt.Settle()
	if !tt.Has(ByLabel("D:").Role(ui.RoleLink)) || !tt.Has(ByLabel("Projects")) {
		t.Fatalf("the new items do not show: %q", tt.Texts())
	}
	if tt.Has(ByLabel("Users")) {
		t.Fatalf("an old item still shows: %q", tt.Texts())
	}
}

// The items a caller passes are copied: changing its slice later does not
// change the path.
func TestBreadCrumbCopiesItems(t *testing.T) {
	items := []string{"C:", "Users"}
	b := BreadCrumb(items...)
	items[1] = "Changed"
	tt := NewTester(t, b, 500, 100)
	if !tt.Has(ByLabel("Users")) {
		t.Fatalf("the path changed with the caller's slice: %q", tt.Texts())
	}
}

func TestBreadCrumbLastIsCurrent(t *testing.T) {
	tt := NewTester(t, BreadCrumb("C:", "Users", "ada"), 500, 100)
	if c := currents(tt); !slices.Equal(c, []string{"ada"}) {
		t.Fatalf("current %q, want ada", c)
	}
	if tt.Has(ByLabel("ada").Role(ui.RoleLink)) {
		t.Fatal("the last item is a link")
	}
}
