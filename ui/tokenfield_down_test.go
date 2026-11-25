package ui_test

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// options returns the names of the options the popup shows, in order.
func options(tt *ui.Tester) []string {
	var names []string
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleListItem {
			names = append(names, a.Name)
		}
	}
	return names
}

// downField shows a TokenField of *tags with *sugg, Query bound to *query
// when it is not nil, and counts the frames it reports Changed.
func downField(tags, sugg *[]string, query *string, changed *int) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		f := ui.TokenField(c, tags, *sugg).Label("Tags").Width(360)
		if query != nil {
			f.Query(query)
		}
		if f.Changed() {
			*changed++
		}
	}, 500, 400)
}

func wantOptions(t *testing.T, tt *ui.Tester, want ...string) {
	t.Helper()
	if got := options(tt); !slices.Equal(got, want) {
		t.Fatalf("the list shows %q, want %q", got, want)
	}
}

// Down in an empty input opens the list of every suggestion that is not a
// token, in their order, none highlighted: Down then Enter takes the first.
func TestTokenFieldDownListsEverySuggestion(t *testing.T) {
	tags, sugg := []string{"beta"}, []string{"alpha", "beta", "gamma"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	wantOptions(t, tt) // a click shows no list
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt, "alpha", "gamma")
	if a, _ := tt.AccessByRole(ui.RoleListItem, "alpha"); a.Focused {
		t.Fatal("Down opening the list highlights alpha, want none")
	}
	if changed != 0 || !slices.Equal(tags, []string{"beta"}) {
		t.Fatalf("Down: tokens %q, Changed %d times", tags, changed)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if !slices.Equal(tags, []string{"beta", "alpha"}) {
		t.Fatalf("Down, Down, Enter: %q", tags)
	}
	wantOptions(t, tt) // choosing closes the list
}

// Up in an empty input opens no list: only Down does.
func TestTokenFieldUpInEmptyInputShowsNothing(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Key(0, ui.KeyUp)
	wantOptions(t, tt)
}

// Case 9: with no suggestions, Down in the empty input does nothing, and
// suggestions coming later show only as the user asks for them.
func TestTokenFieldDownWithoutSuggestions(t *testing.T) {
	tags, sugg := []string{"x"}, []string(nil)
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt)
	if a, _ := tt.AccessByRole(ui.RoleTextField, "Tags"); a.Expanded {
		t.Fatal("the input says its list shows")
	}
	if changed != 0 || !slices.Equal(tags, []string{"x"}) {
		t.Fatalf("Down: tokens %q, Changed %d times", tags, changed)
	}
	sugg = []string{"alpha"}
	tt.Frame()
	wantOptions(t, tt)
}

// Case 10: with Query bound, Down lists the suggestions as the app gave
// them, leaving out the tokens.
func TestTokenFieldDownWithQuery(t *testing.T) {
	tags, sugg := []string{"y"}, []string{"x", "y"}
	query := ""
	changed := 0
	tt := downField(&tags, &sugg, &query, &changed)
	tt.Click("Tags")
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt, "x")
}

// Case 11: with text typed, Down moves in the list of what matches it.
func TestTokenFieldDownWithText(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma", "delta"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Type("ta")
	wantOptions(t, tt, "delta")
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt, "delta")
	tt.Key(0, ui.KeyEnter)
	if !slices.Equal(tags, []string{"delta"}) {
		t.Fatalf("ta, Down, Enter: %q", tags)
	}
}

// Text typed then erased leaves the input empty with the list open and
// nothing in it: Down lists every suggestion.
func TestTokenFieldDownAfterErasing(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Type("q")
	tt.Key(0, ui.KeyBackspace)
	wantOptions(t, tt)
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt, "alpha", "gamma")
}

// A token added with a comma leaves the input empty: Down lists the
// suggestions left.
func TestTokenFieldDownAfterComma(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Type("alpha,")
	if !slices.Equal(tags, []string{"alpha"}) {
		t.Fatalf("alpha,: %q", tags)
	}
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt, "gamma")
}

// Escape closes the list of every suggestion, which a click then leaves
// closed, and typing then filters.
func TestTokenFieldDownThenEscapeAndType(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEscape)
	wantOptions(t, tt)
	tt.Click("Tags")
	wantOptions(t, tt)
	tt.Type("g")
	wantOptions(t, tt, "gamma")
}

// Typing in the list of every suggestion filters it, and erasing the text
// leaves it empty, as before Down.
func TestTokenFieldDownThenTypeAndErase(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma"}
	changed := 0
	tt := downField(&tags, &sugg, nil, &changed)
	tt.Click("Tags")
	tt.Key(0, ui.KeyDown)
	tt.Type("g")
	wantOptions(t, tt, "gamma")
	tt.Key(0, ui.KeyBackspace)
	wantOptions(t, tt)
}

// Down with text, the list closed, does not list every suggestion: only an
// empty input does. With Query bound, which shows the suggestions as
// given, the text could not hide them.
func TestTokenFieldDownWithTextMatchingNothing(t *testing.T) {
	tags, sugg := []string{}, []string{"alpha", "gamma"}
	query := ""
	changed := 0
	tt := downField(&tags, &sugg, &query, &changed)
	tt.Click("Tags")
	tt.Type("zz")
	tt.Key(0, ui.KeyEscape)
	tt.Key(0, ui.KeyDown)
	wantOptions(t, tt)
}
