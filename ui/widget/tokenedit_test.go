package widget

import (
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// tokenInput returns the input of the TokenEdit tt shows: a combo box when it
// has suggestions, a text field when it has none.
func tokenInput(t *testing.T, tt *Tester) *Node {
	t.Helper()
	if tt.Has(ByRole(ui.RoleComboBox)) {
		return tt.Find(ByRole(ui.RoleComboBox))
	}
	return tt.Find(ByRole(ui.RoleTextField))
}

// counted returns fn for OnChange and the number of times it ran.
func counted() (fn func(), n *int) {
	n = new(int)
	return func() { *n++ }, n
}

func wantTokens(t *testing.T, te *TokenEditWidget, want ...string) {
	t.Helper()
	if got := te.Tokens(); !slices.Equal(got, want) || len(got) != len(want) {
		t.Fatalf("Tokens() = %q, want %q", got, want)
	}
}

// Case 1.
func TestTokenEditStartsEmpty(t *testing.T) {
	te := TokenEdit("apple", "banana", "cherry")
	if got := te.Tokens(); len(got) != 0 {
		t.Fatalf("Tokens() = %q, want none", got)
	}
	NewTester(t, te, 400, 300)
	if got := te.Tokens(); len(got) != 0 {
		t.Fatalf("after a frame, Tokens() = %q, want none", got)
	}
}

// Case 2.
func TestTokenEditSetTokensTrimsAndDrops(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().OnChange(fn)
	te.SetTokens("x", " y ", "", "x")
	wantTokens(t, te, "x", "y")
	tt := NewTester(t, te, 400, 300)
	tt.Settle()
	if !tt.HasText("x") || !tt.HasText("y") {
		t.Fatalf("the form shows %q, want x and y", tt.Texts())
	}
	if tt.HasText(" y ") {
		t.Fatalf("the form shows the untrimmed %q", " y ")
	}
	if *n != 0 {
		t.Fatalf("OnChange ran %d times after SetTokens, want 0", *n)
	}
}

// Case 3.
func TestTokenEditEnterAddsTheText(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	in := tokenInput(t, tt)
	in.Type("foo")
	in.Key(0, ui.KeyEnter)
	wantTokens(t, te, "foo")
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1", *n)
	}
	if v := tokenInput(t, tt).Value(); v != "" {
		t.Fatalf("the input holds %q, want it empty", v)
	}
}

// Case 4.
func TestTokenEditCommaAdds(t *testing.T) {
	te := TokenEdit()
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Type("a,b,")
	wantTokens(t, te, "a", "b")
}

// Case 5.
func TestTokenEditRepeatIsNoChange(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().SetTokens("x").OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	in := tokenInput(t, tt)
	in.Type("x")
	in.Key(0, ui.KeyEnter)
	wantTokens(t, te, "x")
	if *n != 0 {
		t.Fatalf("OnChange ran %d times for a repeat, want 0", *n)
	}
}

// Case 6.
func TestTokenEditBackspaceTakesTheLast(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().SetTokens("x", "y").OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Key(0, ui.KeyBackspace)
	wantTokens(t, te, "x")
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1", *n)
	}
}

// Case 7.
func TestTokenEditRemoveButton(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().SetTokens("x", "y").OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	tt.Find(ByLabel("Remove x")).Click()
	wantTokens(t, te, "y")
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1", *n)
	}
}

// Case 8.
func TestTokenEditSuggestions(t *testing.T) {
	te := TokenEdit("apple", "banana", "cherry")
	tt := NewTester(t, te, 400, 300)
	in := tokenInput(t, tt)
	in.Type("an")
	if !tt.HasText("banana") || tt.HasText("apple") || tt.HasText("cherry") {
		t.Fatalf("an suggests %q, want banana only", tt.Texts())
	}
	// The input keeps the focus; Node.Key would focus it again through
	// assistive technology, which the open list answers.
	tt.Tester.Key(0, ui.KeyDown)
	tt.Settle()
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	wantTokens(t, te, "banana")
}

