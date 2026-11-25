package widget

import (
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// tokenSource is a source of tokens, as a config: what get returns, and each
// slice set got, with a log of OnChange and set in the order they ran.
type tokenSource struct {
	mu   sync.Mutex
	src  []string
	sets [][]string
	log  []string
}

func (s *tokenSource) get() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.src
}

func (s *tokenSource) set(v []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.src = v
	s.sets = append(s.sets, v)
	s.log = append(s.log, "set")
}

func (s *tokenSource) onChange() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log = append(s.log, "OnChange")
}

// bound makes a TokenEdit bound to a source holding src.
func bound(src ...string) (*TokenEditWidget, *tokenSource) {
	s := &tokenSource{src: src}
	te := TokenEdit()
	BindTokens(te, s.get, s.set)
	return te, s
}

// wantSets checks set ran once a slice of want, in order.
func wantSets(t *testing.T, s *tokenSource, want ...[]string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sets) != len(want) {
		t.Fatalf("set ran %d times with %q, want %d times with %q", len(s.sets), s.sets, len(want), want)
	}
	for i := range want {
		if !slices.Equal(s.sets[i], want[i]) {
			t.Fatalf("set got %q, want %q", s.sets, want)
		}
	}
}

// typeEnter types text in the token input and presses Enter.
func typeEnter(t *testing.T, tt *Tester, text string) {
	t.Helper()
	in := tokenInput(t, tt)
	in.Type(text)
	in.Key(0, ui.KeyEnter)
	tt.Settle()
}

// Case 1.
func TestBindTokensTakesGet(t *testing.T) {
	te, s := bound("a", "b")
	wantTokens(t, te, "a", "b")
	NewTester(t, te, 400, 300).Settle()
	wantTokens(t, te, "a", "b")
	wantSets(t, s)
}

// Case 2.
func TestBindTokensTrimsAndDrops(t *testing.T) {
	te, s := bound(" a ", "", "a", "b")
	wantTokens(t, te, "a", "b")
	wantSets(t, s)
}

// get returning nil: no tokens.
func TestBindTokensNilSource(t *testing.T) {
	te := TokenEdit().SetTokens("x")
	BindTokens(te, func() []string { return nil }, func([]string) {})
	wantTokens(t, te)
}

// Case 3: Enter adds, set runs once after OnChange.
func TestBindTokensEnterCallsSetAfterOnChange(t *testing.T) {
	te, s := bound("a", "b")
	te.OnChange(s.onChange)
	tt := NewTester(t, te, 400, 300)
	typeEnter(t, tt, "c")
	wantSets(t, s, []string{"a", "b", "c"})
	if !slices.Equal(s.log, []string{"OnChange", "set"}) {
		t.Fatalf("ran %q, want [OnChange set]", s.log)
	}
}

// Case 4, with no OnChange: the binding alone posts.
func TestBindTokensRemoveButton(t *testing.T) {
	te, s := bound("a", "b")
	tt := NewTester(t, te, 400, 300)
	tt.Find(ByLabel("Remove a")).Click()
	tt.Settle()
	wantSets(t, s, []string{"b"})
}

// Case 5.
func TestBindTokensBackspace(t *testing.T) {
	te, s := bound("a", "b")
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Key(0, ui.KeyBackspace)
	tt.Settle()
	wantSets(t, s, []string{"a"})
}

// Case 6.
func TestBindTokensPasteOfLines(t *testing.T) {
	te, s := bound("a", "b")
	tt := NewTester(t, te, 400, 300)
	tokenInput(t, tt).Focus()
	tt.Tester.SetClipboard("x\ny")
	tt.Tester.Command("paste")
	tt.Settle()
	wantSets(t, s, []string{"a", "b", "x", "y"})
}

// Case 7: SetTokens does not call set, before or after a user's edit did.
func TestBindTokensSetTokensDoesNotCallSet(t *testing.T) {
	te, s := bound("a")
	tt := NewTester(t, te, 400, 300)
	te.SetTokens("z")
	tt.Settle()
	wantTokens(t, te, "z")
	wantSets(t, s)
	typeEnter(t, tt, "c")
	wantSets(t, s, []string{"z", "c"})
	te.SetTokens("y")
	tt.Settle()
	wantTokens(t, te, "y")
	wantSets(t, s, []string{"z", "c"})
}

