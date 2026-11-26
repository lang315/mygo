package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func searchTester(t *testing.T, s *SearchBoxWidget) (*Tester, *Node) {
	t.Helper()
	tt := NewTester(t, s, 400, 200)
	return tt, tt.Find(ByRole(ui.RoleTextField))
}

// searchPlaceholder is what assistive technology reads as the search
// field's placeholder.
func searchPlaceholder(t *testing.T, tt *Tester) string {
	t.Helper()
	a, ok := tt.Tester.AccessByRole(ui.RoleTextField, "")
	if !ok {
		t.Fatal("no text field")
	}
	return a.Placeholder
}

func TestSearchBoxTypingFiresOnChange(t *testing.T) {
	changes := make(chan string, 8)
	var s *SearchBoxWidget
	s = SearchBox().OnChange(func() { changes <- s.Text() })
	_, in := searchTester(t, s)
	in.Type("a")
	in.Type("b")
	if got := len(changes); got != 2 {
		t.Fatalf("typing a, b ran OnChange %d times, want 2", got)
	}
	if s.Text() != "ab" {
		t.Fatalf("Text %q, want ab", s.Text())
	}
}

func TestSearchBoxEnterSubmits(t *testing.T) {
	submits := make(chan struct{}, 4)
	s := SearchBox().OnSubmit(func() { submits <- struct{}{} })
	_, in := searchTester(t, s)
	in.Type("go")
	if len(submits) != 0 {
		t.Fatalf("typing submitted %d times", len(submits))
	}
	in.Key(0, ui.KeyEnter)
	if got := len(submits); got != 1 {
		t.Fatalf("Enter ran OnSubmit %d times, want 1", got)
	}
}

func TestSearchBoxPlaceholderShowsWhenEmpty(t *testing.T) {
	s := SearchBox().SetPlaceholder("Find a customer")
	tt, _ := searchTester(t, s)
	if p := searchPlaceholder(t, tt); p != "Find a customer" {
		t.Fatalf("placeholder %q, want Find a customer", p)
	}
}

// Added: the placeholder takes the skin's TextPlaceholder, as a TextBox's.
func TestSearchBoxPlaceholderTakesTheSkin(t *testing.T) {
	light, _ := useTestSkin(t)
	light.TextPlaceholder = ui.Hex("#ff00aa")
	tt := NewTester(t, SearchBox().SetPlaceholder("HHHH HHHH"), 400, 120)
	tt.Settle()
	if n := countNear(tt.Image(), ui.Rect{}, true, light.TextPlaceholder, 40); n < 10 {
		t.Fatalf("%d pixels of the skin's TextPlaceholder, want 10 or more", n)
	}
}

func TestSearchBoxSetterDoesNotFireOnChange(t *testing.T) {
	changes := make(chan struct{}, 4)
	s := SearchBox().OnChange(func() { changes <- struct{}{} })
	tt, in := searchTester(t, s)
	s.SetText("hanoi")
	tt.Settle()
	if in.Value() != "hanoi" {
		t.Fatalf("the field shows %q, want hanoi", in.Value())
	}
	if len(changes) != 0 {
		t.Fatal("SetText ran OnChange")
	}
}

func TestSearchBoxBindsAString(t *testing.T) {
	v := "start"
	s := Bind(SearchBox(), func() string { return v }, func(x string) { v = x })
	_, in := searchTester(t, s)
	if in.Value() != "start" {
		t.Fatalf("the field shows %q, not the source's start", in.Value())
	}
	in.Type("!")
	if v != "start!" {
		t.Fatalf("the source got %q", v)
	}
}