// Case 9.
func TestTokenEditSetSuggestions(t *testing.T) {
	te := TokenEdit("apple")
	tt := NewTester(t, te, 400, 300)
	te.SetSuggestions("kiwi")
	tt.Settle()
	tokenInput(t, tt).Type("k")
	if !tt.HasText("kiwi") {
		t.Fatalf("k suggests %q, want kiwi", tt.Texts())
	}
}

// Case 10.
func TestTokenEditCopies(t *testing.T) {
	te := TokenEdit()
	s := []string{"a"}
	te.SetTokens(s...)
	s[0] = "z"
	wantTokens(t, te, "a")
	got := te.Tokens()
	got[0] = "z"
	wantTokens(t, te, "a")
}

// Case 10, through ui: what ui does to the slice it shows (append into spare
// room, Delete shifting in place) never reaches the widget's tokens, nor the
// other way.
func TestTokenEditCopiesAcrossUI(t *testing.T) {
	src := make([]string, 2, 8) // spare room an append would write into
	src[0], src[1] = "x", "y"
	te := TokenEdit().SetTokens(src...)
	tt := NewTester(t, te, 400, 300)
	before := te.Tokens()
	tt.Find(ByLabel("Remove x")).Click()
	if !slices.Equal(before, []string{"x", "y"}) {
		t.Fatalf("a slice Tokens returned changed to %q", before)
	}
	wantTokens(t, te, "y")
	in := tokenInput(t, tt)
	in.Type("z")
	in.Key(0, ui.KeyEnter)
	wantTokens(t, te, "y", "z")
	if !slices.Equal(src[:2], []string{"x", "y"}) {
		t.Fatalf("the slice passed to SetTokens changed to %q", src[:2])
	}
}

// Case 10, white-box: ui edits the slice it shows in place (slices.Delete
// shifts and zeroes its array) before it commits; until then the widget's
// tokens are untouched, after a build from a setter as after a commit.
func TestTokenEditShownNeverAliasesTokens(t *testing.T) {
	te := TokenEdit().SetTokens("x", "y", "z")
	tt := NewTester(t, te, 400, 300)
	te.shown = slices.Delete(te.shown, 0, 1) // as ui's Remove x, before its commit
	wantTokens(t, te, "x", "y", "z")
	te.commit() // [y z]
	tt.Settle()
	te.shown = slices.Delete(te.shown, 0, 1) // as ui's Remove y, before its commit
	wantTokens(t, te, "y", "z")
}

// Case 11, the first order: a SetTokens after the user's edit, before the
// next frame, applies on that frame.
func TestTokenEditSetterAfterEditWins(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	in := tokenInput(t, tt)
	in.Type("foo")
	tt.Tester.Key(0, ui.KeyEnter) // no settle: OnChange is posted, not run
	tt.Frame()
	te.SetTokens("bar")
	tt.Settle()
	wantTokens(t, te, "bar")
	if !tt.HasText("bar") || tt.HasText("foo") {
		t.Fatalf("the form shows %q, want bar and no foo", tt.Texts())
	}
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1 (the user's edit)", *n)
	}
}

// Case 11, the second order: a SetTokens that runs after the frame was built
// and before ui commits the user's edit loses to the edit, as mirror.commit
// does. The harness cannot interleave a setter there, so the test does what
// ui does: it changes the shown tokens and calls the commit.
func TestTokenEditEditBeatsASetterOfTheSameFrame(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().SetTokens("x").OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	te.SetTokens("bar")                // after the build that showed [x]
	te.shown = append(te.shown, "foo") // the user adds foo to what ui shows
	te.commit()
	tt.Settle()
	wantTokens(t, te, "x", "foo")
	if !tt.HasText("foo") || tt.HasText("bar") {
		t.Fatalf("the form shows %q, want x and foo, no bar", tt.Texts())
	}
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1", *n)
	}
}

// A commit with nothing changed (ui's OnChange with the same tokens) runs no
// OnChange.
func TestTokenEditCommitWithoutChange(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().SetTokens("x").OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	te.commit()
	tt.Settle()
	if *n != 0 {
		t.Fatalf("OnChange ran %d times for no change, want 0", *n)
	}
	wantTokens(t, te, "x")
}

