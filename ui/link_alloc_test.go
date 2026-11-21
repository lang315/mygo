package ui

import (
	"fmt"
	"slices"
	"testing"
)

// Cases of a Link that allocates nothing in a frame:
//  1. one Link at rest allocates as much a frame as a Text in its place.
//  2. 20 Links allocate as much as 20 Texts.
//  3. a click opens the URL once; Enter with the focus does too.
//  4. two Links with their own URLs: each click opens its own.
//  5. a URL that changes between frames: the click opens the new one.
//  6. in a Router page a path is pushed; outside a Router it opens nothing in one.
//  7. an empty URL: the click does nothing.
//  8. hovered, underlined solid; not hovered, not.
//  9. a Link in a RichText (empty label, Children) keeps working.
//
// Case 9 is TestLinkHoverSolidUnderline (inline_test.go); 10 is the suite.

// linkFrameAllocs is the allocations of a frame of a window of n elements
// that build makes, at rest after its first frames.
func linkFrameAllocs(t *testing.T, n int, build func(c *context)) float64 {
	t.Helper()
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			for range n {
				build(c)
			}
		})
	}, 400, 800)
	tt.Frame()
	tt.Frame()
	return allocsPerRun(t, 20, 0, tt.Frame)
}

func TestLinkFrameAllocatesAsMuchAsText(t *testing.T) {
	for _, n := range []int{1, 20} {
		text := linkFrameAllocs(t, n, func(c *context) { coreText(c, "Help") })
		link := linkFrameAllocs(t, n, func(c *context) { coreLink(c, "Help", "https://example.com") })
		if link != text {
			t.Errorf("%d: a frame of links allocates %v, of texts %v", n, link, text)
		}
	}
}

func TestLinkClickOpensItsURLOnce(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreLink(c, "Help", "https://example.com/help")
	}, 200, 100)
	tt.Frame()
	if err := tt.Click("Help"); err != nil {
		t.Fatal(err)
	}
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://example.com/help"}) {
		t.Errorf("a click opened %q", got)
	}
}

func TestLinkEnterOpensItsURL(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreLink(c, "Help", "https://example.com/help").AutoFocus()
	}, 200, 100)
	tt.Frame()
	tt.Key(0, KeyEnter)
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://example.com/help"}) {
		t.Errorf("Enter opened %q", got)
	}
}

func TestLinksKeepTheirOwnURLs(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreLink(c, "One", "https://example.com/one")
			coreLink(c, "Two", "https://example.com/two")
		})
	}, 200, 100)
	tt.Frame()
	tt.Click("Two")
	tt.Click("One")
	want := []string{"https://example.com/two", "https://example.com/one"}
	if got := tt.OpenedURLs(); !slices.Equal(got, want) {
		t.Errorf("opened %q, want %q", got, want)
	}
}

func TestLinkURLChangesBetweenFrames(t *testing.T) {
	n := 1
	tt := coreNewTester(func(c *context) {
		coreLink(c, "Help", fmt.Sprintf("https://example.com/%d", n))
	}, 200, 100)
	tt.Frame()
	n = 2
	tt.Frame()
	tt.Click("Help")
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://example.com/2"}) {
		t.Errorf("opened %q, want the new URL", got)
	}
}

func TestLinkPathGoesToItsRouterOnly(t *testing.T) {
	r := NewRouter("/")
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			r.coreView(c, func(rt *Route) {
				if rt.Match("/") {
					coreLink(c, "Inside", "/notes/42")
				} else {
					coreText(c, "Note")
				}
			})
			coreLink(c, "Outside", "/notes/7")
		})
	}, 300, 200)
	tt.Frame()
	tt.Click("Outside")
	if r.Location() != "/" {
		t.Errorf("a link outside the router went to %q", r.Location())
	}
	tt.Click("Inside")
	if r.Location() != "/notes/42" {
		t.Errorf("a link inside went to %q, want /notes/42", r.Location())
	}
}

func TestLinkWithoutURLDoesNothing(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreLink(c, "Help", "")
	}, 200, 100)
	tt.Frame()
	tt.Click("Help")
	if got := tt.OpenedURLs(); len(got) != 0 {
		t.Errorf("opened %q", got)
	}
}

func TestLinkUnderlineWhileHovered(t *testing.T) {
	var link *node
	tt := coreNewTester(func(c *context) {
		link = coreLink(c, "Help", "https://example.com")
	}, 200, 100)
	tt.Frame()
	if link.ts.solid || link.ts.underline {
		t.Errorf("at rest: %+v", link.ts)
	}
	r, _ := tt.Find("Help")
	tt.Move(center(r))
	if !link.ts.solid || !link.ts.underline {
		t.Errorf("hovered: %+v", link.ts)
	}
	tt.Move(199, 99)
	if link.ts.solid || link.ts.underline {
		t.Errorf("after the pointer left: %+v", link.ts)
	}
}
