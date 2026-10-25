package platform

import "testing"

// calls records what a composition makes of its client.
type composeClient struct {
	TextInputClient
	marked  bool
	keep    bool
	calls   []string
	replace *TextRange
}

func (c *composeClient) MarkedRange() (TextRange, bool) { return TextRange{Start: 2, End: 4}, c.marked }
func (c *composeClient) KeepMarkedUntilCommit() bool    { return c.keep }
func (c *composeClient) UnmarkText()                    { c.calls = append(c.calls, "unmark"); c.marked = false }
func (c *composeClient) ReplaceText(r *TextRange, s string) {
	c.calls = append(c.calls, "replace "+s)
	c.replace = r
}

func TestInputCompositionEndAndReplaceAreOneCallToTheClient(t *testing.T) {
	c := &composeClient{marked: true, keep: true}
	var order InputComposition
	order.End(c)
	order.Replace(c, nil, "一")
	if len(c.calls) != 1 || c.calls[0] != "replace 一" || c.replace == nil || *c.replace != (TextRange{Start: 2, End: 4}) {
		t.Errorf("calls %v, range %v: want one replace of the marked range", c.calls, c.replace)
	}
}

func TestInputCompositionEndWithoutACommitEndsThePreeditAtOnceOrWhenSomethingElseComes(t *testing.T) {
	c := &composeClient{}
	var order InputComposition
	order.End(c) // nothing marked: nothing waits for a commit
	if len(c.calls) != 1 || c.calls[0] != "unmark" {
		t.Errorf("End without a marked range: calls %v", c.calls)
	}
	c = &composeClient{marked: true, keep: true}
	order = InputComposition{}
	order.End(c)
	if len(c.calls) != 0 {
		t.Errorf("End with a marked range unmarked at once: %v", c.calls)
	}
	order.Reset() // the next preedit, a focus change...
	if len(c.calls) != 1 || c.calls[0] != "unmark" {
		t.Errorf("Reset did not end the preedit as it is: %v", c.calls)
	}
	order.Reset()
	if len(c.calls) != 1 {
		t.Errorf("Reset twice: %v", c.calls)
	}
}

// A client that does not ask for it sees what it always saw: the unmark at End.
func TestInputCompositionEndUnmarksAtOnceForAClientThatDoesNotKeepTheMark(t *testing.T) {
	c := &composeClient{marked: true}
	var order InputComposition
	order.End(c)
	order.Replace(c, nil, "一")
	if len(c.calls) != 2 || c.calls[0] != "unmark" || c.replace == nil {
		t.Errorf("calls %v", c.calls)
	}
	order.Reset()
	if len(c.calls) != 2 {
		t.Errorf("Reset unmarked a preedit that was already: %v", c.calls)
	}
}