// placeholder returns the placeholder assistive technology sees on the
// token input, a text field (no suggestions).
func placeholder(t *testing.T, tt *Tester) string {
	t.Helper()
	a, ok := tt.Tester.AccessByRole(ui.RoleTextField, "")
	if !ok {
		t.Fatal("no text field")
	}
	return a.Placeholder
}

// Case 12.
func TestTokenEditPlaceholder(t *testing.T) {
	te := TokenEdit().SetPlaceholder("Add a tag…")
	tt := NewTester(t, te, 400, 300)
	if p := placeholder(t, tt); p != "Add a tag…" {
		t.Fatalf("with no token the placeholder is %q, want Add a tag…", p)
	}
	te.SetTokens("x")
	tt.Settle()
	if p := placeholder(t, tt); p != "" {
		t.Fatalf("with a token the placeholder is %q, want none", p)
	}
	te.SetTokens()
	tt.Settle()
	if p := placeholder(t, tt); p != "Add a tag…" {
		t.Fatalf("with the tokens gone the placeholder is %q, want Add a tag…", p)
	}
}

// Case 12: the placeholder takes the skin's TextPlaceholder, as a TextBox's.
func TestTokenEditPlaceholderTakesTheSkin(t *testing.T) {
	light, _ := useTestSkin(t)
	light.TextPlaceholder = ui.Hex("#ff00aa")
	te := TokenEdit().SetPlaceholder("HHHH HHHH")
	tt := NewTester(t, te, 400, 120)
	tt.Settle()
	if n := countNear(tt.Image(), ui.Rect{}, true, light.TextPlaceholder, 40); n < 10 {
		t.Fatalf("%d pixels of the skin's TextPlaceholder, want 10 or more", n)
	}
}

// The skin's look of fields (KindField) styles a TokenEdit, as a TextBox.
func TestTokenEditTakesTheFieldLook(t *testing.T) {
	pink := ui.Hex("#ff00aa")
	look := NewStyle().Background(Fixed(pink))
	var looks [skinhook.NumKinds]any
	looks[skinhook.KindField] = &look
	skinhook.SetLooks(&looks)
	t.Cleanup(func() { skinhook.SetLooks(nil) })
	tt := NewTester(t, TokenEdit(), 400, 120)
	tt.Settle()
	if n := countNear(tt.Image(), ui.Rect{}, true, pink, 4); n < 500 {
		t.Fatalf("%d pixels of the field look's background, want 500 or more", n)
	}
}

// Case 13. withMac(false): on a Mac menus have the access keys, as the
// other row key tests.
func TestTokenEditRowKeyFocusesTheInput(t *testing.T) {
	withMac(t, false)
	te := TokenEdit("go")
	tt := formOf(t, Fields().Row("&Tags", te))
	altKey(tt, ui.KeyT)
	if !tokenInput(t, tt).Focused() {
		t.Fatal("Alt+T did not focus the token input")
	}
}

// idleAllocations compares an idle frame of a TokenEdit of count tokens with
// one of ui.TokenField of the same tokens: TokenEdit adds nothing.
func idleAllocations(t *testing.T, count int) {
	t.Helper()
	sugg := []string{"a", "b"}
	toks := make([]string, count)
	for i := range toks {
		toks[i] = "t" + strconv.Itoa(i)
	}
	frame := func(tt *Tester) float64 {
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(10, tt.Frame)
	}
	raw := slices.Clone(toks)
	bare := frame(newTester(t, func(c *ui.Context) { ui.TokenField(c, &raw, sugg) }, 400, 300))
	mine := frame(NewTester(t, TokenEdit(sugg...).SetTokens(toks...).SetPlaceholder("p"), 400, 300))
	t.Logf("%d tokens: ui.TokenField %.1f allocations a frame, TokenEdit %.1f", count, bare, mine)
	if mine > bare {
		t.Errorf("%d tokens: TokenEdit allocates %.1f a frame, ui.TokenField %.1f", count, mine, bare)
	}
}

