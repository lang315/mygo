package widget

import (
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// onlyField shows a TokenEdit of suggestions taking only those, counting
// its OnChange.
func onlyField(t *testing.T, suggestions ...string) (*TokenEditWidget, *Tester, *int) {
	fn, n := counted()
	te := TokenEdit(suggestions...).SetSuggestionsOnly(true).OnChange(fn)
	return te, NewTester(t, te, 400, 300), n
}

func wantRuns(t *testing.T, n *int, want int) {
	t.Helper()
	if *n != want {
		t.Fatalf("OnChange ran %d times, want %d", *n, want)
	}
}

// listed returns the options the open list shows, in order.
func listed(tt *Tester) []string {
	var names []string
	for _, a := range tt.Tester.AccessAll() {
		if a.Role == ui.RoleListItem {
			names = append(names, a.Name)
		}
	}
	return names
}

// pickAt takes the option at i of the list Down opens in the focused, empty
// input: Down opens it with none highlighted, each Down moves one on.
func pickAt(tt *Tester, i int) {
	for range i + 2 {
		tt.Tester.Key(0, ui.KeyDown)
		tt.Settle()
	}
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
}

// Case 1.
func TestTokenEditOnlyTakesASuggestionInItsSpelling(t *testing.T) {
	te, tt, n := onlyField(t, "Apple", "Banana")
	in := tokenInput(t, tt)
	in.Type("apple")
	tt.Tester.Key(0, ui.KeyEnter) // the input keeps the focus, the list answers Node.Key's
	tt.Settle()
	wantTokens(t, te, "Apple")
	wantRuns(t, n, 1)
}

// Case 2.
func TestTokenEditOnlyDropsOtherText(t *testing.T) {
	te, tt, n := onlyField(t, "Apple", "Banana")
	in := tokenInput(t, tt)
	in.Type("kiwi")
	in.Key(0, ui.KeyEnter)
	wantTokens(t, te)
	wantRuns(t, n, 0)
	tt.Frame()
	if tt.Has(ByLabel("Remove kiwi")) || slices.Contains(te.shown, "kiwi") {
		t.Fatalf("kiwi shows as a chip: %q", tt.Texts())
	}
}

// Case 3.
func TestTokenEditOnlyCommas(t *testing.T) {
	te, tt, n := onlyField(t, "Apple", "Banana")
	tokenInput(t, tt).Type("kiwi,banana,")
	wantTokens(t, te, "Banana")
	wantRuns(t, n, 1)
}

// Case 4.
func TestTokenEditNotOnlyTakesAnyText(t *testing.T) {
	for _, call := range []bool{false, true} {
		te := TokenEdit("Apple", "Banana")
		if call {
			te.SetSuggestionsOnly(false)
		}
		tt := NewTester(t, te, 400, 300)
		in := tokenInput(t, tt)
		in.Type("kiwi")
		in.Key(0, ui.KeyEnter)
		wantTokens(t, te, "kiwi")
	}
}

// Case 5.
func TestTokenEditOnlyKeepsTheAppsTokens(t *testing.T) {
	te := TokenEdit("Apple").SetSuggestionsOnly(true).SetTokens("kiwi")
	wantTokens(t, te, "kiwi")
	tt := NewTester(t, te, 400, 300)
	tt.Settle()
	wantTokens(t, te, "kiwi")
	// The user's addition leaves the app's token.
	in := tokenInput(t, tt)
	in.Type("apple,")
	wantTokens(t, te, "kiwi", "Apple")
}

// Case 6.
func TestTokenEditOnlyChecksTheCurrentSuggestions(t *testing.T) {
	te, tt, _ := onlyField(t, "Apple")
	te.SetSuggestions("Kiwi")
	tt.Settle()
	in := tokenInput(t, tt)
	in.Type("kiwi")
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	wantTokens(t, te, "Kiwi")
	in.Type("apple,")
	wantTokens(t, te, "Kiwi")
}

// A token the user added stays when it is no longer a suggestion: only
// what the user adds is checked.
func TestTokenEditOnlyKeepsTokensOfOldSuggestions(t *testing.T) {
	te, tt, _ := onlyField(t, "Apple")
	tokenInput(t, tt).Type("apple,")
	wantTokens(t, te, "Apple")
	te.SetSuggestions("Banana")
	tt.Settle()
	tokenInput(t, tt).Type("banana,")
	wantTokens(t, te, "Apple", "Banana")
}

// Case 7.
func TestTokenEditDownListsTheSuggestionsLeft(t *testing.T) {
	te := TokenEdit("a", "b", "c").SetTokens("b")
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Focus()
	tt.Tester.Key(0, ui.KeyDown)
	tt.Settle()
	if got := listed(tt); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("Down lists %q, want [a c]", got)
	}
	tt.Tester.Key(0, ui.KeyDown)
	tt.Settle()
	tt.Tester.Key(0, ui.KeyEnter)
	tt.Settle()
	wantTokens(t, te, "b", "a")
}