// Case 8.
func TestBindTokensReload(t *testing.T) {
	te, s := bound("a")
	tt := NewTester(t, te, 400, 300)
	s.mu.Lock()
	s.src = []string{"q"}
	s.mu.Unlock()
	Reload(te)
	wantTokens(t, te, "q")
	tt.Settle()
	if !tt.HasText("q") || tt.HasText("a") {
		t.Fatalf("the form shows %q, want q and no a", tt.Texts())
	}
	wantSets(t, s)
}

// Case 9: neither the slice set got nor the one get returned is the widget's.
func TestBindTokensCopies(t *testing.T) {
	src := make([]string, 2, 8) // spare room an append would write into
	src[0], src[1] = "a", "b"
	s := &tokenSource{src: src}
	te := BindTokens(TokenEdit(), s.get, s.set)
	src[0] = "!"
	wantTokens(t, te, "a", "b")
	tt := NewTester(t, te, 400, 300)
	typeEnter(t, tt, "c")
	wantSets(t, s, []string{"a", "b", "c"})
	got := s.sets[0]
	got[0] = "!"
	wantTokens(t, te, "a", "b", "c")
	tt.Find(ByLabel("Remove b")).Click() // ui deletes in place from what it shows
	tt.Settle()
	if !slices.Equal(got, []string{"!", "b", "c"}) {
		t.Fatalf("a slice set got changed to %q", got)
	}
	wantSets(t, s, []string{"!", "b", "c"}, []string{"a", "c"})
}

// Case 10: binding again replaces the binding.
func TestBindTokensRebind(t *testing.T) {
	te, s1 := bound("a")
	s2 := &tokenSource{src: []string{"b"}}
	BindTokens(te, s2.get, s2.set)
	wantTokens(t, te, "b")
	tt := NewTester(t, te, 400, 300)
	typeEnter(t, tt, "c")
	wantSets(t, s1)
	wantSets(t, s2, []string{"b", "c"})
	s2.mu.Lock()
	s2.src = []string{"r"}
	s2.mu.Unlock()
	Reload(te)
	wantTokens(t, te, "r")
}

// Case 11: with SetSuggestionsOnly, a refused entry calls no set.
func TestBindTokensSuggestionsOnly(t *testing.T) {
	s := &tokenSource{}
	te := BindTokens(TokenEdit("Apple").SetSuggestionsOnly(true), s.get, s.set)
	tt := NewTester(t, te, 400, 300)
	typeEnter(t, tt, "kiwi")
	wantSets(t, s)
	tokenInput(t, tt).Type("apple")
	tt.Tester.Key(0, ui.KeyEnter) // the input keeps the focus, the list answers Node.Key's
	tt.Settle()
	wantSets(t, s, []string{"Apple"})
}

// Case 12: a repeat is no change.
func TestBindTokensRepeatCallsSetOnce(t *testing.T) {
	te, s := bound()
	tt := NewTester(t, te, 400, 300)
	typeEnter(t, tt, "x")
	typeEnter(t, tt, "x")
	wantSets(t, s, []string{"x"})
}

// set gets the tokens as the edit committed them, not a SetTokens that ran
// after the commit, before set. The harness cannot interleave a setter there,
// so the test does what ui does: it changes the shown tokens and commits.
func TestBindTokensSetGetsTheCommittedTokens(t *testing.T) {
	te, s := bound("a")
	tt := NewTester(t, te, 400, 300)
	te.shown = append(te.shown, "c") // the user adds c to what ui shows
	te.commit()                      // posts set, not run yet
	te.SetTokens("bar")
	tt.Settle()
	wantSets(t, s, []string{"a", "c"})
	wantTokens(t, te, "bar")
}

// Case 13: Reload and SetTokens on another goroutine while the user edits:
// run under -race.
func TestBindTokensRace(t *testing.T) {
	te, s := bound("x", "y")
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
			if i%20 == 0 {
				te.SetTokens("p", "q")
			}
			if i%30 == 0 {
				Reload(te)
			}
			_ = te.Tokens()
		}
	}()
	in := tokenInput(t, tt)
	for i := range 5 {
		in.Type("t" + strconv.Itoa(i))
		in.Key(0, ui.KeyEnter)
		tt.Tester.SetClipboard("l" + strconv.Itoa(i) + "\nm" + strconv.Itoa(i))
		tt.Tester.Command("paste")
		tt.Settle()
		in.Key(0, ui.KeyBackspace)
		tt.Settle()
	}
	close(stop)
	wg.Wait()
	s.mu.Lock()
	n := len(s.sets)
	s.mu.Unlock()
	if n == 0 {
		t.Fatal("set never ran")
	}
}