// Case 14.
func TestTokenEditIdleAllocations(t *testing.T) { idleAllocations(t, 3) }

// Case 14 with no token: the placeholder and its skin color cost nothing.
func TestTokenEditIdleAllocationsEmpty(t *testing.T) { idleAllocations(t, 0) }

// Case 15: a paste of several lines adds a token a line.
func TestTokenEditPasteOfLines(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Focus()
	tt.Tester.SetClipboard("a\nb\r\nc\n\nb")
	tt.Tester.Command("paste")
	tt.Settle()
	wantTokens(t, te, "a", "b", "c")
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1", *n)
	}
	if v := tokenInput(t, tt).Value(); v != "" {
		t.Fatalf("the input holds %q, want it empty", v)
	}
}

// Case 15: a lone \r breaks a line too, as ui takes it; a paste of lines that
// are all tokens already is no change.
func TestTokenEditPasteOfCRAndRepeats(t *testing.T) {
	fn, n := counted()
	te := TokenEdit().OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Focus()
	tt.Tester.SetClipboard("a\rb")
	tt.Tester.Command("paste")
	tt.Settle()
	wantTokens(t, te, "a", "b")
	tt.Tester.SetClipboard("b\na\n")
	tt.Tester.Command("paste")
	tt.Settle()
	wantTokens(t, te, "a", "b")
	if *n != 1 {
		t.Fatalf("OnChange ran %d times, want 1 (the second paste adds nothing)", *n)
	}
	tt.Tester.SetClipboard("  c \n\t a\n")
	tt.Tester.Command("paste")
	tt.Settle()
	wantTokens(t, te, "a", "b", "c")
	if v := tokenInput(t, tt).Value(); v != "" {
		t.Fatalf("the input holds %q, want it empty", v)
	}
}

// Case 15: a paste of one line stays text in the input.
func TestTokenEditPasteOfOneLine(t *testing.T) {
	te := TokenEdit()
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Focus()
	tt.Tester.SetClipboard("abc")
	tt.Tester.Command("paste")
	tt.Settle()
	if got := te.Tokens(); len(got) != 0 {
		t.Fatalf("Tokens() = %q, want none", got)
	}
	if v := tokenInput(t, tt).Value(); v != "abc" {
		t.Fatalf("the input holds %q, want abc", v)
	}
}

// Case 16: an idle frame of a TokenEdit allocates no more than ui.TokenField
// of the same tokens, at 50 and 500.
func TestTokenEditManyTokens(t *testing.T) {
	idleAllocations(t, 50)
	idleAllocations(t, 500)
}

// Setters and getters on other goroutines while the user edits: run under
// -race.
func TestTokenEditRace(t *testing.T) {
	te := TokenEdit("apple", "banana").SetTokens("x", "y")
	tt := NewTester(t, te, 400, 300)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_ = te.Tokens()
			if i%50 == 0 {
				te.SetSuggestions("apple", "banana", "cherry")
				te.SetPlaceholder("p")
			}
		}
	}()
	in := tokenInput(t, tt)
	in.Type("foo")
	in.Key(0, ui.KeyEnter)
	tt.Find(ByLabel("Remove x")).Click()
	in.Key(0, ui.KeyBackspace)
	close(stop)
	wg.Wait()

	// SetTokens on another goroutine while the user pastes lines and types.
	stop = make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			if i%20 == 0 {
				te.SetTokens("p", "q")
			}
			_ = te.Tokens()
		}
	}()
	for i := range 5 {
		tt.Tester.SetClipboard("l" + strconv.Itoa(i) + "\nm" + strconv.Itoa(i))
		tt.Tester.Command("paste")
		tt.Settle()
		tt.Tester.Type("t" + strconv.Itoa(i) + ",")
		tt.Settle()
	}
	close(stop)
	wg.Wait()
	te.SetTokens("a")
	tt.Settle()
	wantTokens(t, te, "a")
}