// Case 8.
func TestTokenEditKeepsPickOrder(t *testing.T) {
	te, tt, n := onlyField(t, "a", "b", "c")
	tokenInput(t, tt).Focus()
	pickAt(tt, 2)
	wantTokens(t, te, "c")
	pickAt(tt, 0)
	wantTokens(t, te, "c", "a")
	wantRuns(t, n, 2)
}

// Case 12.
func TestTokenEditOnlyStillRemoves(t *testing.T) {
	fn, n := counted()
	te := TokenEdit("x").SetSuggestionsOnly(true).SetTokens("x", "y", "z").OnChange(fn)
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Key(0, ui.KeyBackspace)
	wantTokens(t, te, "x", "y")
	wantRuns(t, n, 1)
	tt.Find(ByLabel("Remove x")).Click()
	wantTokens(t, te, "y")
	wantRuns(t, n, 2)
}

// The user's adding a suggestion already a token in another case adds
// nothing.
func TestTokenEditOnlyRepeatInAnotherCase(t *testing.T) {
	te, tt, n := onlyField(t, "Apple", "Banana")
	tokenInput(t, tt).Type("apple,APPLE,")
	wantTokens(t, te, "Apple")
	wantRuns(t, n, 1)
	tt.Frame()
	if got := te.shown; !slices.Equal(got, []string{"Apple"}) {
		t.Fatalf("ui shows %q, want [Apple]", got)
	}
}

// A suggestion of the very spelling typed wins over one equal ignoring case.
func TestTokenEditOnlyExactSpellingFirst(t *testing.T) {
	te, tt, _ := onlyField(t, "Apple", "apple")
	tokenInput(t, tt).Type("apple,")
	wantTokens(t, te, "apple")
}

// A paste of lines is what the user adds: filtered too.
func TestTokenEditOnlyFiltersPastes(t *testing.T) {
	te, tt, n := onlyField(t, "Apple", "Banana")
	tokenInput(t, tt).Focus()
	tt.Tester.SetClipboard("apple\nkiwi")
	tt.Tester.Command("paste")
	tt.Settle()
	wantTokens(t, te, "Apple")
	wantRuns(t, n, 1)
}

// A token of the app that is not a suggestion survives the user's edit
// committed after a setter of the same frame, as
// TestTokenEditEditBeatsASetterOfTheSameFrame.
func TestTokenEditOnlyEditBeatsASetterOfTheSameFrame(t *testing.T) {
	te := TokenEdit("Apple").SetSuggestionsOnly(true).SetTokens("kiwi")
	tt := NewTester(t, te, 400, 300)
	te.SetTokens("bar")                  // after the build that showed [kiwi]
	te.shown = append(te.shown, "apple") // the user adds apple to what ui shows
	te.commit()
	tt.Settle()
	wantTokens(t, te, "kiwi", "Apple")
}

// Case 13: run under -race.
func TestTokenEditOnlyRace(t *testing.T) {
	te := TokenEdit("apple", "banana").SetTokens("x")
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
			te.SetSuggestionsOnly(i%2 == 0)
			if i%10 == 0 {
				te.SetSuggestions("apple", "banana", "t"+strconv.Itoa(i%3))
			}
			_ = te.Tokens()
		}
	}()
	// The input keeps the focus; the keys go to it.
	tokenInput(t, tt).Focus()
	for i := range 5 {
		tt.Tester.Type("t" + strconv.Itoa(i) + ",apple,")
		tt.Settle()
		tt.Tester.SetClipboard("banana\nkiwi")
		tt.Tester.Command("paste")
		tt.Settle()
		tt.Tester.Key(0, ui.KeyDown)
		tt.Settle()
		tt.Tester.Key(0, ui.KeyEscape)
		tt.Settle()
		tt.Tester.Key(0, ui.KeyBackspace)
		tt.Settle()
	}
	close(stop)
	wg.Wait()
}

// Case 14: an idle frame with SetSuggestionsOnly allocates no more than
// one without.
func TestTokenEditOnlyIdleAllocations(t *testing.T) {
	frame := func(only bool) float64 {
		te := TokenEdit("a", "b").SetTokens("t0", "t1", "t2").SetSuggestionsOnly(only)
		tt := NewTester(t, te, 400, 300)
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(10, tt.Frame)
	}
	off, on := frame(false), frame(true)
	t.Logf("allocations a frame: %.1f without, %.1f with SetSuggestionsOnly", off, on)
	if on > off {
		t.Errorf("SetSuggestionsOnly: %.1f allocations a frame, %.1f without", on, off)
	}
}
