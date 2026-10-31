package ui

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/transfer"
)

// stateCount is how many states a Tester's engine keeps after its frames.
func stateCount(tt *Tester) int { return len(tt.rt.states) }

// prune drops the states the last pass did not build, and an idle frame skips its
// walk of them when it counted every state as built. The count must be of states,
// not of builds: a keyed element looks up its place first (a state a build makes,
// and Key then takes back), so a pass can build as many states as the engine
// keeps while one of them was not built. A key that goes while another comes in
// the same pass is that case; the states left are those of a Tester that only
// ever showed the second set.
func TestPruneDropsAStateWhoseKeyWentWhileAnotherCame(t *testing.T) {
	for _, tc := range []struct{ before, after []string }{
		{[]string{"a"}, []string{"b"}},
		{[]string{"a", "b"}, []string{"a", "c"}},
		{[]string{"a", "b", "c"}, []string{"d"}},
	} {
		keys := tc.before
		view := func(c *context) {
			coreColumn(c).Children(func() {
				for _, k := range keys {
					coreBox(c).Key(k).Size(10, 10)
				}
			})
		}
		tt := coreNewTester(view, 100, 100)
		tt.Frame()
		tt.Frame()
		keys = tc.after
		tt.Frame()
		fresh := coreNewTester(view, 100, 100)
		fresh.Frame()
		fresh.Frame()
		if got, want := stateCount(tt), stateCount(fresh); got != want {
			t.Errorf("%v then %v: the engine keeps %d states; a Tester that only showed the second keeps %d", tc.before, tc.after, got, want)
		}
		for _, s := range tt.rt.states {
			if s.seen != tt.rt.frame || s.pass != tt.rt.pass {
				t.Errorf("%v then %v: the state of an element that was not built is kept (born in frame %d)", tc.before, tc.after, s.born)
			}
		}
		// And again, an idle frame after a frame that pruned.
		tt.Frame()
		fresh.Frame()
		if got, want := stateCount(tt), stateCount(fresh); got != want {
			t.Errorf("%v then %v, idle: %d states, want %d", tc.before, tc.after, got, want)
		}
	}
}

// A drop target that goes takes its formats back from the platform: the formats the
// window registers are the defaults and those of the targets the last pass built,
// and a pass that built none registers the defaults again, which an idle frame
// reaches without looking at the states.
func TestDropFormatsFollowTheTargetsOfThePass(t *testing.T) {
	defaults := []transfer.Format{transfer.FileList, transfer.URIList, platform.DragSessionFormat}
	slices.Sort(defaults)
	custom := transfer.Format("application/vnd.mygo.test+json")
	with := append(slices.Clone(defaults), custom)
	slices.Sort(with)
	target := false
	tt := coreNewTester(func(c *context) {
		zone := coreBox(c).Size(100, 100)
		if target {
			coreDropData(zone, transfer.DropOptions{Formats: []transfer.Format{custom}, Operations: transfer.Copy})
		}
	}, 200, 200)
	check := func(what string, want []transfer.Format) {
		t.Helper()
		if !slices.Equal(tt.rt.dropFormats, want) {
			t.Errorf("%s: the window registers %v, want %v", what, tt.rt.dropFormats, want)
		}
	}
	tt.Frame()
	check("no target", defaults)
	target = true
	tt.Frame()
	check("a target", with)
	tt.Frame()
	check("the target again", with)
	target = false
	tt.Frame()
	check("the target went", defaults)
	tt.Frame()
	check("idle", defaults)
	target = true
	tt.Frame()
	check("back", with)
}

// An element that was found by its place and then given a key leaves the state of
// its place behind, which an earlier frame built (unlook puts back its stamp):
// the count must not take it as seen.
func TestPruneDropsTheStateOfAPlaceThatGotAKey(t *testing.T) {
	keyed := false
	view := func(c *context) {
		coreColumn(c).Children(func() {
			b := coreBox(c).Size(10, 10)
			if keyed {
				b.Key("k")
			}
		})
	}
	tt := coreNewTester(view, 100, 100)
	tt.Frame()
	tt.Frame()
	keyed = true
	tt.Frame()
	fresh := coreNewTester(view, 100, 100)
	fresh.Frame()
	fresh.Frame()
	if got, want := stateCount(tt), stateCount(fresh); got != want {
		t.Errorf("the box got a key: the engine keeps %d states; a Tester that only showed it keyed keeps %d", got, want)
	}
	for _, s := range tt.rt.states {
		if s.seen != tt.rt.frame || s.pass != tt.rt.pass {
			t.Errorf("the state of the place the box left is kept (born in frame %d)", s.born)
		}
	}
}

// countedSeen checks the count prune relies on against the states themselves: the
// number the last pass counted is the number of states it stamped as seen.
func countedSeen(rt *engine) string {
	if rt.seenFrame != rt.frame || rt.seenPass != rt.pass {
		return ""
	}
	n := 0
	for _, s := range rt.states {
		if s.seen == rt.frame && s.pass == rt.pass {
			n++
		}
	}
	if n != rt.seenN {
		return fmt.Sprintf("the pass counted %d states as seen, and %d are", rt.seenN, n)
	}
	return ""
}
